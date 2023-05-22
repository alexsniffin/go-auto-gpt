package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/openai"
	langChainPrompt "github.com/tmc/langchaingo/prompts"
	"go-autogpt/internal/agents/planner/handler"
	supervisor "go-autogpt/internal/agents/supervisor/actor"
	"go-autogpt/pkg/data"
	"go-autogpt/pkg/logger"
	"go-autogpt/pkg/memory/buffer"
	"go-autogpt/pkg/messages"
	"go-autogpt/pkg/models"
	"go-autogpt/pkg/prompts"
	"time"
)

type Planner struct {
	id        uuid.UUID
	handler   *handler.Handler
	memory    buffer.Memories // todo remove when langchaingo supports
	state     models.State
	err       models.Error
	history   []models.TaskHistory // todo store the complete state at the api (with durable storage eventually)???
	completed bool
}

var (
	NewActionPrompt = langChainPrompt.NewPromptTemplate(prompts.PlannerNewAction, []string{"Goal"})
)

func New() actor.Actor {
	llm, _ := openai.New() // todo err
	chain := chains.NewLLMChain(llm, NewActionPrompt)
	return &Planner{
		id:        uuid.Nil,
		handler:   handler.New(chain),
		memory:    buffer.Memories{Items: make([]buffer.Memory, 0)},
		state:     models.Init,
		history:   make([]models.TaskHistory, 0),
		completed: false,
	}
}

func (agent *Planner) Receive(ac actor.Context) {
	l := log.With().Fields(map[string]interface{}{logger.ActorIDField: ac.Self().GetId(), logger.AgentNameField: "planner"}).Logger()
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
	case messages.GetStatus:
		l.Debug().Msg("GetStatus message received from user")

		match, err := data.SanitizeAnswer(agent.memory.Items[0].Answer) // todo restructure
		if err != nil {
			l.Error().Err(err).Msg("error unmarshalling json")
			ac.Respond(err)
			return
		}
		var ans map[string][]string
		err = json.Unmarshal([]byte(match), &ans)
		if err != nil {
			l.Error().Err(err).Msg("error unmarshalling json")
			ac.Respond(err)
			return
		}
		if agent.completed {
			agent.state = models.Finished
			l.Info().Msg("Work complete!")
			// ac.Stop(ac.Self())
			// todo need to store the state in the api before stopping the actor
		}
		ac.Respond(models.Status{
			Planner: models.Agent{
				State:       agent.state,
				Plan:        ans,
				TaskHistory: agent.history,
				Errs:        agent.err,
			},
		})
		return
	case messages.NewGoal:
		l.Debug().Str(logger.RequestTaskID, msg.RequestID.String()).Msgf("NewGoal received from user: %v", msg)
		agent.state = models.Thinking
		agent.id = msg.RequestID

		l.Info().Str(logger.RequestTaskID, agent.id.String()).Msg("planning...")
		hRes := agent.handler.Plan(context.Background(), msg) // todo timeout
		if hRes.Error != nil {
			t := time.Now()
			agent.err = models.Error{Err: hRes.Error, Message: msg, Time: &t}
			agent.state = models.Failed
			return
		}
		agent.memory.Add(buffer.Memory{
			Question: hRes.Question,
			Answer:   hRes.Answer,
		})

		match, err := data.SanitizeAnswer(hRes.Answer)
		if err != nil {
			t := time.Now()
			agent.err = models.Error{Err: err, Message: msg, Time: &t}
			agent.state = models.Failed
			return
		}

		tasks, err := parseAnswer(match)
		if err != nil {
			t := time.Now()
			agent.err = models.Error{Err: hRes.Error, Message: msg, Time: &t}
			agent.state = models.Failed
			l.Error().Err(err).Str(logger.RequestTaskID, agent.id.String()).Msg("unable to parse answer from plan")
			return
		}
		if len(tasks) == 0 {
			t := time.Now()
			agent.completed = true
			agent.err = models.Error{Message: "unable to build a plan from the goal", Time: &t}
			return
		}

		props := actor.PropsFromProducer(supervisor.New)
		child := ac.Spawn(props)
		l.Info().Str(logger.RequestTaskID, agent.id.String()).Msg("sending plan to supervisor...")
		ac.Send(child, messages.NewPlan{RequestID: agent.id, Plan: models.Plan{
			Goal:  msg.Goal,
			Tasks: tasks,
		}})
	case messages.TaskResult:
		l.Debug().Str(logger.RequestTaskID, agent.id.String()).Msgf("TaskResult received from supervisor agent: %v", msg)
		agent.history = append(agent.history, msg.TaskHistory)
	case messages.SupervisorComplete:
		l.Debug().Str(logger.RequestTaskID, agent.id.String()).Msgf("SupervisorComplete received from supervisor agent: %v", msg)
		agent.completed = true
		agent.state = models.Finished
	case messages.ReportError:
		l.Debug().Str(logger.RequestTaskID, agent.id.String()).Msgf("ReportError received from supervisor agent: %v", msg)
		agent.completed = true
		agent.state = models.Failed
		agent.err = msg.Error
	default:
		l.Warn().Str(logger.RequestTaskID, agent.id.String()).Msgf("unknown message: %v", msg)
	}
	agent.state = models.Idle
}

func parseAnswer(answer string) ([]string, error) {
	ba := []byte(answer)
	res := map[string][]string{}
	err := json.Unmarshal(ba, &res)
	if err != nil {
		return []string{}, fmt.Errorf("unmarshal: %w", err)
	}
	return res["tasks"], nil
}
