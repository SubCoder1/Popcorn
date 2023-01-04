// Exposes all of the REST APIs related to User authentication in Popcorn.

package auth

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Registers all of the REST API handlers related to internal package auth onto the gin server.
func RegisterAUTHHandlers(router *gin.Engine, service Service, logger log.Logger) {
	authgroup := router.Group("/api/auth")
	{
		authgroup.POST("/register", register(service, logger))
	}
}

// register returns a handler which handles user registration in Popcorn.
func register(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Adding unique ID to the client's context
		// Useful for debugging as logger is setup to fetch this ID
		rqId, uuiderr := uuid.NewRandom()
		if uuiderr != nil {
			logger.Error().Err(uuiderr).Msg("Error during generating UUID for ReqID.")
		} else {
			gctx.Set("ReqID", rqId.String())
		}

		var user entity.User

		// Serialize received data into User struct
		if binderr := gctx.BindJSON(&user); binderr != nil {
			// Error occured during serialization: status - 500
			logger.WithCtx(gctx).Error().Err(binderr).Msg("Binding error occured with User struct.")
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}

		// Apply the service logic for User registration in Popcorn
		token, err := service.Register(gctx, user)
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

		// Registration successful, send the JWT token as a response
		gctx.JSON(http.StatusOK, token)
	}
}
