// The main.go of Popcorn where everything from loggers, Dbs to routers are configured.
// Distributions of third-party pkgs into internal services and repositories are also handled here.

package main

import (
	"Popcorn/internal/auth"
	"Popcorn/internal/errors"
	"Popcorn/internal/gang"
	"Popcorn/internal/sse"
	"Popcorn/internal/storage"
	"Popcorn/internal/user"
	"Popcorn/pkg/cleanup"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"Popcorn/pkg/middlewares"
	"Popcorn/pkg/validations"
	"context"
	"net/http"
	"os"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
)

var (
	VERSION     = os.Getenv("VERSION") // Indicates the current version of Popcorn.
	ENVIRONMENT = os.Getenv("ENV")     // Loads the environment set for Popcorn to run on [DEV, PROD].
	// Livekit credentials
	LIVEKIT_CONFIG = gang.LivekitConfig{
		Host:      os.Getenv("LIVEKIT_HOST"),
		ApiKey:    os.Getenv("LIVEKIT_API_KEY"),
		ApiSecret: os.Getenv("LIVEKIT_SECRET_KEY"),
	}
)

func main() {
	// Top-level context of Popcorn
	ctx := context.Background()

	// Initializing the logger
	logger := log.New(VERSION)

	if ENVIRONMENT == "" {
		// if environment is empty, it means the env file wasn't correctly loaded into the platform. Exit immediately!
		logger.Fatal().Err(errors.New("os couldn't load Environment variables.")).Msg("")
	}
	logger.Info().Msg("Welcome to Popcorn")
	logger.Info().Msgf("Popcorn Environment: %s", ENVIRONMENT)

	// Opening a Redis DB connection
	// This object will be passed around internally for accessing the DB
	dbConnWrp, dberr := db.NewDbConnection(ctx, logger)
	// Sending a PING request to DB for connection status check
	if dberr != nil || dbConnWrp.CheckDbConnection(ctx, logger) != nil {
		// Db connection failure
		os.Exit(6)
	}

	// Initializing validator
	govalidator.SetFieldsRequiredByDefault(true)
	// Adding custom validation tags into ext-package govalidator
	validations.RegisterCustomValidations(ctx, logger)
	user.RegisterCustomValidations(ctx, logger)
	gang.RegisterCustomValidations(ctx, logger)

	// Fetching server address and port from the environment
	srvaddr, srvport := os.Getenv("SRV_ADDR"), os.Getenv("SRV_PORT")
	// Running the server with defined addr and port.
	srv := &http.Server{
		Addr:    srvaddr + ":" + srvport,
		Handler: setupRouter(ctx, dbConnWrp, logger),
	}
	// ListenAndServe is a blocking operation, putting it a goroutine
	go func() {
		logger.Info().Msgf("Popcorn service running at: %s", srvaddr+":"+srvport)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Error in ListenAndServe()")
		}
	}()

	// Graceful shutdown of Popcorn server triggered due to system interruptions
	wait := cleanup.GracefulShutdown(ctx, logger, 5*time.Minute, []cleanup.Operation{
		func(ctx context.Context) error {
			// Stop all open stream ingresses
			return gang.DeleteAllOpenIngress(ctx, logger, LIVEKIT_CONFIG)
		},
		func(ctx context.Context) error {
			// Disconnect SSE connections & coressponding channels, then shutdown gin server
			sse.Cleanup(ctx)
			return srv.Shutdown(ctx)
		},
		func(ctx context.Context) error {
			// Close Redis DB Connection
			return dbConnWrp.CloseDbConnection(ctx)
		},
	})
	<-wait
}

// Helper to build up the router and register handlers from internal packages in Popcorn.
func setupRouter(ctx context.Context, dbConnWrp *db.RedisDB, logger log.Logger) *gin.Engine {
	// Set any environment variables to be used in handlers here
	accSecret := os.Getenv("ACCESS_SECRET")
	refSecret := os.Getenv("REFRESH_SECRET")

	addr := os.Getenv("ACCESS_CTL_ALLOW_ORGIN")

	// Initializing the gin server
	ginMode := os.Getenv("GIN_MODE")
	gin.SetMode(ginMode)
	router := gin.New()

	// Declare global middlewares here
	router.Use(log.LoggerGinExtension(logger))            // Forcing gin to use custom Logger instead of the default one
	router.Use(gin.Recovery())                            // Recovery middleware recovers from any panics and writes a 500 if there was one
	router.Use(middlewares.CorrelationMiddleware(logger)) // Fill up every request with unique CorrelationID
	router.Use(middlewares.CORSMiddleware(addr))          // CORS middleware

	// Initialize Repository instance which will be used internally being passed around through service params
	authRepo := auth.NewRepository(dbConnWrp)
	userRepo := user.NewRepository(dbConnWrp)
	gangRepo := gang.NewRepository(dbConnWrp)

	// Initialize internal Service instance
	authService := auth.NewService(accSecret, refSecret, userRepo, authRepo, logger)
	userService := user.NewService(userRepo, logger)
	sseService := sse.NewService(logger)
	gangService := gang.NewService(LIVEKIT_CONFIG, gangRepo, userRepo, sseService, logger)

	// Launch SSE Listener in a seperate goroutine
	sseService.GetOrSetEvent(ctx)
	go sseService.Listen(ctx)

	// Declare internal middlewares here
	accAuthMiddleware := auth.AuthMiddleware(logger, authRepo, userRepo, "access_token", accSecret)
	refAuthMiddleware := auth.AuthMiddleware(logger, authRepo, userRepo, "refresh_token", refSecret)
	sseConnMiddleware := sse.SSEConnManagerMiddleware(sseService, logger)
	tusAuthMiddleware := storage.ContentStorageMiddleware(logger, gangRepo)

	// Register handlers of different internal packages in Popcorn
	// Register internal package auth handler
	auth.APIHandlers(router, authService, accAuthMiddleware, refAuthMiddleware, logger)
	// Register internal package user handler
	user.APIHandlers(router, userService, accAuthMiddleware, logger)
	// Register internal package gang handler
	gang.APIHandlers(router, gangService, accAuthMiddleware, logger)
	// Register internal package sse handler
	sse.APIHandlers(router, sseService, accAuthMiddleware, sseConnMiddleware, logger)
	// Register tusd file storage handler
	storage_handler := storage.GetTusdStorageHandler(gangRepo, sseService, logger)
	storage.APIHandlers(router, storage_handler, accAuthMiddleware, tusAuthMiddleware, logger)

	return router
}
