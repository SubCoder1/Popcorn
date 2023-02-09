package test

import (
	"Popcorn/pkg/log"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Format of Request helper TestAPI() handles
type RequestAPITest struct {
	Method       string         // Method of API request - [GET, POST, PUT, DELETE . . .]
	Path         string         // API Path
	Body         *bytes.Reader  // Request Body
	WantResponse []int          // Expected Response according to request
	Header       http.Header    // Request headers
	Parameters   url.Values     // Query parameters
	Cookie       []*http.Cookie // Request Cookies
}

// Response attached with every APITest
type APIResponse struct {
	Header http.Header    // Response headers
	Cookie []*http.Cookie // Response cookies
	Body   []byte         // Response body in []bytes unmarshallable to specific struct
}

// Helper to attach a mock header
func MockHeader() http.Header {
	header := http.Header{}
	header.Add("Content-Type", "application/json")
	return header
}

// Helper to execute API tests in Popcorn.
func ExecuteAPITest(logger log.Logger, t *testing.T, router *gin.Engine, request *RequestAPITest) APIResponse {
	// Setup the test request
	req, reqerr := http.NewRequest(request.Method, request.Path, request.Body)
	if reqerr != nil {
		// Error in NewRequest
		logger.Error().Err(reqerr).Msg("Error occured during calling NewRequest in ExecuteAPITest()")
		t.Error()
		return APIResponse{}
	}
	// Attach headers, cookies & query paramters before calling ServeHTTP
	req.Header = request.Header
	req.URL.RawQuery = request.Parameters.Encode()
	for _, cookie := range request.Cookie {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert the response
	assert.Contains(t, request.WantResponse, w.Code)

	// Attach Response Body
	response := w.Result()
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)

	return APIResponse{
		Header: response.Header.Clone(),
		Cookie: response.Cookies(),
		Body:   data,
	}
}
