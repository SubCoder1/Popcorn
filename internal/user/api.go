// Exposes all of the REST APIs related to User Model in Popcorn.

package user

import (
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Registers all of the REST API handlers related to internal package auth onto the gin server.
func APIHandlers(router *gin.Engine, service Service, AuthWithAcc gin.HandlerFunc, logger log.Logger) {
	usergroup := router.Group("/api/user")
	{
		usergroup.GET("/get", AuthWithAcc, getUser(service, logger))
	}
}

// getUser returns a handler which takes care of getting user details in Popcorn.
// requires auth to access.
func getUser(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Apply the service logic for Get User in Popcorn
		user, err := service.getuser(gctx)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
			return
		}
		gctx.JSON(http.StatusOK, gin.H{
			"user": user,
		})
	}
}
