package main

import (
	"context"
	"github.com/asynkron/protoactor-go/actor"
	zLog "github.com/rs/zerolog/log"
	"go-autogpt/internal/api"
	"go-autogpt/pkg/logger"
	"log"
	"os/signal"
	"syscall"
	"time"
)

// todo implement config
// only expected value is for OPENAI_API_KEY
func main() {
	log.Println("starting server")
	err := logger.NewGlobal("info", true)
	if err != nil {
		log.Panicf("failed to initialize logger: %v", err)
	}

	system := actor.NewActorSystem().Root
	app := api.New(system)

	go func() {
		err := app.Start()
		if err != nil {
			zLog.Panic().Err(err).Msg("server crash")
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	stop()
	zLog.Info().Msg("shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Stop(ctx); err != nil {
		zLog.Panic().Err(err).Msg("server forced to shutdown")
	}
	// todo shutdown actor system?

	zLog.Info().Msg("server exiting")
}
