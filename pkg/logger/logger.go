package logger

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

const (
	AgentNameField = "agent"
	TaskField      = "task"
	ActorIDField   = "actor"
	RequestTaskID  = "task"
)

func NewGlobal(level string, pretty bool) error {
	l, err := zerolog.ParseLevel(level)
	if err != nil {
		return err
	}

	zerolog.SetGlobalLevel(l)

	if pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	return nil
}
