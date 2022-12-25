// Custom logging utility used internally all over Popcorn.

package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var output io.Writer
var Logger zerolog.Logger

func Setup(env string) {
	// setting configurations for logger.
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339Nano
	if env == "DEV" {
		// Prettified ConsoleOutput for local environment.
		output = zerolog.ConsoleWriter{Out: os.Stdout}
	} else {
		// ConsoleWriter prettifies log, inefficient in prod.
		output = os.Stdout
	}
	// Finally, initialize the Logger with TimeStamp and CallStack features.
	Logger = zerolog.New(output).With().Timestamp().Caller().Stack().Logger()
	Logger.Info().Msg("Zerolog is ready to use!")
}
