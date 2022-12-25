// The main file of Popcorn.

package main

import (
	"Popcorn/internal/config"
	"Popcorn/pkg/logger"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

var (
	// Indicates the current version of Popcorn.
	Version = "1.0.0"
	// Indicates the current environment of Popcorn.
	Env string
)

func init() {
	// Fetching the mandatory env argument passed with main.go.
	// values should be either DEV or PROD for development or production environment.
	flag.StringVar(&Env, "env", "", "Popcorn Environment: DEV, PROD")
	flag.Parse()
	if !slices.Contains([]string{"DEV", "PROD"}, Env) {
		flag.PrintDefaults()
		os.Exit(2)
	}
	// Setup logger with Prettified logs in DEV, JSON in PROD environment.
	logger.Setup(Env)
	logger.Logger.Info().Msg(fmt.Sprintf("Welcome to Popcorn v%s", Version))
}

func main() {
	var addr, port string
	if Env == "DEV" {
		// load the development configurations.
		config.LoadDevConfig()
		// Fetching addr and port depending upon env flag.
		addr, port = os.Getenv("ADDR"), os.Getenv("PORT")
		// This is the preferred mode used by gin server in DEV environment.
		gin.SetMode(gin.DebugMode)
	}

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
