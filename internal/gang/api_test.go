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
}

// GangTestData struct variable which stores unmarshalled all of the testdata for gang tests.
var testdata *GangTestData

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
	// Use user.SetOrUpdate repository method to set user data
	testUser := entity.User{
		Username: "me_Marta_Beard..23",
		FullName: "Marta Beard",
		Password: "popcorn123",
	}
	testUser.SelectProfilePic()
	_, dberr := userRepo.SetOrUpdateUser(ctx, logger, testUser, true)
	if dberr != nil {
		// Issues in SetOrUpdateUser()
		t.Fail()
	}
	userCookie := http.Cookie{
		Name:     "user",
		Value:    "me_Marta_Beard..23",
		HttpOnly: true,
	}
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
	// Use user.SetOrUpdate repository method to set user data
	testUser := entity.User{
		Username: "me_Ruben_Stone..23",
		FullName: "Ruben Stone",
		Password: "popcorn123",
	}
	testUser.SelectProfilePic()
	_, dberr := userRepo.SetOrUpdateUser(ctx, logger, testUser, true)
	if dberr != nil {
		// Issues in SetOrUpdateUser()
		t.Fail()
	}
	userCookie := http.Cookie{
		Name:     "user",
		Value:    "me_Ruben_Stone..23",
		HttpOnly: true,
	}
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

			// Delete the created gang else next test will return 400
			gangRepo.DelGang(ctx, logger, "me_Ruben_Stone..23")
		})
	}
}
