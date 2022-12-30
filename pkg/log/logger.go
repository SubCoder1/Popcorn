// Custom logging utility used internally all over Popcorn.

package log

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// Output of Logger based on what environment Popcorn is being run on.
var output io.Writer

func init() {
	// setting configurations for logger
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339Nano
	if os.Getenv("ENV") == "DEV" {
		// Set output of Logger to prettified ConsoleOutput for local environment
		output = zerolog.ConsoleWriter{Out: os.Stdout}
	} else {
		// ConsoleWriter prettifies log, inefficient in prod
		output = os.Stdout
	}
}

// Logger acts as a wrapper for zerolog with custom features.
type Logger interface {
	// WithCtx returns a sub-logger based of root logger with added context.
	WithCtx(context.Context) Logger
	// Info level log starts a log message with INFO level.
	Info() *zerolog.Event
	// Debug level log starts a log message with DEBUG level.
	Debug() *zerolog.Event
	// Warn level log starts a log message with WARNING level.
	Warn() *zerolog.Event
	// Error level log starts a log message with ERROR level.
	Error() *zerolog.Event
	// Fatal level log starts a log message with FATAL level.
	Fatal() *zerolog.Event
}

type logger struct {
	zerolog.Logger
}

// Creates a new logger instance for other packages to use the internal zerolog.
func New(version string) Logger {
	return &logger{zerolog.New(output).With().Str("Version", version).Timestamp().Caller().Stack().Logger()}
}

// Returns a sub-logger by adding additional requestID context to it.
// Helps in debugging issues.
func (l *logger) WithCtx(ctx context.Context) Logger {
	requestID := ctx.Value("ReqID").(string)
	if requestID != "" {
		return &logger{l.With().Str("ReqID", requestID).Timestamp().Caller().Stack().Logger()}
	}
	return l
}
