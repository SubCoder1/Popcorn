// Redis DB Connetor tests in Popcorn.

package db

import (
	"Popcorn/pkg/log"
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

// Global instance of log.Logger to be used during cleanup testing.
var logger log.Logger

// Global instance of Db instance to be used during cleanup testing.
var client *RedisDB

// Global context
var ctx context.Context = context.Background()

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

	// Logger
	logger = log.New(version)
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

func TestDbConnectionLifeCycle(t *testing.T) {
	var dberr error
	client, dberr = NewDbConnection(ctx, logger)
	// Check if there were any issues returned from NewDbConnection
	assert.True(t, dberr == nil)
	// Check if connection is successful
	assert.True(t, client.CheckDbConnection(ctx, logger) == nil)
	// Close connection
	assert.True(t, client.CloseDbConnection(ctx) == nil)
	// Check if connection is still active
	assert.False(t, client.CheckDbConnection(ctx, logger) == nil)
}
