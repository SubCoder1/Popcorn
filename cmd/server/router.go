// List of all REST API endpoints being used by Popcorn can be found here.

package main

import (
	"Popcorn/internal/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Router(router *gin.Engine) {
	// This is the route to default path.
	router.GET("/", func(gctx *gin.Context) {
		gctx.String(http.StatusOK, "Welcome to Popcorn!")
	})

	// User related route groups.
	usergroup := router.Group("/user")
	{
		// Route to register users on Popcorn
		usergroup.POST("/register", user.RegisterUser())
	}
}
