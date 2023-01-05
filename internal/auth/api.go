// Exposes all of the REST APIs related to User authentication in Popcorn.

package auth

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Registers all of the REST API handlers related to internal package auth onto the gin server.
func AuthHandlers(router *gin.Engine, service Service, AuthWithAcc gin.HandlerFunc, AuthWithRef gin.HandlerFunc, logger log.Logger) {
	authgroup := router.Group("/api/auth")
	{
		authgroup.POST("/register", register(service, logger))
		authgroup.POST("/login", login(service, logger))
		authgroup.POST("/logout", AuthWithAcc, logout(service, logger))
		authgroup.POST("/refresh_token", AuthWithRef, refresh_token(service, logger))
	}
}

// register returns a handler which takes care of user registration in Popcorn.
func register(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var user entity.User

		// Serialize received data into User struct
		if binderr := gctx.BindJSON(&user); binderr != nil {
			// Error occured during serialization
			logger.WithCtx(gctx).Error().Err(binderr).Msg("Binding error occured with User struct.")
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}

		// Apply the service logic for User registration in Popcorn
		token, err := service.register(gctx, user)
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

		// Registration successful, send the JWT as a response
		gctx.JSON(http.StatusOK, token)
	}
}

// login returns a handler which takes care of user login in Popcorn.
func login(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var user entity.User

		// Serialize received data into User struct
		if binderr := gctx.BindJSON(&user); binderr != nil {
			// Error occured during serialization
			logger.WithCtx(gctx).Error().Err(binderr).Msg("Binding error occured with User struct.")
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}

		// Apply the service logic for User login in Popcorn
		token, err := service.login(gctx, user)
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

		// login successful, send the JWT as a response
		gctx.JSON(http.StatusOK, token)
	}
}

// Logout returns a handler which takes care of user logout from Popcorn.
func logout(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		err := service.logout(gctx)
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

// refresh_token returns a handler which takes care of refreshing JWT for users in Popcorn.
// Incoming request should pass AuthMiddleware in order for this handler to work.
func refresh_token(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch UserID from context
		userID, ok := gctx.Value("UserID").(uint64)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in refresh_token")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		// Generate fresh pair of JWT for user
		token, err := service.refreshtoken(gctx, userID)
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

		// Successfully created a new JWT for user
		gctx.JSON(http.StatusOK, token)
	}
}
