// The main file of Popcorn.

package main

import (
	"Popcorn/pkg/logger"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	// Indicates the current version of Popcorn.
	Version = "1.0.0"
	// Address and Port to be used by gin.
	addr, port string
)

func init() {
	if len(os.Getenv("ENV")) == 0 {
		panic("[FATAL] Error during loading environment variables.")
	}

	logger.Logger.Info().Msg(fmt.Sprintf("Welcome to Popcorn: v%s", Version))
	logger.Logger.Info().Msg(fmt.Sprintf("Popcorn Environment: %s", os.Getenv("ENV")))

	// Fetching addr and port depending upon env flag.
	addr, port = os.Getenv("ADDR"), os.Getenv("PORT")
	// This is the preferred mode used by gin server in DEV environment.
	if os.Getenv("ENV") == "DEV" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
}

func main() {
	// Initializing the gin server.
	server := gin.New()

	// Forcing gin to use custom Logger instead of the default one.
	server.Use(logger.LoggerGinExtension(&logger.Logger))
	server.Use(gin.Recovery())

	// Running router.Router() which routes all of the REST API groups and paths.
	Router(server)

	// Running the server with defined addr and port.
	srv := &http.Server{
		Addr:    addr + ":" + port,
		Handler: server,
	}

	// For graceful-shutdown
	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil {
			logger.Logger.Err(err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	logger.Logger.Info().Msg("Shutting down Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Fatal().Msg(err.Error())
	}
	logger.Logger.Info().Msg("Shutdown completed.")
}
