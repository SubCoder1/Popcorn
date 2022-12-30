// Exposes all of the REST APIs related to User authentication in Popcorn.

package auth

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	logger "Popcorn/pkg/log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Registers all of the REST API handlers related to internal package auth onto the gin server.
func RegisterAUTHHandlers(router *gin.Engine, service Service, dbwrp *db.RedisDB) {
	authgroup := router.Group("/auth")
	{
		authgroup.POST("/register", register(service, dbwrp))
	}
}

// register returns a handler which handles user registration in Popcorn.
func register(service Service, dbwrp *db.RedisDB) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var user entity.User

		// Serialize received data into User struct
		if binderr := gctx.BindJSON(&user); binderr != nil {
			// Error occured during serialization: status - 500
			logger.Logger.Error().Err(binderr).Msg("Binding error occured with User struct.")
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}

		// Apply the service logic for User registration in Popcorn
		token, err := service.Register(gctx.Request.Context(), dbwrp, user)
		if err != nil {
			// Error occured, might be validation or server error
			err := err.(errors.ErrorResponse)
			gctx.JSON(err.Status, err)
			return
		}

		// Registration successful, send the JWT token as a response
		gctx.JSON(http.StatusOK, gin.H{"token": token})
	}
}
