// Auth API tests in Popcorn.

package auth

import (
	"Popcorn/internal/test"
	"Popcorn/internal/user"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"Popcorn/pkg/validations"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
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

// Global context
var ctx context.Context = context.Background()

// Auth testdata structure, helps in unmarshalling testdata/auth.json
type AuthTestData struct {
	// User registration testdata
	Register map[string]*struct {
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
func setupMockRouter(dbConnWrp *db.RedisDB, logger log.Logger) {
	mockRouter = test.MockRouter()
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
}

// Sets up resources before testing Auth APIs in Popcorn.
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
	// Db client instance
	client = db.NewDbConnection(ctx, logger)
	// Sending a PING request to DB for connection status check
	client.CheckDbConnection(ctx, logger)
	// Initializing validator
	govalidator.SetFieldsRequiredByDefault(true)
	// Adding custom validation tags into ext-package govalidator
	validations.RegisterCustomValidations(ctx, logger)
	user.RegisterCustomValidations(ctx, logger)
	// Initializing router
	setupMockRouter(client, logger)
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
}

// Cleans up the resources built during execution of setup()
func teardown() {
	logger.Info().Msg("Cleaning up resources ...")
	client.CleanTestDbData(ctx, logger)
	client.CloseDbConnection(ctx)
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
				Method:       http.MethodPost,
				Path:         "/api/auth/register",
				Body:         bytes.NewReader(body),
				WantResponse: data.WantResponse,
				Header:       test.MockHeader(),
				Parameters:   url.Values{},
				Cookie:       []*http.Cookie{},
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

func TestLogin(t *testing.T) {
	// Running a API test call to register an user first to test successful or Wrong password cases
	body, mrserr := json.Marshal(testdata.Register["TestUserSuccessOrDuplicateUser1"].Body)
	if mrserr != nil {
		logger.Error().Err(mrserr).Msg("Couldn't marshall authtest struct into json in TestRegister()")
		t.Fatal()
	}

	request := test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/auth/register",
		Body:         bytes.NewReader(body),
		WantResponse: testdata.Register["TestUserSuccessOrDuplicateUser1"].WantResponse,
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)

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
				Method:       http.MethodPost,
				Path:         "/api/auth/login",
				Body:         bytes.NewReader(body),
				WantResponse: data.WantResponse,
				Header:       test.MockHeader(),
				Parameters:   url.Values{},
				Cookie:       []*http.Cookie{},
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

func TestLogoutWithoutSettingToken(t *testing.T) {
	// Run a logout sub-test with token cookies not set, expected 401 - "Cookie not present"
	request := test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/auth/logout",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusUnauthorized},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}

func TestLogoutWithInvalidToken(t *testing.T) {
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

	request := test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/auth/logout",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusUnauthorized},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{&cookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}

func TestLogoutSuccess(t *testing.T) {
	// Running a successful register sub-test to get access_token & refresh_token
	// access_token will be useful to test Logout API
	var response test.APIResponse
	data := struct {
		Username interface{} `json:"username,omitempty"`
		FullName interface{} `json:"full_name,omitempty"`
		Password interface{} `json:"password,omitempty"`
	}{
		Username: "me_Bill..Weber..23",
		FullName: "Bill Weber",
		Password: "popcorn123",
	}
	body, mrserr := json.Marshal(data)
	if mrserr != nil {
		logger.Error().Err(mrserr).Msg("Couldn't marshall authtest struct into json in TestRegister()")
		t.Fatal()
	}

	request := test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/auth/register",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{},
	}
	// Save the response as we need the auth token cookie
	response = test.ExecuteAPITest(logger, t, mockRouter, &request)

	// Run a logout sub-test with valid token cookies, expected 200
	request = test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/auth/logout",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       response.Cookie,
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}

func TestRefreshTokenWithoutSettingToken(t *testing.T) {
	// Run a refresh_token sub-test with token cookies not set, expected 401 - "Cookie not present"
	request := test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/auth/refresh_token",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusUnauthorized},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}

func TestRefreshTokenWithInvalidToken(t *testing.T) {
	// Run a refresh_token sub-test with invalid token cookies, expected 401
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
	request := test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/auth/refresh_token",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusUnauthorized},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{&cookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}

func TestRefreshTokenSuccess(t *testing.T) {
	// Running a successful register sub-test to get access_token & refresh_token
	// refresh_token will be useful to test refreshToken API
	var initialResponse test.APIResponse
	data := struct {
		Username interface{} `json:"username,omitempty"`
		FullName interface{} `json:"full_name,omitempty"`
		Password interface{} `json:"password,omitempty"`
	}{
		Username: "me_Susan_Koerner..23",
		FullName: "Susan Koerner",
		Password: "popcorn123",
	}
	body, mrserr := json.Marshal(data)
	if mrserr != nil {
		logger.Error().Err(mrserr).Msg("Couldn't marshall authtest struct into json in TestRegister()")
		t.Fatal()
	}

	request := test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/auth/register",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{},
	}
	// Save the response as we need the auth token cookie
	initialResponse = test.ExecuteAPITest(logger, t, mockRouter, &request)

	// Run a refresh_token sub-test with valid token cookies, expected 200
	request = test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/auth/refresh_token",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       initialResponse.Cookie,
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}

func TestValidateTokenWithoutSettingToken(t *testing.T) {
	// Run a validate_token sub-test with token cookies not set, expected 401 - "Cookie not present"
	request := test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/auth/validate_token",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusUnauthorized},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}

func TestValidateTokenWithInvalidToken(t *testing.T) {
	// Run a validate_token sub-test with invalid token cookies, expected 401
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
	request := test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/auth/validate_token",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusUnauthorized},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{&cookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}

func TestValidateTokenSuccess(t *testing.T) {
	// Running a successful register sub-test to get access_token & refresh_token
	// access_token will be useful to test validateToken API
	var initialResponse test.APIResponse
	data := struct {
		Username interface{} `json:"username,omitempty"`
		FullName interface{} `json:"full_name,omitempty"`
		Password interface{} `json:"password,omitempty"`
	}{
		Username: "me_Pepe_Rodriguez..23",
		FullName: "Pepe Rodriguez",
		Password: "popcorn123",
	}
	body, mrserr := json.Marshal(data)
	if mrserr != nil {
		logger.Error().Err(mrserr).Msg("Couldn't marshall authtest struct into json in TestRegister()")
		t.Fatal()
	}

	request := test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/auth/register",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{},
	}
	// Save the response as we need the auth token cookie
	initialResponse = test.ExecuteAPITest(logger, t, mockRouter, &request)

	// Run a validate_token sub-test with valid token cookies, expected 200
	request = test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/auth/validate_token",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       initialResponse.Cookie,
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}
