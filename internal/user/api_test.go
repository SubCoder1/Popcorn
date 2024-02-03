// User API tests in Popcorn.

package user

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/test"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"Popcorn/pkg/validations"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

// Global instance of log.Logger to be used during user API testing.
var logger log.Logger

// Global instance of gin MockRouter to be used during user API testing.
var mockRouter *gin.Engine

// Global instance of Db instance to be used during user API testing.
var client *db.RedisDB

// Global instance of user Repository to be used during user API testing.
var userRepo Repository

// Global context
var ctx context.Context = context.Background()

// User testdata structure, helps in unmarshalling testdata/user.json
type UserTestData struct {
	SearchUserInvalid map[string]*struct {
		Body     url.Values `json:"body,omitempty"`
		Response []int      `json:"response"`
	} `json:"search_user_invalid"`
	SearchUserValid map[string]*struct {
		Body     url.Values `json:"body,omitempty"`
		Response []int      `json:"response"`
	} `json:"search_user_valid"`
	UserList []entity.User `json:"user_list,omitempty"`
}

// UserTestData struct variable which stores unmarshalled all of the testdata for user tests.
var testdata *UserTestData

// TestUser account to be used during user API tests
var testUser entity.User

// TestUser Cookie to be passed during tests
var userCookie http.Cookie

// Helper to build up a mock router instance for testing Popcorn.
func setupMockRouter(dbConnWrp *db.RedisDB, logger log.Logger) {
	// Mock router instance
	mockRouter = test.MockRouter()

	// Repositories needed by user APIs and services to work
	userRepo = NewRepository(dbConnWrp)

	// Register internal package user handler
	userService := NewService(userRepo, logger)
	APIHandlers(mockRouter, userService, test.MockAuthMiddleware(logger), logger)
}

// Helper to register test user required in the tests below
func registerTestUser(username, fullname string) {
	// Use user.SetOrUpdate repository method to set user data
	testUser = entity.User{
		Username: username,
		FullName: fullname,
		Password: "popcorn123",
	}
	testUser.SelectProfilePic()
	_, dberr := userRepo.SetOrUpdateUser(ctx, logger, testUser, true)
	if dberr != nil {
		// Issues in SetOrUpdateUser()
		logger.Fatal().Err(dberr).Msg("Couldn't create testUser, Aborting test run.")
	}
	// User Cookie to be passed during tests
	userCookie = http.Cookie{
		Name:     "user",
		Value:    username,
		HttpOnly: true,
	}
}

// Initializes resources needed before user API tests.
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
	var dberr error
	client, dberr = db.NewDbConnection(ctx, logger)
	// Sending a PING request to DB for connection status check
	if dberr != nil || client.CheckDbConnection(ctx, logger) != nil {
		// connection failure
		os.Exit(6)
	}
	// Initializing validator
	govalidator.SetFieldsRequiredByDefault(true)
	// Adding custom validation tags into ext-package govalidator
	validations.RegisterCustomValidationTags(ctx, logger)
	RegisterCustomValidationTags(ctx, logger)

	// Initializing router
	setupMockRouter(client, logger)

	// Read testdata and unmarshall into UserTestData
	datafilebytes, oserr := os.ReadFile("../../testdata/user.json")
	if oserr != nil {
		// Error during readint testdata/user.json
		logger.Fatal().Err(oserr).Msg("Couldn't read testdata/user.json, Aborting test run.")
	}
	mrsherr := json.Unmarshal(datafilebytes, &testdata)
	if mrsherr != nil {
		// Error during unmarshalling into UserTestData
		logger.Fatal().Err(mrsherr).Msg("Couldn't unmarshall into UserTestData, Aborting test run.")
	}

	// Setup a test user account to be used for user API testing
	registerTestUser("me_Oswaldo_Solis..23", "Oswaldo Solis")

	logger.Info().Msg("Test resources setup successful.")
}

// Cleans up the resources built during execution of setup().
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

func TestGetUser404(t *testing.T) {
	// Make a call to get_user API to fetch registered user data
	registeredUserCookie := http.Cookie{
		Name:     "user",
		Value:    "me_Carlos_Gillespie..23",
		HttpOnly: true,
	}
	request := test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/user/get",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusNotFound},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &registeredUserCookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}

func TestGetUserSuccess(t *testing.T) {
	// Make a call to get_user API to fetch registered user data
	request := test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/user/get",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}

func TestSearchUserInvalid(t *testing.T) {
	// Loop through every test scenarios defined in testdata/user.json -> search_user_invalid
	for subTestName, subTestBody := range testdata.SearchUserInvalid {
		subTestBody := subTestBody
		t.Run(subTestName, func(t *testing.T) {
			t.Parallel()
			request := test.RequestAPITest{
				Method:       http.MethodGet,
				Path:         "/api/user/search",
				Body:         bytes.NewReader([]byte{}),
				WantResponse: subTestBody.Response,
				Header:       test.MockHeader(),
				Parameters:   subTestBody.Body,
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

func TestSearchUserValid(t *testing.T) {
	// Loop through every test scenarios defined in testdata/user.json -> search_user_valid
	for subTestName, subTestBody := range testdata.SearchUserValid {
		subTestBody := subTestBody
		t.Run(subTestName, func(t *testing.T) {
			t.Parallel()
			request := test.RequestAPITest{
				Method:       http.MethodGet,
				Path:         "/api/user/search",
				Body:         bytes.NewReader([]byte{}),
				WantResponse: subTestBody.Response,
				Header:       test.MockHeader(),
				Parameters:   subTestBody.Body,
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

func TestSearchUserPaginated(t *testing.T) {
	// Register list of users from testdata/user.json to test pagination in search_user API
	// We're directly accessing repository method as we only need to search the user
	// So no need to hash user's password
	var wg sync.WaitGroup
	for _, testUser := range testdata.UserList {
		testUser := testUser
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, dberr := userRepo.SetOrUpdateUser(ctx, logger, testUser, true)
			if dberr != nil {
				t.Fail()
			}
		}()
	}
	wg.Wait()

	// Make a call to search_user API with "me." as search param
	// Every username registered above started with me.
	// Expected response, 200 with a new cursor (pagination)
	request := test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/user/search",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{"username": {"me."}},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
	}
	response := test.ExecuteAPITest(logger, t, mockRouter, &request)
	searchResult := struct {
		Result []entity.User `json:"result"`
		Page   int64         `json:"page"`
	}{}
	assert.Nil(t, json.Unmarshal(response.Body, &searchResult))
	assert.True(t, len(searchResult.Result) >= 1)
	assert.True(t, searchResult.Page != 0)

	// Make another call with a new Page (cursor)
	newCursor := strconv.Itoa(int(searchResult.Page))
	request = test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/user/search",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{"username": {"me."}, "cursor": {newCursor}},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
	}
	response = test.ExecuteAPITest(logger, t, mockRouter, &request)
	assert.Nil(t, json.Unmarshal(response.Body, &searchResult))
	assert.True(t, len(searchResult.Result) >= 1)
	assert.True(t, searchResult.Page == 0)
}
