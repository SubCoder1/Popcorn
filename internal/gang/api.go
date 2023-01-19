// Exposes all of the REST APIs related to Gang creation in Popcorn.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Registers all of the REST API handlers related to internal package gang onto the gin server.
func GangHandlers(router *gin.Engine, service Service, AuthWithAcc gin.HandlerFunc, logger log.Logger) {
	gangGroup := router.Group("/api/gang")
	{
		gangGroup.POST("/create", AuthWithAcc, create_gang(service, logger))
	}
}

// createGang returns a handler which takes care of creating gangs in Popcorn.
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
		gctx.JSON(http.StatusOK, gin.H{
			"gang": gang,
		})
	}
}
