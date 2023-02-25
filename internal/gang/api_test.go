// Gang API tests in Popcorn.

package gang

import (
	"Popcorn/internal/entity"
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

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

// Global instance of log.Logger to be used during gang API testing.
var logger log.Logger

// Global instance of gin MockRouter to be used during gang API testing.
var mockRouter *gin.Engine

// Global instance of Db instance to be used during gang API testing.
var client *db.RedisDB

// Global instance of user Repository to be used during gang API testing.
var userRepo user.Repository

// Global instance of gang Repository to be used during gang API testing.
var gangRepo Repository

// Global context
var ctx context.Context = context.Background()

type GangTestData struct {
	CreateGangInvalid map[string]*struct {
		Body *struct {
			Name    interface{} `json:"gang_name,omitempty"`
			PassKey interface{} `json:"gang_pass_key,omitempty"`
			Limit   interface{} `json:"gang_member_limit,omitempty"`
		} `json:"body,omitempty"`
		WantResponse []int `json:"response"`
	} `json:"create_gang_invalid"`

	CreateGangValid map[string]*struct {
		Body *struct {
			Name    interface{} `json:"gang_name,omitempty"`
			PassKey interface{} `json:"gang_pass_key,omitempty"`
			Limit   interface{} `json:"gang_member_limit,omitempty"`
		} `json:"body,omitempty"`
		WantResponse []int `json:"response"`
	} `json:"create_gang_valid"`

	JoinGangInvalid map[string]*struct {
		Body *struct {
			Admin   interface{} `json:"gang_admin,omitempty"`
			Name    interface{} `json:"gang_name,omitempty"`
			PassKey interface{} `json:"gang_pass_key,omitempty"`
		} `json:"body,omitempty"`
		WantResponse []int `json:"response"`
	} `json:"join_gang_invalid"`

	JoinGangValid map[string]*struct {
		Body *struct {
			Admin   interface{} `json:"gang_admin,omitempty"`
			Name    interface{} `json:"gang_name,omitempty"`
			PassKey interface{} `json:"gang_pass_key,omitempty"`
		} `json:"body,omitempty"`
		WantResponse []int `json:"response"`
	} `json:"join_gang_valid"`

	SearchGangInvalid map[string]*struct {
		Body     url.Values `json:"body,omitempty"`
		Response []int      `json:"response"`
	} `json:"search_gang_invalid"`

	SearchGangValid map[string]*struct {
		Body     url.Values `json:"body,omitempty"`
		Response []int      `json:"response"`
	} `json:"search_gang_valid"`

	GangList []entity.Gang `json:"gang_list"`
}

// GangTestData struct variable which stores unmarshalled all of the testdata for gang tests.
var testdata *GangTestData

// TestUser account to be used during gang API tests
var testUser entity.User

// TestUser Cookie to be passed during tests
var userCookie http.Cookie

// Helper to build up a mock router instance for testing Popcorn.
func setupMockRouter(dbConnWrp *db.RedisDB, logger log.Logger) {
	// Initializing mock router
	mockRouter = test.MockRouter()

	// Repositories needed by gang APIs and services to work
	userRepo = user.NewRepository(dbConnWrp)
	gangRepo = NewRepository(dbConnWrp)

	// Register internal package gang handler
	gangService := NewService(gangRepo, userRepo, logger)
	APIHandlers(mockRouter, gangService, test.MockAuthMiddleware(logger), logger)
}

// Helper to register list of gang to avoid repetition in tests below
func registerGangList() {
	// Register list of gangs from testdata/gang.json
	for _, testGang := range testdata.GangList {
		testGang := testGang
		testCookie := http.Cookie{
			Name:     "user",
			Value:    testGang.Admin,
			HttpOnly: true,
		}
		body, mrserr := json.Marshal(testGang)
		if mrserr != nil {
			logger.Fatal().Err(mrserr).Msg("Couldn't marshall JoinGangInvalid struct into json in TestJoinGangInvalid()")
		}

		request := test.RequestAPITest{
			Method:       http.MethodPost,
			Path:         "/api/gang/create",
			Body:         bytes.NewReader(body),
			WantResponse: []int{http.StatusOK},
			Header:       test.MockHeader(),
			Parameters:   url.Values{},
			Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testCookie},
		}
		test.ExecuteAPITest(logger, &testing.T{}, mockRouter, &request)
	}
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
	RegisterCustomValidations(ctx, logger)

	// Initializing router
	setupMockRouter(client, logger)

	// Read testdata and unmarshall into UserTestData
	datafilebytes, oserr := os.ReadFile("../../testdata/gang.json")
	if oserr != nil {
		// Error during readint testdata/user.json
		logger.Fatal().Err(oserr).Msg("Couldn't read testdata/gang.json, Aborting test run.")
	}
	mrsherr := json.Unmarshal(datafilebytes, &testdata)
	if mrsherr != nil {
		// Error during unmarshalling into UserTestData
		logger.Fatal().Err(mrsherr).Msg("Couldn't unmarshall into UserTestData, Aborting test run.")
	}

	// Setup a test user account to be used for gang API testing
	// Use user.SetOrUpdate repository method to set user data
	testUser = entity.User{
		Username: "me_Marta_Beard..23",
		FullName: "Marta Beard",
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
		Value:    "me_Marta_Beard..23",
		HttpOnly: true,
	}

	// Register list of gangs
	registerGangList()

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

func TestCreateGangInvalid(t *testing.T) {
	// Loop through every test scenarios defined in testdata/gang.json -> create_gang_invalid
	for name, data := range testdata.CreateGangInvalid {
		data := data
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			body, mrserr := json.Marshal(data.Body)
			if mrserr != nil {
				logger.Error().Err(mrserr).Msg("Couldn't marshall CreateGangInvalid struct into json in TestCreateGangInvalid()")
				t.Fatal()
			}

			request := test.RequestAPITest{
				Method:       http.MethodPost,
				Path:         "/api/gang/create",
				Body:         bytes.NewReader(body),
				WantResponse: data.WantResponse,
				Header:       test.MockHeader(),
				Parameters:   url.Values{},
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

func TestCreateGangValid(t *testing.T) {
	// Loop through every test scenarios defined in testdata/gang.json -> create_gang_valid
	for name, data := range testdata.CreateGangValid {
		data := data
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			body, mrserr := json.Marshal(data.Body)
			if mrserr != nil {
				logger.Error().Err(mrserr).Msg("Couldn't marshall CreateGangValid struct into json in TestCreateGangValid()")
				t.Fatal()
			}

			request := test.RequestAPITest{
				Method:       http.MethodPost,
				Path:         "/api/gang/create",
				Body:         bytes.NewReader(body),
				WantResponse: data.WantResponse,
				Header:       test.MockHeader(),
				Parameters:   url.Values{},
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)

			// Delete the created gangs so that it can be reusable in other tests
			gangRepo.DelGang(ctx, logger, testUser.Username)
		})
	}
}

func TestGetGang(t *testing.T) {
	// Make a call to get_gang API to fetch blank created or joined gang data of testUser
	request := test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/gang/get",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
	}
	response := test.ExecuteAPITest(logger, t, mockRouter, &request)
	// get_gang response structure
	gangData := struct {
		Gang      []entity.GangResponse `json:"gang"`
		CanCreate bool                  `json:"canCreateGang"`
		CanJoin   bool                  `json:"canJoinGang"`
	}{}
	assert.Nil(t, json.Unmarshal(response.Body, &gangData))
	assert.True(t, len(gangData.Gang) == 0)
	assert.True(t, gangData.CanCreate)
	assert.True(t, gangData.CanJoin)

	testGang := entity.Gang{
		Admin:   testUser.Username,
		Name:    "Test Gang",
		PassKey: "12345",
		Limit:   2,
	}
	_, dberr := gangRepo.SetOrUpdateGang(ctx, logger, &testGang)
	if dberr != nil {
		// Issues in SetOrUpdateGang()
		t.Fail()
	}
	// Make a call to get_gang API to fetch created gang data of testUser
	request = test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/gang/get",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
	}
	response = test.ExecuteAPITest(logger, t, mockRouter, &request)
	assert.Nil(t, json.Unmarshal(response.Body, &gangData))
	assert.True(t, len(gangData.Gang) == 1)
	assert.False(t, gangData.CanCreate)
	assert.True(t, gangData.CanJoin)
}

func TestGetGangInvites(t *testing.T) {
	// Make a call to get_gang_invites API to fetch testUser's invites list
	request := test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/gang/get/invites",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
	}
	response := test.ExecuteAPITest(logger, t, mockRouter, &request)
	// get_gang_invites response structure
	gangInvites := struct {
		Invites []entity.GangInvite `json:"invites"`
	}{}
	assert.Nil(t, json.Unmarshal(response.Body, &gangInvites))
	assert.True(t, len(gangInvites.Invites) == 0)
}

func TestGetGangMembers(t *testing.T) {
	// Make a call to get_gang_members API to fetch empty members list
	// As testUser isn't an admin of any gang
	request := test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/gang/get/gang_members",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
	}
	response := test.ExecuteAPITest(logger, t, mockRouter, &request)
	// get_gang_members response structure
	gangMembersList := struct {
		Members []entity.User `json:"members"`
	}{}
	assert.Nil(t, json.Unmarshal(response.Body, &gangMembersList))
	assert.True(t, len(gangMembersList.Members) == 0)
}

func TestJoinGangInvalid(t *testing.T) {
	// Loop through every test scenarios defined in testdata/gang.json -> join_gang_invalid
	for name, data := range testdata.JoinGangInvalid {
		data := data
		t.Run(name, func(t *testing.T) {
			body, mrserr := json.Marshal(data.Body)
			if mrserr != nil {
				logger.Error().Err(mrserr).Msg("Couldn't marshall JoinGangInvalid struct into json in TestJoinGangInvalid()")
				t.Fatal()
			}

			request := test.RequestAPITest{
				Method:       http.MethodPost,
				Path:         "/api/gang/join",
				Body:         bytes.NewReader(body),
				WantResponse: data.WantResponse,
				Header:       test.MockHeader(),
				Parameters:   url.Values{},
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

func TestJoinGangValid(t *testing.T) {
	// Loop through every test scenarios defined in testdata/gang.json -> join_gang_valid
	for name, data := range testdata.JoinGangValid {
		data := data
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			body, mrserr := json.Marshal(data.Body)
			if mrserr != nil {
				logger.Error().Err(mrserr).Msg("Couldn't marshall JoinGangValid struct into json in TestJoinGangValid()")
				t.Fatal()
			}

			request := test.RequestAPITest{
				Method:       http.MethodPost,
				Path:         "/api/gang/join",
				Body:         bytes.NewReader(body),
				WantResponse: data.WantResponse,
				Header:       test.MockHeader(),
				Parameters:   url.Values{},
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &userCookie},
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

func TestSearchGangInvalid(t *testing.T) {
	// Loop through every test scenarios defined in testdata/gang.json -> search_gang_invalid
	for subTestName, subTestBody := range testdata.SearchGangInvalid {
		subTestBody := subTestBody
		t.Run(subTestName, func(t *testing.T) {
			t.Parallel()
			request := test.RequestAPITest{
				Method:       http.MethodGet,
				Path:         "/api/gang/search",
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

func TestSearchGangValid(t *testing.T) {
	// Loop through every test scenarios defined in testdata/gang.json -> search_gang_valid
	for subTestName, subTestBody := range testdata.SearchGangValid {
		subTestBody := subTestBody
		t.Run(subTestName, func(t *testing.T) {
			t.Parallel()
			request := test.RequestAPITest{
				Method:       http.MethodGet,
				Path:         "/api/gang/search",
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
