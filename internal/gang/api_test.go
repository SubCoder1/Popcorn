// Gang API tests in Popcorn.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/metrics"
	"Popcorn/internal/sse"
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
	"strconv"
	"sync"
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

// Global instance of metrics Repository to be used during metrics API testing.
var metricsRepo metrics.Repository

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
		Body         url.Values `json:"body,omitempty"`
		WantResponse []int      `json:"response"`
	} `json:"search_gang_invalid"`

	SearchGangValid map[string]*struct {
		Body         url.Values `json:"body,omitempty"`
		WantResponse []int      `json:"response"`
	} `json:"search_gang_valid"`

	GangInviteInvalid map[string]*struct {
		Body *struct {
			Name interface{} `json:"gang_name,omitempty"`
			For  interface{} `json:"gang_invite_for,omitempty"`
		} `json:"body,omitempty"`
		WantResponse []int `json:"response"`
	} `json:"gang_invite_invalid"`

	GangList []entity.Gang `json:"gang_list"`
}

// GangTestData struct variable which stores unmarshalled all of the testdata for gang tests.
var testdata *GangTestData

// TestUser account to be used during gang API tests
var testUser entity.User

// TestUser Cookie to be passed during tests
var testUserCookie http.Cookie

// Helper to build up a mock router instance for testing Popcorn.
func setupMockRouter(dbConnWrp *db.RedisDB, logger log.Logger) {
	// Initializing mock router
	mockRouter = test.MockRouter()

	// Initializing livekit mock config
	livekitMockConfig := entity.LivekitConfig{
		Host:      "ws://localhost:8000",
		ApiKey:    "LivekitAPI",
		ApiSecret: "LivekitAPISecret",
	}

	// Repositories needed by gang APIs and services to work
	userRepo = user.NewRepository(dbConnWrp)
	gangRepo = NewRepository(dbConnWrp)
	metricsRepo = metrics.NewRepository(dbConnWrp)

	// Register internal package gang handler
	sseService := sse.NewService(logger)
	metricsService := metrics.NewService(livekitMockConfig, metricsRepo, logger)
	gangService := NewService(livekitMockConfig, gangRepo, userRepo, sseService, metricsService, logger)
	APIHandlers(mockRouter, gangService, test.MockAuthMiddleware(logger), logger)
}

// Helper to register list of gang to avoid repetition in tests below
func registerGangList() {
	var wg sync.WaitGroup
	// Register list of gangs from testdata/gang.json
	for _, testGang := range testdata.GangList {
		testGang := testGang
		wg.Add(1)
		go func() {
			defer wg.Done()
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
		}()
	}
	wg.Wait()
}

// Helper to register test user required in the tests below
func registerTestUser(username, fullname string) (entity.User, http.Cookie) {
	// Use user.SetOrUpdate repository method to set user data
	testUser := entity.User{
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
	testUserCookie := http.Cookie{
		Name:     "user",
		Value:    username,
		HttpOnly: true,
	}

	return testUser, testUserCookie
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
	user.RegisterCustomValidationTags(ctx, logger)
	RegisterCustomValidationTags(ctx, logger)

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
	testUser, testUserCookie = registerTestUser("me_Marta_Beard..23", "Marta Beard")

	// Register list of gangs
	registerGangList()

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
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
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
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
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
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
	}
	response := test.ExecuteAPITest(logger, t, mockRouter, &request)
	// get_gang response structure
	gangData := map[string]interface{}{}
	assert.Nil(t, json.Unmarshal(response.Body, &gangData))

	testGang := entity.Gang{
		Admin:   testUser.Username,
		Name:    "Test Gang",
		PassKey: "12345",
		Limit:   2,
	}
	_, dberr := gangRepo.SetOrUpdateGang(ctx, logger, &testGang, false)
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
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
	}
	response = test.ExecuteAPITest(logger, t, mockRouter, &request)
	assert.Nil(t, json.Unmarshal(response.Body, &gangData))
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
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
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
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
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
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
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
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
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
				WantResponse: subTestBody.WantResponse,
				Header:       test.MockHeader(),
				Parameters:   subTestBody.Body,
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
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
				WantResponse: subTestBody.WantResponse,
				Header:       test.MockHeader(),
				Parameters:   subTestBody.Body,
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

func TestSearchGangPaginated(t *testing.T) {
	// As every gang registered in setup() starts with "My"
	// Make a call to SearchGang with "My" as search query
	// Expected response, 200 with a new cursor (pagination)
	request := test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/gang/search",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{"gang_name": {"My"}},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
	}
	response := test.ExecuteAPITest(logger, t, mockRouter, &request)
	searchResult := struct {
		Result []entity.GangResponse `json:"result"`
		Page   int64                 `json:"page"`
	}{}
	assert.Nil(t, json.Unmarshal(response.Body, &searchResult))
	assert.True(t, len(searchResult.Result) >= 1)
	assert.True(t, searchResult.Page != 0)

	// Make another call with a new Page (cursor)
	newCursor := strconv.Itoa(int(searchResult.Page))
	request = test.RequestAPITest{
		Method:       http.MethodGet,
		Path:         "/api/gang/search",
		Body:         bytes.NewReader([]byte{}),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{"gang_name": {"My"}, "cursor": {newCursor}},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
	}
	response = test.ExecuteAPITest(logger, t, mockRouter, &request)
	assert.Nil(t, json.Unmarshal(response.Body, &searchResult))
	assert.True(t, len(searchResult.Result) >= 1)
	assert.True(t, searchResult.Page == 0)
}

// Send / Accept / Reject Gang Invite invalid test is same as that of TestSendGangInviteInvalid
// As they use the same entity.GangInvite struct with same validations
func TestGangInviteInvalid(t *testing.T) {
	// Loop through every test scenarios defined in testdata/gang.json -> gang_invite_invalid
	for subTestName, subTestBody := range testdata.GangInviteInvalid {
		subTestBody := subTestBody
		body, mrserr := json.Marshal(subTestBody.Body)
		if mrserr != nil {
			logger.Error().Err(mrserr).Msg("Couldn't marshall GangInviteInvalid struct into json in TestSendGangInviteInvalid()")
			t.Fatal()
		}
		t.Run(subTestName, func(t *testing.T) {
			t.Parallel()
			request := test.RequestAPITest{
				Method:       http.MethodPost,
				Path:         "/api/gang/send_invite",
				Body:         bytes.NewReader(body),
				WantResponse: subTestBody.WantResponse,
				Header:       test.MockHeader(),
				Parameters:   url.Values{},
				Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
			}
			test.ExecuteAPITest(logger, t, mockRouter, &request)
		})
	}
}

// Send / Accept / Reject / Gang Invite valid test and Boot gang member test
// Merged all of the test as all requires creating a gang first
func TestGangInviteBoot(t *testing.T) {
	// Delete any gang created by testUser during the execution of above tests
	gangRepo.DelGang(ctx, logger, testUser.Username)
	// Create a gang for testUser
	testGang := entity.Gang{
		Name:    "My Gang 123",
		PassKey: "12345",
		Limit:   2,
	}
	body, mrserr := json.Marshal(testGang)
	if mrserr != nil {
		logger.Error().Err(mrserr).Msg("Couldn't marshall Gang struct into json in TestSendGangInviteValid()")
		t.Fatal()
	}

	request := test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/gang/create",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)

	// Create a temp user who's going to receive the gang invite
	_, tempUserCookie := registerTestUser("Temp_User123", "Temp User")

	// Send invite
	testInvite := entity.GangInvite{
		Admin: testUser.Username,
		Name:  "My Gang 123",
		For:   "Temp_User123",
	}
	body, mrserr = json.Marshal(testInvite)
	if mrserr != nil {
		logger.Error().Err(mrserr).Msg("Couldn't marshall GangInvite struct into json in TestSendGangInviteValid()")
		t.Fatal()
	}
	request = test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/gang/send_invite",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)

	// Reject Invite
	request = test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/gang/reject_invite",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &tempUserCookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)

	// Send another invite to test request accept
	request = test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/gang/send_invite",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)

	// Accept Invite
	request = test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/gang/accept_invite",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &tempUserCookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)

	// Boot member (Invalid)
	testBootInvalidGangName := entity.GangExit{
		Member: "Temp_User123",
		Name:   testGang.Name + "invaliD",
	}
	body, mrserr = json.Marshal(testBootInvalidGangName)
	if mrserr != nil {
		logger.Error().Err(mrserr).Msg("Couldn't marshall GangExit struct into json in TestGangInvite()")
		t.Fatal()
	}

	request = test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/gang/boot_member",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusBadRequest},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)

	testBootInvalidUsername := entity.GangExit{
		Member: "Temp_User_1",
		Name:   testGang.Name,
	}

	body, mrserr = json.Marshal(testBootInvalidUsername)
	if mrserr != nil {
		logger.Error().Err(mrserr).Msg("Couldn't marshall GangExit struct into json in TestGangInvite()")
		t.Fatal()
	}

	request = test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/gang/boot_member",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusBadRequest},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)

	// Boot member (Valid)
	testBootValid := entity.GangExit{
		Member: "Temp_User123",
		Name:   testGang.Name,
	}
	body, mrserr = json.Marshal(testBootValid)
	if mrserr != nil {
		logger.Error().Err(mrserr).Msg("Couldn't marshall GangExit struct into json in TestGangInvite()")
		t.Fatal()
	}

	request = test.RequestAPITest{
		Method:       http.MethodPost,
		Path:         "/api/gang/boot_member",
		Body:         bytes.NewReader(body),
		WantResponse: []int{http.StatusOK},
		Header:       test.MockHeader(),
		Parameters:   url.Values{},
		Cookie:       []*http.Cookie{test.MockAuthAllowCookie, &testUserCookie},
	}
	test.ExecuteAPITest(logger, t, mockRouter, &request)
}
