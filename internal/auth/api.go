// Exposes all of the REST APIs related to User authentication in Popcorn.

package auth

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var domain string = os.Getenv("SRV_ADDR")

// Registers all of the REST API handlers related to internal package auth onto the gin server.
func APIHandlers(router *gin.Engine, service Service, AuthWithAcc gin.HandlerFunc, AuthWithRef gin.HandlerFunc, logger log.Logger) {
	authGroup := router.Group("/api/auth")
	{
		authGroup.GET("/validate_token", AuthWithAcc)
		authGroup.POST("/register", register(service, logger))
		authGroup.POST("/login", login(service, logger))
		authGroup.POST("/logout", AuthWithAcc, AuthWithRef, logout(service, logger))
		authGroup.POST("/refresh_token", AuthWithRef, refresh_token(service, logger))
	}
}

// register returns a handler which takes care of user registration in Popcorn.
func register(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var user entity.User

		// Serialize received data into User struct
		if binderr := gctx.ShouldBindJSON(&user); binderr != nil {
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

		// Registration successful, Add the jwt in request's cookie with httpOnly as true
		access_token_cookie := &http.Cookie{
			Name:     "access_token",
			Value:    token["access_token"].(string),
			Expires:  token["access_token_exp"].(time.Time),
			MaxAge:   token["access_token_maxAge"].(int),
			Domain:   domain,
			Path:     "/api",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(gctx.Writer, access_token_cookie)
		refresh_token_cookie := &http.Cookie{
			Name:     "refresh_token",
			Value:    token["refresh_token"].(string),
			Expires:  token["refresh_token_exp"].(time.Time),
			MaxAge:   token["refresh_token_maxAge"].(int),
			Domain:   domain,
			Path:     "/api",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(gctx.Writer, refresh_token_cookie)

		gctx.Status(http.StatusOK)
	}
}

// login returns a handler which takes care of user login in Popcorn.
func login(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var user entity.UserLogin

		// Serialize received data into User struct
		if binderr := gctx.ShouldBindJSON(&user); binderr != nil {
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

		// login successful, Add the jwt in request's cookie with httpOnly as true
		access_token_cookie := &http.Cookie{
			Name:     "access_token",
			Value:    token["access_token"].(string),
			Expires:  token["access_token_exp"].(time.Time),
			MaxAge:   token["access_token_maxAge"].(int),
			Domain:   domain,
			Path:     "/api",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(gctx.Writer, access_token_cookie)
		refresh_token_cookie := &http.Cookie{
			Name:     "refresh_token",
			Value:    token["refresh_token"].(string),
			Expires:  token["refresh_token_exp"].(time.Time),
			MaxAge:   token["refresh_token_maxAge"].(int),
			Domain:   domain,
			Path:     "/api",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(gctx.Writer, refresh_token_cookie)

		gctx.Status(http.StatusOK)
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
		// Delete access_token cookie from client's header
		access_token_cookie := &http.Cookie{
			Name:     "access_token",
			Value:    "",
			Expires:  time.Now(),
			MaxAge:   0,
			Domain:   domain,
			Path:     "/api",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(gctx.Writer, access_token_cookie)
		refresh_token_cookie := &http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Expires:  time.Now(),
			MaxAge:   0,
			Domain:   domain,
			Path:     "/api",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(gctx.Writer, refresh_token_cookie)

		gctx.Status(http.StatusOK)
	}
}

// refresh_token returns a handler which takes care of refreshing JWT for users in Popcorn.
// Incoming request should pass AuthMiddleware in order for this handler to work.
func refresh_token(service Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch Username from context
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in refresh_token")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		// Generate fresh pair of JWT for user
		token, err := service.refreshtoken(gctx, user.Username)
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

		// Refresh successful, Add the jwt in request's cookie with httpOnly as true
		access_token_cookie := &http.Cookie{
			Name:     "access_token",
			Value:    token["access_token"].(string),
			Expires:  token["access_token_exp"].(time.Time),
			MaxAge:   token["access_token_maxAge"].(int),
			Domain:   domain,
			Path:     "/api",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(gctx.Writer, access_token_cookie)
		refresh_token_cookie := &http.Cookie{
			Name:     "refresh_token",
			Value:    token["refresh_token"].(string),
			Expires:  token["refresh_token_exp"].(time.Time),
			MaxAge:   token["refresh_token_maxAge"].(int),
			Domain:   domain,
			Path:     "/api",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(gctx.Writer, refresh_token_cookie)

		gctx.Status(http.StatusOK)
	}
}
