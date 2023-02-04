package test

import (
	"Popcorn/pkg/log"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Format of Request helper TestAPI() handles
type RequestAPITest struct {
	Method       string            // Method of API request - [GET, POST, PUT, DELETE . . .]
	Path         string            // API Path
	Body         *bytes.Reader     // Request Body
	WantResponse []int             // Expected Response according to request
	Headers      map[string]string // Request headers
}

// Helper to execute API tests in Popcorn.
func ExecuteAPITest(logger log.Logger, t *testing.T, router *gin.Engine, request RequestAPITest) {
	// Setup the test request
	req, reqerr := http.NewRequest(request.Method, request.Path, request.Body)
	for key, val := range request.Headers {
		req.Header.Set(key, val)
	}
	if reqerr != nil {
		// Error in NewRequest
		logger.Error().Err(reqerr).Msg("Error occured during calling NewRequest in TestRegister()")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// t.Log(w.Body)
	// Assert the response
	assert.Contains(t, request.WantResponse, w.Code)
}
