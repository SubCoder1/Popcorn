// The main.go of Popcorn where everything from loggers, Dbs to routers are configured.
// Distributions of third-party pkgs into internal services and repositories are also handled here.

package main

import (
	"Popcorn/internal/auth"
	"Popcorn/internal/errors"
	"Popcorn/internal/gang"
	"Popcorn/internal/user"
	"Popcorn/pkg/cleanup"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"Popcorn/pkg/middlewares"
	"Popcorn/pkg/validations"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
)

var (
	// Indicates the current version of Popcorn.
	version string = os.Getenv("VERSION")
	// Loads the environment set for Popcorn to run on [DEV, PROD].
	environment string = os.Getenv("ENV")
)

func main() {
	// Top-level context of Popcorn
	ctx := context.Background()

	// Initializing the logger
	logger := log.New(version)

	if environment == "" {
		// if environment is empty, it means the env file wasn't correctly loaded into the platform. Exit immediately!
		logger.Fatal().Err(errors.New("os couldn't load Environment variables.")).Msg("")
	}
	logger.Info().Msg("Welcome to Popcorn")
	logger.Info().Msg(fmt.Sprintf("Popcorn Environment: %s", environment))

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
		logger.Info().Msg(fmt.Sprintf("Popcorn service running at: %s", srvaddr+":"+srvport))
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Error in ListenAndServe()")
		}
	}()

	// Graceful shutdown of Popcorn server triggered due to system interruptions
	wait := cleanup.GracefulShutdown(ctx, logger, 5*time.Second, map[string]cleanup.Operation{
		"Redis-server": func(ctx context.Context) error {
			return dbConnWrp.CloseDbConnection(ctx)
		},
		"Gin": func(ctx context.Context) error {
			return srv.Shutdown(ctx)
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

	// Create Repository instance which will be used internally being passed around through service params
	authRepo := auth.NewRepository(dbConnWrp)
	userRepo := user.NewRepository(dbConnWrp)
	gangRepo := gang.NewRepository(dbConnWrp)

	// Declare internal middlewares here
	accAuthMiddleware := auth.AuthMiddleware(logger, authRepo, "access_token", accSecret)
	refAuthMiddleware := auth.AuthMiddleware(logger, authRepo, "refresh_token", refSecret)

	// Register handlers of different internal packages in Popcorn
	// Register internal package auth handler
	authService := auth.NewService(accSecret, refSecret, userRepo, authRepo, logger)
	auth.APIHandlers(router, authService, accAuthMiddleware, refAuthMiddleware, logger)
	// Register internal package user handler
	userService := user.NewService(userRepo, logger)
	user.APIHandlers(router, userService, accAuthMiddleware, logger)
	// Register internal package gang handler
	gangService := gang.NewService(gangRepo, userRepo, logger)
	gang.APIHandlers(router, gangService, accAuthMiddleware, logger)
	return router
}
