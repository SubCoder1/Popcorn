// Closes open external connections before shutting down Popcorn.
// Inspired from https://medium.com/tokopedia-engineering/gracefully-shutdown-your-go-application-9e7d5c73b5ac

package cleanup

import (
	"Popcorn/pkg/log"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// operation is a clean up function standard.
type Operation func(ctx context.Context) error

// GracefulShutdown function waits for termination system-calls and performs clean-up operations.
func GracefulShutdown(ctx context.Context, logger log.Logger, timeout time.Duration, operations []Operation) <-chan bool {
	wait := make(chan bool)

	go func() {
		// buffered channel to receive shutdown signal
		s := make(chan os.Signal, 1)
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		<-s

		logger.WithCtx(ctx).Warn().Msg("Graceful shutdown in progress.")

		// Putting process shutdown on wait till all of the ffmpeg processes are closed
		time.Sleep(2 * time.Second)

		// Force exit after timeout duration has been elapsed
		force := time.AfterFunc(timeout, func() {
			logger.WithCtx(ctx).Warn().Msg(fmt.Sprintf("Timeout of %fs has been elapsed. Forcing shutdown!", timeout.Seconds()))
			os.Exit(2)
		})
		defer force.Stop()

		for _, op := range operations {
			if err := op(ctx); err != nil {
				logger.WithCtx(ctx).Fatal().Err(err)
			}
		}

		close(wait)
	}()

	return wait
}
