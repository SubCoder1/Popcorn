// Custom logging utility used internally all over Popcorn.

package logger

import (
	"Popcorn/internal/config"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var output io.Writer
var Logger zerolog.Logger

func init() {
	// setting configurations for logger
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339Nano
	output = zerolog.ConsoleWriter{Out: os.Stdout}
	// ConsoleWriter prettifies log, inefficient in prod
	config.LoadDevConfig()
	if os.Getenv("ENV") != "DEV" {
		output = os.Stdout
	}
	// Finally, initialize the Logger with TimeStamp and CallStack features
	Logger = zerolog.New(output).With().Timestamp().Caller().Stack().Logger()
}
