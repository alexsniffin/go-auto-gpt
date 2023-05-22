package actor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/openai"
	langChainPrompts "github.com/tmc/langchaingo/prompts"
	searchActor "go-autogpt/internal/agents/search/actor"
	"go-autogpt/internal/agents/supervisor/handler"
	terminalActor "go-autogpt/internal/agents/terminal/actor"
	"go-autogpt/pkg/data"
	"go-autogpt/pkg/logger"
	"go-autogpt/pkg/memory/buffer"
	"go-autogpt/pkg/messages"
	"go-autogpt/pkg/models"
	"go-autogpt/pkg/prompts"
	"go-autogpt/pkg/tools"
	"time"
)

type Supervisor struct {
	handler    *handler.Handler
	id         uuid.UUID
	goal       string
	tasksQueue []string
	history    []models.TaskHistory
	memory     buffer.Memories // todo remove when langchaingo supports
	state      models.State
}

var (
	TaskPrompt = langChainPrompts.NewPromptTemplate(prompts.TaskTemplate, []string{"Goal", "Task", "History"})
)

func New() actor.Actor {
	llm, _ := openai.New() // todo err
	chain := chains.NewLLMChain(llm, TaskPrompt)
	return &Supervisor{
		handler:    handler.New(chain),
		id:         uuid.Nil,
		tasksQueue: make([]string, 0),
		history:    make([]models.TaskHistory, 0),
		memory:     buffer.Memories{Items: make([]buffer.Memory, 0)},
		state:      models.Init,
	}
}

func (agent *Supervisor) Receive(ac actor.Context) {
	l := log.With().Fields(map[string]interface{}{logger.ActorIDField: ac.Self().GetId(), logger.AgentNameField: "supervisor"}).Logger()
	switch msg := ac.Message().(type) {
	case *actor.Started:
		l.Debug().Msg("starting actor")
	case *actor.Stopping:
		l.Debug().Msg("stopping actor")
	case *actor.Stopped:
		l.Debug().Msg("stopped actor and its children")
	case *actor.Restarting:
		l.Debug().Msg("restarting actor")
	case *actor.Terminated:
		l.Debug().Msg("child actor terminated")
	case messages.NewPlan: // from planner
		l.Debug().Str(logger.RequestTaskID, msg.RequestID.String()).Msgf("NewPlan received from planner agent: %v", msg)
		agent.goal = msg.Goal
		agent.id = msg.RequestID
		agent.tasksQueue = append(agent.tasksQueue, msg.Tasks...)
		agent.Next(ac, msg)
	case messages.SearchResult: // from search actor
		l.Debug().Str(logger.RequestTaskID, agent.id.String()).Msgf("SearchResult received from search agent: %v", msg)
		agent.history[len(agent.history)-1].Result = msg.Result
		if finish := agent.reportTaskToParent(ac, agent.history[len(agent.history)-1]); finish {
			return
		}
		agent.Next(ac, msg)
	case messages.CommandResult:
		l.Debug().Str(logger.RequestTaskID, agent.id.String()).Msgf("CommandResult received from terminal agent: %v", msg)
		agent.history[len(agent.history)-1].Result = msg
		if finish := agent.reportTaskToParent(ac, agent.history[len(agent.history)-1]); finish {
			return
		}
		agent.Next(ac, msg)
	case messages.ReportError:
		l.Debug().Str(logger.RequestTaskID, agent.id.String()).Msgf("ReportError received from child agent: %v", msg)
		agent.reportErrorToParent(ac, msg.Error)
		return
	default:
		l.Warn().Str(logger.RequestTaskID, agent.id.String()).Msgf("unknown message: %v", msg)
	}
	agent.state = models.Idle
}

func (agent *Supervisor) Next(ac actor.Context, msg interface{}) {
	l := log.With().Fields(map[string]interface{}{logger.ActorIDField: ac.Self().GetId(), logger.AgentNameField: "supervisor"}).Logger()
	agent.state = models.Thinking
	task := agent.tasksQueue[0]
	agent.tasksQueue = agent.tasksQueue[1:] // pop

	l.Info().Str(logger.TaskField, task).Msg("grabbing next task off the queue...")
	l.Info().Str(logger.TaskField, task).Msg("thinking about a solution for the task...")
	hRes := agent.handler.Solution(context.Background(), task, agent.goal, agent.marshalHistory())
	if hRes.Error != nil {
		t := time.Now()
		agent.reportErrorToParent(ac, models.Error{ErrMessage: hRes.Error.Error(), Time: &t, Message: msg})
		return
	}
	agent.memory.Add(buffer.Memory{
		Question: hRes.Question,
		Answer:   hRes.Answer,
	})

	match, err := data.SanitizeAnswer(hRes.Answer)
	if err != nil {
		t := time.Now()
		agent.reportErrorToParent(ac, models.Error{ErrMessage: err.Error(), Message: msg, Time: &t})
		return
	}

	ans, err := parseAnswer(match)
	if err != nil {
		t := time.Now()
		agent.reportErrorToParent(ac, models.Error{ErrMessage: err.Error(), Time: &t, Message: msg})
		return
	}

	l.Info().Str(logger.TaskField, task).Msgf("solution determined, using %s to solve the task...", ans.Tool)
	switch tools.Tool(ans.Tool) {
	case tools.Search: // todo impl
		props := actor.PropsFromProducer(searchActor.New)
		child := ac.Spawn(props)
		ac.Send(child, messages.NewSearch{Search: ans.Inputs[0], ExpectedOutcome: ans.Outcome, PossibleLimitations: ans.Limitations})
		agent.history = append(agent.history, models.TaskHistory{Task: task, Solution: ans})
		return
	case tools.Terminal:
		props := actor.PropsFromProducer(terminalActor.New)
		child := ac.Spawn(props)
		ac.Send(child, messages.ExecuteCommand{RequestID: agent.id, Command: ans.Inputs[0], Reason: ans.Reasoning, Task: task}) // todo dont assume one input
		agent.history = append(agent.history, models.TaskHistory{Task: task, Solution: ans})
		return
	default:
		l.Error().Msgf("unknown tool: %v", ans.Tool)
		t := time.Now()
		agent.reportErrorToParent(ac, models.Error{ErrMessage: "unknown tool when determining solution from task", Message: msg, Time: &t})
		return
	}
}

func (agent *Supervisor) marshalHistory() string {
	res, _ := json.Marshal(agent.history) // todo err
	return string(res)
}

func (agent *Supervisor) reportTaskToParent(ac actor.Context, task models.TaskHistory) bool {
	log.Info().Msg("reporting completed task to parent...")
	if len(agent.tasksQueue) == 0 {
		log.Info().Msg("we have completed all the tasks in our queue, report the results back to the user!")
		agent.state = models.Finished
		ac.Send(ac.Parent(), messages.TaskResult{TaskHistory: task})
		ac.Send(ac.Parent(), messages.SupervisorComplete{Result: task.Result})
		ac.Stop(ac.Self())
		return true
	} else {
		ac.Send(ac.Parent(), messages.TaskResult{TaskHistory: task})
		return false
	}
}

func (agent *Supervisor) reportErrorToParent(ac actor.Context, err models.Error) {
	agent.state = models.Failed
	log.Error().Err(errors.New(err.ErrMessage)).Msg("reporting error to parent...")
	ac.Send(ac.Parent(), messages.ReportError{Error: err})
	ac.Stop(ac.Self())
}

func parseAnswer(answer string) (models.Solution, error) {
	ba := []byte(answer)
	res := models.Solution{}
	err := json.Unmarshal(ba, &res)
	if err != nil {
		return models.Solution{}, fmt.Errorf("unmarshal: %w", err)
	}
	return res, nil
}
