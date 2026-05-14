package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func New(environment string, level ...string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	if len(level) > 0 && level[0] != "" {
		parsed, err := zerolog.ParseLevel(level[0])
		if err == nil {
			zerolog.SetGlobalLevel(parsed)
		}
	}
	if environment == "development" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Logger()
	}
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
