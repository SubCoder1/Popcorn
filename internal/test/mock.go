// Mock methods required in Popcorn tests are all here.

package test

import (
	"Popcorn/pkg/log"
	"Popcorn/pkg/middlewares"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

// Global instance of gin MockRouter to be used during API testing.
var testRouter *gin.Engine

// Singleton to make sure testRouter is initialized only once.
var once sync.Once

func MockRouter() *gin.Engine {
	once.Do(func() {
		// Initializing the gin test server
		ginMode := os.Getenv("GIN_MODE")
		gin.SetMode(ginMode)
		testRouter = gin.Default()
		testRouter.Use(middlewares.CORSMiddleware("*")) // CORS middleware which allows request from all origin
	})
	return testRouter
}

// Cookie to be used in tests to bypass MockAuthMiddleware
var MockAuthAllowCookie *http.Cookie = &http.Cookie{
	Name:     "mode",
	Value:    "test",
	HttpOnly: true,
}

func MockAuthMiddleware(logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		token, err := gctx.Request.Cookie("mode")
		if err != nil {
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		} else if token.Value != "test" {
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		user, err := gctx.Request.Cookie("user")
		if err != nil {
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Set Username in request's context
		// This pair will be used further down in the handler chain
		gctx.Set("Username", user.Value)
		gctx.Next()
	}
}
