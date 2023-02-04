// Auth API unit tests in Popcorn.

package auth

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/user"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"Popcorn/pkg/middlewares"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/joho/godotenv"
)

// Global instance of log.Logger to be used during auth API testing.
var logger log.Logger

// Global instance of gin MockRouter to be used during auth API testing.
var mockRouter *gin.Engine

// Global instance of Db instance to be used during auth API testing.
var client *db.RedisDB

// Auth testdata structure, helps in unmarshalling testdata/auth.json
type AuthTestData struct {
	Register map[string]struct {
		Body         entity.User `json:"body"`
		WantResponse int         `json:"response"`
	} `json:"register"`
}

// AuthTestData struct variable which stores unmarshalled all of the testdata for auth tests.
var testdata *AuthTestData

// Helper to build up a mock router instance for testing Popcorn.
func SetupMockRouter(ctx context.Context, dbConnWrp *db.RedisDB, logger log.Logger) *gin.Engine {
	// Initializing the gin test server
	mockRouter = gin.Default()
	mockRouter.Use(middlewares.CORSMiddleware("*")) // CORS middleware which allows request from all origin

	// Repositories needed by auth APIs and services to work
	authRepo := NewRepository(dbConnWrp)
	userRepo := user.NewRepository(dbConnWrp)
	// Middlewares used by auth APIs
	accAuthMiddleware := MockAuthMiddleware(logger, "access_token")
	refAuthMiddleware := MockAuthMiddleware(logger, "refresh_token")

	// Register internal package auth handler
	authService := NewService("MockAccessSecret", "MockRefreshSecret", userRepo, authRepo, logger)
	APIHandlers(mockRouter, authService, accAuthMiddleware, refAuthMiddleware, logger)

	return mockRouter
}

func TestMain(m *testing.M) {
	// Initializing Resources before test run
	// Load test.env
	enverr := godotenv.Load("../../config/test.env")
	if enverr != nil {
		// Error during loading test.env, abort test run immediately
		os.Exit(4)
	}
	version := os.Getenv("VERSION")
	ctx := context.Background()
	// Logger
	logger = log.New(version)
	// Db client instance
	client = db.NewDbConnection(ctx, logger)
	// Sending a PING request to DB for connection status check
	client.CheckDbConnection(ctx, logger)
	// Initializing validator
	govalidator.SetFieldsRequiredByDefault(true)
	// Adding custom validation tags into ext-package govalidator
	user.RegisterCustomValidations(ctx, logger)
	// Mock router instance
	mockRouter = SetupMockRouter(ctx, client, logger)
	// Read testdata and unmarshall into AuthTest
	datafilebytes, oserr := os.ReadFile("../../testdata/auth.json")
	if oserr != nil {
		// Error during readint testdata/auth.json
		logger.Fatal().Err(oserr).Msg("Couldn't read testdata/auth.json, Aborting test run.")
	}
	mrsherr := json.Unmarshal(datafilebytes, &testdata)
	if mrsherr != nil {
		// Error during unmarshalling into AuthTestData
		logger.Fatal().Err(mrsherr).Msg("Couldn't unmarshall into AuthTestData, Aborting test run.")
	}

	// Running the tests
	testExitCode := m.Run()

	// Cleanup Resources
	logger.Info().Msg("Cleaning up resources ...")
	client.CleanTestDbData(ctx, logger)
	client.CloseDbConnection(ctx)
	logger.Info().Msg("Cleanup complete :)")

	// Exit
	os.Exit(testExitCode)
}

func TestRegister(t *testing.T) {
	// Loop through every test scenarios defined in testdata/auth.json -> register
	for test, request := range testdata.Register {
		logger.Info().Msg(fmt.Sprintf("Running test - %s", test))
		request := request // Fixes "loop variable request captured by func literal" issue
		t.Run(test, func(t *testing.T) {
			t.Parallel()
			// Convert request.Body into bytes to add in NewRequest
			body, mrserr := json.Marshal(request.Body)
			if mrserr != nil {
				logger.Error().Err(mrserr).Msg("Couldn't marshall authtest struct into json in TestRegister()")
				t.Fatal()
			}
			// Make the test request
			req, reqerr := http.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if reqerr != nil {
				// Error in NewRequest
				logger.Error().Err(reqerr).Msg("Error occured during calling NewRequest in TestRegister()")
			}
			w := httptest.NewRecorder()
			mockRouter.ServeHTTP(w, req)
			// t.Log(w.Body)
			// Assert the response
			assert.Equal(t, request.WantResponse, w.Code)
		})
	}
}
