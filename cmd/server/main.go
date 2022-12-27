// The main file of Popcorn.

package main

import (
	"Popcorn/pkg/cleanup"
	"Popcorn/pkg/db"
	"Popcorn/pkg/logger"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	// Indicates the current version of Popcorn.
	Version = "1.0.0"
	// Address and Port to be used by gin.
	srvaddr, srvport string
)

func init() {
	if len(os.Getenv("ENV")) == 0 {
		logger.Logger.Fatal().Err(errors.New("os couldn't load ENV."))
	}

	logger.Logger.Info().Msg(fmt.Sprintf("Welcome to Popcorn: v%s", Version))
	logger.Logger.Info().Msg(fmt.Sprintf("Popcorn Environment: %s", os.Getenv("ENV")))

	// Sending a PING request to DB for connection status check.
	err := db.PingToRedisServer(db.RedisDAO)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Redis client couldn't PING the redis-server.")
	}

	// Fetching addr and port depending upon env flag.
	srvaddr, srvport = os.Getenv("SRV_ADDR"), os.Getenv("SRV_PORT")
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
		Addr:    srvaddr + ":" + srvport,
		Handler: server,
	}

	// ListenAndServe is a blocking operation, putting it a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Logger.Fatal().Err(err)
		}
	}()

	// Graceful shutdown of Popcorn server triggered due to system interruptions.
	wait := cleanup.GracefulShutdown(context.Background(), 5*time.Second, map[string]cleanup.Operation{
		"Redis-server": func(ctx context.Context) error {
			return db.RedisDAO.Close()
		},
		"Gin": func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	<-wait
}
