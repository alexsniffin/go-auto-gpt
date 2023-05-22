package actor

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/rs/zerolog/log"
	"go-autogpt/internal/agents/search/handler"
	"go-autogpt/pkg/logger"
	"go-autogpt/pkg/memory/buffer"
	"go-autogpt/pkg/messages"
	"go-autogpt/pkg/models"
)

type Search struct {
	handler *handler.Handler
	memory  buffer.Memories // todo remove when langchaingo supports
	state   models.State
	errs    []models.Error
}

func New() actor.Actor {
	return &Search{
		handler: handler.New(),
		memory:  buffer.Memories{Items: make([]buffer.Memory, 0)},
		state:   models.Init,
		errs:    make([]models.Error, 0),
	}
}

func (agent *Search) Receive(ac actor.Context) {
	l := log.With().Fields(map[string]interface{}{logger.ActorIDField: ac.Self().GetId(), logger.AgentNameField: "search"}).Logger()
	switch msg := ac.Message().(type) {
	case *actor.Started:
		l.Debug().Msg("starting actor")
	case *actor.Stopping:
		l.Debug().Msg("stopping actor")
	case *actor.Stopped:
		l.Debug().Msg("stopped actor and its children")
	case *actor.Restarting:
		l.Debug().Msg("restarting actor")
	case messages.NewSearch:
		l.Info().Msgf("NewSearch received: %v", msg.Search)
		ac.Send(ac.Parent(), messages.SearchResult{Result: "the result!"}) // todo impl real search
	default:
		l.Warn().Msgf("unknown message: %v", msg)
	}
	agent.state = models.Idle
}
