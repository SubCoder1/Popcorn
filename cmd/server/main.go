// The main file of Popcorn.

package main

import (
	"Popcorn/internal/auth"
	"Popcorn/internal/errors"
	"Popcorn/pkg/cleanup"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"Popcorn/pkg/validation"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var (
	// Indicates the current version of Popcorn.
	Version = "1.0.0"
	// Loads the environment set for Popcorn to run on [DEV, PROD].
	environment = os.Getenv("ENV")
)

func main() {
	// Top-level context of Popcorn
	ctx := context.WithValue(context.Background(), "ReqID", "main")

	// Initializing the logger
	logger := log.New(Version)

	if len(environment) == 0 {
		// Fatal starts a new message with fatal level
		// The os.Exit(1) function is called by the Msg method, which terminates the program immediately
		logger.WithCtx(ctx).Fatal().Err(errors.New("os couldn't load ENV.")).Msg("")
	}
	logger.WithCtx(ctx).Info().Msg("Welcome to Popcorn")
	logger.WithCtx(ctx).Info().Msg(fmt.Sprintf("Popcorn Environment: %s", os.Getenv("ENV")))

	// Opening a Redis DB connection
	// This object will be passed around internally for accessing the DB
	var client *redis.Client
	dbConnWrp := db.NewDBConnection(ctx, logger, client)
	// Sending a PING request to DB for connection status check
	dberr := dbConnWrp.CheckDBConnection(ctx, logger)
	if dberr != nil {
		logger.WithCtx(ctx).Fatal().Err(dberr).Msg("Redis client couldn't PING the redis-server.")
	}

	// Initializing validator
	govalidator.SetFieldsRequiredByDefault(true)
	// Adding custom validation tags into ext-package govalidator
	validation.RegisterCustomValidations(ctx, logger)

	// Fetching server address and port from the environment
	srvaddr, srvport := os.Getenv("SRV_ADDR"), os.Getenv("SRV_PORT")
	// Running the server with defined addr and port.
	srv := &http.Server{
		Addr:    srvaddr + ":" + srvport,
		Handler: buildHandler(ctx, dbConnWrp, logger),
	}

	// ListenAndServe is a blocking operation, putting it a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Error().Err(err).Msg("Error in ListenAndServe()")
		}
	}()

	// Graceful shutdown of Popcorn server triggered due to system interruptions
	wait := cleanup.GracefulShutdown(ctx, logger, 5*time.Second, map[string]cleanup.Operation{
		"Redis-server": func(ctx context.Context) error {
			return dbConnWrp.CloseDBConnection(ctx)
		},
		"Gin": func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	<-wait
}

// Helper to build up the server and register handlers from internal packages in Popcorn
func buildHandler(ctx context.Context, dbConnWrp *db.RedisDB, logger log.Logger) *gin.Engine {
	// This is the preferred mode used by gin server in DEV environment
	if os.Getenv("ENV") == "DEV" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	// Initializing the gin server
	server := gin.New()

	// Forcing gin to use custom Logger instead of the default one
	server.Use(log.LoggerGinExtension(logger))
	// Recovery middleware recovers from any panics and writes a 500 if there was one
	server.Use(gin.Recovery())

	server.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "middleware isn't the issue"})
	})

	// Register handlers of different internal packages in Popcorn
	// Register internal package auth handler
	auth.RegisterAUTHHandlers(server, auth.NewService(logger), dbConnWrp, logger)

	return server
}
