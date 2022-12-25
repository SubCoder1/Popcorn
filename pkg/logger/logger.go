// Custom logging utility used internally all over Popcorn.

package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var (
	// zerolog based logger to be used internally all over Popcorn.
	Logger zerolog.Logger
	// Output of Logger based on what environment Popcorn is being run on.
	output io.Writer
)

func Setup(env string) {
	// setting configurations for logger.
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339Nano
	if env == "DEV" {
		// Set output of Logger to prettified ConsoleOutput for local environment.
		output = zerolog.ConsoleWriter{Out: os.Stdout}
	} else {
		// ConsoleWriter prettifies log, inefficient in prod.
		output = os.Stdout
	}
	// Finally, initialize the Logger with TimeStamp and CallStack features.
	Logger = zerolog.New(output).With().Timestamp().Caller().Stack().Logger()
	Logger.Info().Msg("Zerolog is ready to use!")
}
