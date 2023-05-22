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
	"go-autogpt/internal/agents/terminal/handler"
	agentModel "go-autogpt/internal/agents/terminal/models"
	"go-autogpt/pkg/data"
	"go-autogpt/pkg/logger"
	"go-autogpt/pkg/memory/buffer"
	"go-autogpt/pkg/messages"
	"go-autogpt/pkg/models"
	"go-autogpt/pkg/prompts"
	"time"
)

type Terminal struct {
	handler     *handler.Handler
	id          uuid.UUID
	memory      buffer.Memories // todo remove when langchaingo supports
	state       models.State
	maxAttempts int
}

var (
	TerminalDiagnoseErrorPrompt = langChainPrompts.NewPromptTemplate(prompts.CommandDiagnoseTemplate, []string{"PreviousAttempts", "Task"})
)

func New() actor.Actor {
	llm, _ := openai.New() // todo err
	chain := chains.NewLLMChain(llm, TerminalDiagnoseErrorPrompt)
	return &Terminal{
		handler: handler.New(chain),
		id:      uuid.Nil,
		memory:  buffer.Memories{Items: make([]buffer.Memory, 0)},
		state:   models.Init,
		// to prevent infinite loop
		maxAttempts: 5, // todo add as a config
	}
}

func (agent *Terminal) Receive(ac actor.Context) {
	l := log.With().Fields(map[string]interface{}{logger.ActorIDField: ac.Self().GetId(), logger.AgentNameField: "terminal"}).Logger()
	switch msg := ac.Message().(type) {
	case *actor.Started:
		l.Debug().Msg("starting actor")
	case *actor.Stopping:
		l.Debug().Msg("stopping actor")
	case *actor.Stopped:
		l.Debug().Msg("stopped actor and its children")
	case *actor.Restarting:
		l.Debug().Msg("restarting actor")
	case messages.ExecuteCommand:
		l.Debug().Msgf("ExecuteCommand received: %v", msg)
		agent.state = models.Thinking
		if msg.RequestID != uuid.Nil {
			agent.id = msg.RequestID
		}

		err := agent.handler.CreateDirectoryIfNotExists(agent.id.String())
		if err != nil {
			t := time.Now()
			agent.reportErrorToParent(ac, models.Error{ErrMessage: err.Error(), Message: msg, Time: &t})
			return
		}

		l.Info().Msgf("attempting to run command: %v", msg.Command)
		out, err := agent.handler.RunCommand(context.Background(), msg.Command, agent.id.String()) // todo timeout
		if err != nil {
			agent.state = models.Failed
			l.Error().Err(err).Msgf("command failed: %v", msg.Command)
			previous := append(msg.PreviousAttempts, messages.CommandAttempt{
				Command: msg.Command,
				Error:   err.Error(),
				Reason:  msg.Reason,
			})
			ac.Send(ac.Self(), messages.DiagnoseCommand{PreviousAttempts: previous, Task: msg.Task})
			return
		}

		l.Info().Msgf("command succeeded with output: %v", out)
		ac.Send(ac.Parent(), messages.CommandResult{Result: out, DiagnosticAttempts: msg.PreviousAttempts})
		ac.Stop(ac.Self())
	case messages.DiagnoseCommand:
		// todo this should honestly use sub-prompts to determine what is available to help determine the next step
		// it currently will attempt to brute force rather than intelligently diagnose
		l.Debug().Msgf("DiagnoseCommand received: %v", msg)
		agent.state = models.Thinking
		if agent.maxAttempts <= 0 {
			t := time.Now()
			l.Error().Msg("maxAttempts exceeded for terminal agent")
			agent.reportErrorToParent(ac, models.Error{ErrMessage: "maxAttempts exceeded for terminal agent", Message: msg, Time: &t})
			return
		}
		agent.maxAttempts--

		l.Info().Msg("diagnosing problem from previous command...")
		previousAttempts := agent.marshalPreviousAttempts(msg.PreviousAttempts)
		hRes := agent.handler.DiagnoseNextAttempt(context.Background(), msg.Task, previousAttempts)
		if hRes.Error != nil {
			t := time.Now()
			agent.reportErrorToParent(ac, models.Error{ErrMessage: hRes.Error.Error(), Message: msg, Time: &t})
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

		diagnose, err := parseDiagnose(match)
		if err != nil {
			t := time.Now()
			agent.reportErrorToParent(ac, models.Error{ErrMessage: err.Error(), Message: msg, Time: &t})
			return
		}

		l.Info().Msgf("new solution determined, I should run the command: %s because %s...", diagnose.Command, diagnose.Reason)
		out, err := agent.handler.RunCommand(context.Background(), diagnose.Command, agent.id.String())
		if err != nil {
			agent.state = models.Failed
			l.Error().Err(err).Msgf("command failed again: %v", diagnose.Command)
			previous := append(msg.PreviousAttempts, messages.CommandAttempt{
				Command: diagnose.Command,
				Error:   err.Error(),
				Reason:  diagnose.Reason,
			})
			ac.Send(ac.Self(), messages.DiagnoseCommand{PreviousAttempts: previous, Task: msg.Task})
			return
		}

		previous := append(msg.PreviousAttempts, messages.CommandAttempt{
			Command: diagnose.Command,
			Reason:  diagnose.Reason,
			Output:  out,
		})

		l.Info().Msg("command succeeded, I should try the original command now...")
		ac.Send(ac.Self(), messages.ExecuteCommand{Command: msg.PreviousAttempts[0].Command, Task: msg.Task, Reason: msg.PreviousAttempts[0].Reason, PreviousAttempts: previous})
	default:
		l.Warn().Str(logger.RequestTaskID, agent.id.String()).Msgf("unknown message: %v", msg)
	}
	agent.state = models.Idle
}

func (agent *Terminal) marshalPreviousAttempts(previousAttempts []messages.CommandAttempt) string {
	res, _ := json.Marshal(previousAttempts) // todo err
	return string(res)
}

func (agent *Terminal) reportErrorToParent(ac actor.Context, err models.Error) {
	agent.state = models.Failed
	log.Error().Err(errors.New(err.ErrMessage)).Msg("reporting error to parent...")
	ac.Send(ac.Parent(), messages.ReportError{Error: err})
	ac.Stop(ac.Self())
}

func parseDiagnose(answer string) (agentModel.Diagnose, error) {
	ba := []byte(answer)
	res := agentModel.Diagnose{}
	err := json.Unmarshal(ba, &res)
	if err != nil {
		return agentModel.Diagnose{}, fmt.Errorf("unmarshal: %w", err)
	}
	return res, nil
}
