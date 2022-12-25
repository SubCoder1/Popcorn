// List of all REST API endpoints being used by Popcorn can be found here.

package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Router(router *gin.Engine) {
	// This is the route to default path
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to Popcorn!")
	})
}
