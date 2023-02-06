// Auth API unit tests in Popcorn.

package auth

import (
	"Popcorn/internal/test"
	"Popcorn/internal/user"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"Popcorn/pkg/middlewares"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"

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
	// User registration testdata
	Register map[string]struct {
		Body *struct {
			Username interface{} `json:"username,omitempty"`
			FullName interface{} `json:"full_name,omitempty"`
			Password interface{} `json:"password,omitempty"`
		} `json:"body,omitempty"`
		WantResponse []int `json:"response"`
	} `json:"register"`
	// User login testdata
	Login map[string]struct {
		Body *struct {
			Username interface{} `json:"username,omitempty"`
			Password interface{} `json:"password,omitempty"`
		} `json:"body,omitempty"`
		WantResponse []int `json:"response"`
	} `json:"login"`
}

// AuthTestData struct variable which stores unmarshalled all of the testdata for auth tests.
var testdata *AuthTestData

// Helper to build up a mock router instance for testing Popcorn.
func SetupMockRouter(ctx context.Context, dbConnWrp *db.RedisDB, logger log.Logger) *gin.Engine {
	// Initializing the gin test server
	mockRouter = gin.Default()
	mockRouter.Use(middlewares.CORSMiddleware("*")) // CORS middleware which allows request from all origin
	// Mock secret keys to create auth token
	mockAccSecret, mockRefSecret := "MockAccessSecret", "MockRefreshSecret"

	// Repositories needed by auth APIs and services to work
	authRepo := NewRepository(dbConnWrp)
	userRepo := user.NewRepository(dbConnWrp)
	// Middlewares used by auth APIs
	accAuthMiddleware := AuthMiddleware(logger, authRepo, "access_token", mockAccSecret)
	refAuthMiddleware := AuthMiddleware(logger, authRepo, "refresh_token", mockRefSecret)

	// Register internal package auth handler
	authService := NewService(mockAccSecret, mockRefSecret, userRepo, authRepo, logger)
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
	logger.Info().Msg("Test resources setup successful.")

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
	for name, data := range testdata.Register {
		data := data // Fixes "loop variable request captured by func literal" issue
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Convert request.Body into bytes to add in NewRequest
			body, mrserr := json.Marshal(data.Body)
			if mrserr != nil {
				logger.Error().Err(mrserr).Msg("Couldn't marshall authtest struct into json in TestRegister()")
				t.Fatal()
			}

			request := test.RequestAPITest{
				Method:       "POST",
				Path:         "/api/auth/register",
				Body:         bytes.NewReader(body),
				WantResponse: data.WantResponse,
				Header:       test.MockHeader(),
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

func TestLogin(t *testing.T) {
	// Running a sub-test to register an user first to test successful or Wrong password cases
	// This test cannot be ran parallel to the ones below as it might give uneven results
	t.Run("TestUserSuccessOrDuplicateUser1", func(t *testing.T) {
		// Convert request.Body into bytes to add in NewRequest
		body, mrserr := json.Marshal(testdata.Register["TestUserSuccessOrDuplicateUser1"].Body)
		if mrserr != nil {
			logger.Error().Err(mrserr).Msg("Couldn't marshall authtest struct into json in TestRegister()")
			t.Fatal()
		}

		request := test.RequestAPITest{
			Method:       "POST",
			Path:         "/api/auth/register",
			Body:         bytes.NewReader(body),
			WantResponse: testdata.Register["TestUserSuccessOrDuplicateUser1"].WantResponse,
			Header:       test.MockHeader(),
		}
		test.ExecuteAPITest(logger, t, mockRouter, &request)
	})
	// Loop through every test scenarios defined in testdata/auth.json -> login
	for name, data := range testdata.Login {
		data := data // Fixes "loop variable request captured by func literal" issue
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Convert request.Body into bytes to add in NewRequest
			body, mrserr := json.Marshal(data.Body)
			if mrserr != nil {
				logger.Error().Err(mrserr).Msg("Couldn't marshall authtest struct into json in TestLogin()")
				t.Fatal()
			}

			request := test.RequestAPITest{
				Method:       "POST",
				Path:         "/api/auth/login",
				Body:         bytes.NewReader(body),
				WantResponse: data.WantResponse,
				Header:       test.MockHeader(),
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

func TestLogout(t *testing.T) {
	// Running a successful register sub-test to get access_token & refresh_token
	// These 2 tokens will be useful to test logout
	var response test.APIResponse
	t.Run("TestRegisterUserForLogout", func(t *testing.T) {
		// Convert request.Body into bytes to add in NewRequest
		data := struct {
			Username interface{} `json:"username,omitempty"`
			FullName interface{} `json:"full_name,omitempty"`
			Password interface{} `json:"password,omitempty"`
		}{
			Username: "me_Bill..Weber..23",
			FullName: "Bill Weber",
			Password: "popcornarnab1",
		}
		body, mrserr := json.Marshal(data)
		if mrserr != nil {
			logger.Error().Err(mrserr).Msg("Couldn't marshall authtest struct into json in TestRegister()")
			t.Fatal()
		}

		request := test.RequestAPITest{
			Method:       "POST",
			Path:         "/api/auth/register",
			Body:         bytes.NewReader(body),
			WantResponse: testdata.Register["TestUserSuccessOrDuplicateUser1"].WantResponse,
			Header:       test.MockHeader(),
			Cookie:       []*http.Cookie{},
		}
		// Save the response as we need the auth token cookie
		response = test.ExecuteAPITest(logger, t, mockRouter, &request)
	})

	// Run a logout sub-test with token cookies not set, expected 401 - "Cookie not present"
	t.Run("TestLogoutWithoutSettingToken", func(t *testing.T) {
		request := test.RequestAPITest{
			Method:       "POST",
			Path:         "/api/auth/logout",
			Body:         bytes.NewReader([]byte{}),
			WantResponse: []int{401},
			Header:       test.MockHeader(),
			Cookie:       []*http.Cookie{},
		}
		test.ExecuteAPITest(logger, t, mockRouter, &request)
	})

	// Run a logout sub-test with invalid token cookies, expected 401
	mockClaim := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 4)),
		Issuer:    "Test",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   "Authentication",
	}
	mockToken := jwt.NewWithClaims(jwt.SigningMethodHS256, mockClaim)
	access_token, _ := mockToken.SignedString([]byte("MockAccessSecret"))
	var domain string = os.Getenv("SRV_ADDR")
	cookie := http.Cookie{
		Name:     "access_token",
		Value:    access_token,
		Expires:  time.Now().Add(time.Hour * 4),
		MaxAge:   (4 * 60) * 60,
		Domain:   domain,
		Path:     "/api",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}
	t.Run("TestLogoutWithInvalidHeaders", func(t *testing.T) {
		request := test.RequestAPITest{
			Method:       "POST",
			Path:         "/api/auth/logout",
			Body:         bytes.NewReader([]byte{}),
			WantResponse: []int{401},
			Header:       test.MockHeader(),
			Cookie:       []*http.Cookie{&cookie},
		}
		test.ExecuteAPITest(logger, t, mockRouter, &request)
	})

	// Run a logout sub-test with valid token cookies, expected 200
	t.Run("TestLogoutWithValidToken", func(t *testing.T) {
		request := test.RequestAPITest{
			Method:       "POST",
			Path:         "/api/auth/logout",
			Body:         bytes.NewReader([]byte{}),
			WantResponse: []int{200},
			Header:       test.MockHeader(),
			Cookie:       response.Cookie,
		}
		test.ExecuteAPITest(logger, t, mockRouter, &request)
	})
}
