// Exposes all of the REST APIs related to Gang creation in Popcorn.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Registers all of the REST API handlers related to internal package gang onto the gin server.
func GangHandlers(router *gin.Engine, service Service, AuthWithAcc gin.HandlerFunc, logger log.Logger) {
	gangGroup := router.Group("/api/gang")
	{
		gangGroup.POST("/create", AuthWithAcc, create_gang(service, logger))
		gangGroup.GET("/get", AuthWithAcc, get_gang(service, logger))
		gangGroup.GET("/get/invites", AuthWithAcc, get_gang_invites(service, logger))
		gangGroup.POST("/join", AuthWithAcc, join_gang(service, logger))
		gangGroup.GET("/search", AuthWithAcc, search_gang(service, logger))
	}
}

// create_gang returns a handler which takes care of creating gangs in Popcorn.
func create_gang(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var gang entity.Gang

		// Serialize received data into User struct
		if binderr := gctx.BindJSON(&gang); binderr != nil {
			// Error occured during serialization
			logger.WithCtx(gctx).Error().Err(binderr).Msg("Binding error occured with User struct.")
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}

		// Fetch username from context which will be used as the gang admin
		var ok bool = true
		gang.Admin, ok = gctx.Value("Username").(string)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in create_gang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}

		// Apply the service logic for Create Gang in Popcorn
		err := service.creategang(gctx, &gang)
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
		gctx.Status(http.StatusOK)
	}
}

// get_gang returns a handler which takes care of getting user created or joined gangs in Popcorn.
func get_gang(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used in getgang service
		username, ok := gctx.Value("Username").(string)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in get_gang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		data, canCreate, canJoin, err := service.getgang(gctx, username)
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
			"gang":          data,
			"canCreateGang": canCreate,
			"canJoinGang":   canJoin,
		})
	}
}

func get_gang_invites(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used in getganginvites service
		username, ok := gctx.Value("Username").(string)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in get_gang_invites")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		invites, err := service.getganginvites(gctx, username)

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
			"invites": invites,
		})
	}
}

// join_gang returns a handler which takes care of joining a gang in Popcorn.
func join_gang(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the joingang service
		username, ok := gctx.Value("Username").(string)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in get_gang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var gangKey entity.GangKey
		// Serialize received data into GangKey struct
		if binderr := gctx.BindJSON(&gangKey); binderr != nil {
			// Error occured during serialization
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		// Set gang-key which is of format gang:<gang_admin>
		gangKey.Key = "gang:" + gangKey.Admin
		err := service.joingang(gctx, username, gangKey)
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
		gctx.Status(http.StatusOK)
	}
}

// search_gang returns a handler which takes care of gang search in Popcorn.
func search_gang(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used in getgang service
		username, ok := gctx.Value("Username").(string)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in get_gang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}

		var query entity.GangSearch
		gang_name := gctx.DefaultQuery("gang_name", "")
		cursor, converr := strconv.Atoi(gctx.DefaultQuery("cursor", "0"))
		if converr != nil || cursor < 0 || cursor > 1000 {
			// Invalid cursor input
			gctx.Status(http.StatusBadRequest)
			return
		}
		// bind data into query struct
		query.Name = gang_name
		query.Cursor = cursor

		response, newCursor, err := service.searchgang(gctx, query, username)
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
