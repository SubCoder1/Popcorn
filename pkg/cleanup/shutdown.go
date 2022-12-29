// Closes open external connections before shutting down Popcorn.
// Inspired from https://medium.com/tokopedia-engineering/gracefully-shutdown-your-go-application-9e7d5c73b5ac

package cleanup

import (
	"Popcorn/pkg/logger"
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// operation is a clean up function standard.
type Operation func(ctx context.Context) error

// GracefulShutdown function waits for termination system-calls and performs clean-up operations.
func GracefulShutdown(ctx context.Context, timeout time.Duration, operations map[string]Operation) <-chan bool {
	wait := make(chan bool)

	go func() {
		// buffered channel to receive shutdown signal
		s := make(chan os.Signal, 1)
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		<-s

		logger.Logger.Warn().Msg("Graceful shutdown in progress.")

		// Force exit after timeout duration has been elapsed
		force := time.AfterFunc(timeout, func() {
			logger.Logger.Warn().Msg(fmt.Sprintf("Timeout of %fs has been elapsed. Forcing shutdown!", timeout.Seconds()))
			os.Exit(3)
		})
		defer force.Stop()

		// Executing the cleanup operations asynchronously for better performance
		var wg sync.WaitGroup

		for opname, op := range operations {
			// Adding task to be executed asynchronously
			wg.Add(1)
			go func(opname string, op Operation) {
				defer wg.Done()
				logger.Logger.Info().Msg(fmt.Sprintf("Shutting down: %s", opname))
				if err := op(ctx); err != nil {
					logger.Logger.Fatal().Err(err)
				}
				logger.Logger.Info().Msg(fmt.Sprintf("%s shutdown completed.", opname))
			}(opname, op)
		}
		// Wait for all of the tasks to finish
		wg.Wait()
		close(wait)
	}()

	return wait
}
