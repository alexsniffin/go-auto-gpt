package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asynkron/protoactor-go/actor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/justinas/alice"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	planner "go-autogpt/internal/agents/planner/actor"
	"go-autogpt/pkg/logger"
	"go-autogpt/pkg/messages"
	"go-autogpt/pkg/models"
	"io"
	"net/http"
	"time"
)

type command struct {
	Goal string `json:"goal"`
}

type getStatus struct {
	Status models.Status `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type Server struct {
	ac     *actor.RootContext
	server *http.Server
	state  requestsCache
}

func New(ac *actor.RootContext) *Server {
	r := chi.NewRouter()
	r.Use(logMiddleware())
	requests := newRequestsCache()

	r.Get("/status/{id}", func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Msg("status request")
		idParam := chi.URLParam(r, "id")
		id, err := uuid.Parse(idParam)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Debug().Msg("cannot parse id")
			render.JSON(w, r, errorResponse{Error: "unable to parse id"})
			return
		}
		pid, ok := requests.get(id)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			log.Debug().Str(logger.RequestTaskID, idParam).Msg("cannot find id")
			return
		}

		future := ac.RequestFuture(pid, messages.GetStatus{}, time.Minute) // blocking
		res, err := future.Result()
		if err != nil {
			requests.remove(id)
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Str(logger.RequestTaskID, idParam).Err(err).Msg("unable to get status from actor")
			return
		}
		if err, ok := res.(error); ok {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Str(logger.RequestTaskID, idParam).Err(err).Msg("unable to get status from actor")
			return
		}

		if status, ok := res.(models.Status); ok {
			render.JSON(w, r, getStatus{status})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Str(logger.RequestTaskID, idParam).Err(err).Msg("unknown status from actor")
		}
	})

	r.Post("/new", func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Msg("new request")
		cmd := command{}
		err := unmarshalRequestBody(r, &cmd)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Debug().Msg("cannot parse body")
			render.JSON(w, r, errorResponse{Error: "unable to parse body"})
			return
		}

		decider := func(reason interface{}) actor.Directive {
			log.Error().Msgf("handling failure for child. reason: %v", reason)
			return actor.RestartDirective
		}

		strategy := actor.NewOneForOneStrategy(3, 10000, decider)

		// todo allow for the configuration of remote actors
		props := actor.PropsFromProducer(planner.New, actor.WithSupervisor(strategy))
		pid := ac.Spawn(props)

		id := uuid.New()
		ac.Send(pid, messages.NewGoal{RequestID: id, Goal: cmd.Goal})
		requests.add(id, pid)

		log.Debug().Str(logger.RequestTaskID, id.String()).Msg("agent job has been started")
		render.JSON(w, r, struct {
			Id string `json:"id"`
		}{id.String()})
	})

	return &Server{
		ac: ac,
		server: &http.Server{
			Addr:    fmt.Sprint(":", 8080), // todo use config
			Handler: r,
		},
	}
}

func (s *Server) Start() error {
	err := s.server.ListenAndServe()
	if err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}

	log.Info().Msg("http server started")
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	err := s.server.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("http server: %w", err)
	}

	return nil
}

func logMiddleware() func(http.Handler) http.Handler {
	c := alice.New()
	c = c.Append(hlog.NewHandler(log.Logger))
	c = c.Append(hlog.RemoteAddrHandler("ip"))
	c = c.Append(hlog.UserAgentHandler("agent"))
	c = c.Append(hlog.RefererHandler("referer"))
	c = c.Append(hlog.RequestIDHandler("req_id", "Request-Id"))
	c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("verb", r.Method).
			Stringer("url", r.URL).
			Int("size", size).
			Int("status", status).
			Int64("duration", duration.Milliseconds()).
			Msg("REQ")
	}))

	return c.Then
}

func unmarshalRequestBody(req *http.Request, output interface{}) error {
	if req.Body == nil {
		return errors.New("invalid body in request")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if err = req.Body.Close(); err != nil {
		return err
	}
	if err = json.Unmarshal(body, &output); err != nil {
		return err
	}

	return nil
}
