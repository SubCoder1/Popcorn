// Graceful shutdown tests in Popcorn.

package cleanup

import (
	"Popcorn/internal/test"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

// Global instance of log.Logger to be used during cleanup testing.
var logger log.Logger

// Global instance of gin MockRouter to be used during cleanup testing.
var mockRouter *gin.Engine

// Global instance of gin MockRouter to be used during cleanup testing.
var srv *http.Server

// Address and Port of srv
var srvaddr, srvport string

// Global instance of Db instance to be used during cleanup testing.
var client *db.RedisDB

// Global context
var ctx context.Context = context.Background()

// Helper to build up a mock router instance for testing Popcorn.
func setupMockRouter(dbConnWrp *db.RedisDB, logger log.Logger) {
	// Initializing mock router
	mockRouter = test.MockRouter()
	mockRouter.GET("/api", func(gctx *gin.Context) {
		gctx.Status(http.StatusOK)
	})
}

// Sets up resources before testing graceful shutdown in Popcorn.
func setup() {
	// Initializing Resources before test run

	// Load test.env
	enverr := godotenv.Load("../../config/test.env")
	if enverr != nil {
		// Error during loading test.env, abort test run immediately
		os.Exit(4)
	}
	version := os.Getenv("VERSION")
	srvaddr, srvport = os.Getenv("SRV_ADDR"), os.Getenv("SRV_PORT")

	// Logger
	logger = log.New(version)

	// Db client instance
	var dberr error
	client, dberr = db.NewDbConnection(ctx, logger)
	if dberr != nil || client.CheckDbConnection(ctx, logger) != nil {
		// connection failure
		os.Exit(6)
	}
	// Sending a PING request to DB for connection status check
	client.CheckDbConnection(ctx, logger)

	// Initializing router
	setupMockRouter(client, logger)

	// Running the server with defined addr and port.
	srv = &http.Server{
		Addr:    srvaddr + ":" + srvport,
		Handler: mockRouter,
	}

	logger.Info().Msg("Test resources setup successful.")
}

// Cleans up the resources built during execution of setup()
func teardown() {
	logger.Info().Msg("Cleaning up resources ...")
	if client.CheckDbConnection(ctx, logger) == nil {
		// client still open
		client.CleanTestDbData(ctx, logger)
		client.CloseDbConnection(ctx)
	}
	logger.Info().Msg("Cleanup complete :)")
}

func TestMain(m *testing.M) {
	// Setting up Resources
	setup()
	// Running the tests
	testExitCode := m.Run()
	// Cleanup Resources
	teardown()
	// Exit
	os.Exit(testExitCode)
}

func TestGracefulShutdownSIGINT(t *testing.T) {
	// ListenAndServe is a blocking operation, putting it a goroutine
	go func() {
		logger.Info().Msg(fmt.Sprintf("Popcorn service running at: %s", srvaddr+":"+srvport))
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("Error in ListenAndServe()")
		}
	}()
	// Send SIGINT signal to test graceful shutdown
	go func(logger log.Logger) {
		time.Sleep(5000)
		logger.Info().Msg("Sending SIGINT signal")
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}(logger)

	// Graceful shutdown of Popcorn server triggered due to system interruptions
	wait := GracefulShutdown(ctx, logger, 5*time.Second, map[string]Operation{
		"Redis-server": func(ctx context.Context) error {
			return client.CloseDbConnection(ctx)
		},
		"Gin": func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	<-wait

	assert.True(t, client.CheckDbConnection(ctx, logger) != nil)
	_, testerr := http.Get(fmt.Sprintf("http://%s:%s/api", srvaddr, srvport))
	assert.True(t, testerr != nil)
}

func TestGracefulShutdownSIGTERM(t *testing.T) {
	// ListenAndServe is a blocking operation, putting it a goroutine
	go func() {
		logger.Info().Msg(fmt.Sprintf("Popcorn service running at: %s", srvaddr+":"+srvport))
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("Error in ListenAndServe()")
		}
	}()
	// Send SIGINT signal to test graceful shutdown
	go func() {
		time.Sleep(5000)
		logger.Info().Msg("Sending SIGTERM signal")
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	// Graceful shutdown of Popcorn server triggered due to system interruptions
	wait := GracefulShutdown(ctx, logger, 5*time.Second, map[string]Operation{
		"Redis-server": func(ctx context.Context) error {
			return client.CloseDbConnection(ctx)
		},
		"Gin": func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	<-wait

	assert.True(t, client.CheckDbConnection(ctx, logger) != nil)
	_, testerr := http.Get(fmt.Sprintf("http://%s:%s/api", srvaddr, srvport))
	assert.True(t, testerr != nil)
}
