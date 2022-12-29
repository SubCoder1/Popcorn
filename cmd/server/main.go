// The main file of Popcorn.

package main

import (
	"Popcorn/internal/auth"
	"Popcorn/internal/errors"
	"Popcorn/pkg/cleanup"
	"Popcorn/pkg/db"
	logger "Popcorn/pkg/log"
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
	Version     = "1.0.0"
	environment = os.Getenv("ENV")
)

func init() {
	if len(environment) == 0 {
		// Fatal starts a new message with fatal level.
		// The os.Exit(1) function is called by the Msg method, which terminates the program immediately.
		logger.Logger.Fatal().Err(errors.New("os couldn't load ENV.")).Msg("")
	}

	logger.Logger.Info().Msg(fmt.Sprintf("Welcome to Popcorn: v%s", Version))
	logger.Logger.Info().Msg(fmt.Sprintf("Popcorn Environment: %s", os.Getenv("ENV")))
}

func main() {
	ctx := context.Background()
	// Opening a Redis DB connection,
	// This object will be passed around internally for accessing the DB.
	var client *redis.Client
	dbConnWrp := db.NewDBConnection(client)
	// Sending a PING request to DB for connection status check.
	dberr := dbConnWrp.CheckDBConnection(ctx)
	if dberr != nil {
		logger.Logger.Fatal().Err(dberr).Msg("Redis client couldn't PING the redis-server.")
	}

	// Initializing validator
	govalidator.SetFieldsRequiredByDefault(true)
	// Adding custom validation tags into ext-package govalidator
	validation.RegisterCustomValidations()

	// Fetching server address and port from the environment.
	srvaddr, srvport := os.Getenv("SRV_ADDR"), os.Getenv("SRV_PORT")
	// Running the server with defined addr and port.
	srv := &http.Server{
		Addr:    srvaddr + ":" + srvport,
		Handler: buildHandler(dbConnWrp),
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
			return dbConnWrp.CloseDBConnection()
		},
		"Gin": func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	<-wait
}

// Helper to build up the server and register handlers from internal packages in Popcorn.
func buildHandler(dbConnWrp *db.RedisDB) *gin.Engine {
	// This is the preferred mode used by gin server in DEV environment.
	if os.Getenv("ENV") == "DEV" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	// Initializing the gin server.
	server := gin.New()

	// Forcing gin to use custom Logger instead of the default one.
	server.Use(logger.LoggerGinExtension(&logger.Logger))
	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	server.Use(gin.Recovery())
	// Register internal package auth handler
	auth.RegisterAUTHHandlers(server, auth.NewService(), dbConnWrp)

	return server
}
