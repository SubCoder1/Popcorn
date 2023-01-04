// The main file of Popcorn.

package main

import (
	"Popcorn/internal/auth"
	"Popcorn/internal/errors"
	"Popcorn/internal/user"
	"Popcorn/pkg/cleanup"
	"Popcorn/pkg/db"
	"Popcorn/pkg/globalcontext"
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
		// if environment is empty, it means the env file wasn't correctly loaded into the platform. Exit immediately!
		logger.WithCtx(ctx).Fatal().Err(errors.New("os couldn't load ENV.")).Msg("")
	}
	logger.WithCtx(ctx).Info().Msg("Welcome to Popcorn")
	logger.WithCtx(ctx).Info().Msg(fmt.Sprintf("Popcorn Environment: %s", environment))

	// Opening a Redis DB connection
	// This object will be passed around internally for accessing the DB
	var client *redis.Client
	dbConnWrp := db.NewDBConnection(ctx, logger, client)
	// Sending a PING request to DB for connection status check
	dbConnWrp.CheckDBConnection(ctx, logger)
	// Setting default values to global keys if not exist already in the DB
	// Ex of global key: total number of users -> users:0
	// The values of these global keys will be used/changed internally
	globalDBkeys := map[string]interface{}{"users": 0}
	if _, dberr := dbConnWrp.SetGlobKeyIfNotExists(ctx, logger, globalDBkeys); dberr != nil {
		// Global keys not set, better to exit else unknown issues will pop-up in server code
		os.Exit(3)
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
	// Set any environment variables to be used in handlers here
	accSecret := os.Getenv("ACCESS_SECRET")
	refSecret := os.Getenv("REFRESH_SECRET")

	// Set the gin mode according to environment
	if environment == "DEV" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	// Initializing the gin server
	server := gin.New()

	// Declare global middlewares here
	server.Use(log.LoggerGinExtension(logger))           // Forcing gin to use custom Logger instead of the default one
	server.Use(gin.Recovery())                           // Recovery middleware recovers from any panics and writes a 500 if there was one
	server.Use(globalcontext.UniqueIDMiddleware(logger)) // Fill up every request with unique UUID

	// Create Repository instance which will be used internally being passed around through service params
	authrepo := auth.NewRepository(dbConnWrp)
	userrepo := user.NewRepository(dbConnWrp)

	// Declare internal middlewares here
	accAuthMiddleware := auth.AuthMiddleware(logger, authrepo, "access_token", accSecret)
	refAuthMiddleware := auth.AuthMiddleware(logger, authrepo, "refresh_token", refSecret)

	// Register handlers of different internal packages in Popcorn
	// Register internal package auth handler
	authservice := auth.NewService(accSecret, refSecret, userrepo, authrepo, logger)
	auth.AuthHandlers(server, authservice, accAuthMiddleware, refAuthMiddleware, logger)

	return server
}
