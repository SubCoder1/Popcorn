// Exposes all of the REST APIs related to User Model in Popcorn.

package user

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Registers all of the REST API handlers related to internal package user onto the gin server.
func APIHandlers(router *gin.Engine, service Service, AuthWithAcc gin.HandlerFunc, logger log.Logger) {
	userGroup := router.Group("/api/user")
	{
		userGroup.GET("/get", AuthWithAcc, getUser(service, logger))
		userGroup.GET("/search", AuthWithAcc, searchUser(service, logger))
	}
}

// getUser returns a handler which takes care of getting user details in Popcorn.
func getUser(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used in searchuser service
		username, ok := gctx.Value("Username").(string)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in get_gang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		// Apply the service logic for Get User in Popcorn
		user, err := service.getuser(gctx, username)
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

// searchUser returns a handler which takes care of user search in Popcorn.
func searchUser(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used in searchuser service
		username, ok := gctx.Value("Username").(string)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in get_gang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var query entity.UserSearch
		request_username := gctx.DefaultQuery("username", "")
		request_cursor, converr := strconv.Atoi(gctx.DefaultQuery("cursor", "0"))
		if converr != nil || request_cursor < 0 || request_cursor > 1000 {
			// Invalid cursor input
			gctx.Status(http.StatusBadRequest)
			return
		}
		// bind data into query struct
		query.Username = request_username
		query.Cursor = request_cursor

		response, newCursor, err := service.searchuser(gctx, query, username)
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
			"result": response,
			"page":   newCursor,
		})
	}
}
