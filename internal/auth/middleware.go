// Auth middleware is used to validate JWT token sent via header.
// This verification is needed for endpoints which needs authenticated users.

package auth

import (
	"Popcorn/internal/errors"
	"Popcorn/internal/user"
	"Popcorn/pkg/log"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// This middleware is used to verify and validate incoming JWT, TokenType can either be "access_token" or "refresh_token".
// Access-Secret and Refresh-Secret will be used to parse access_token and refresh_token respectively.
// Blocks the request to go further into other handlers if token is invalid.
func AuthMiddleware(logger log.Logger, authRepo Repository, userRepo user.Repository, tokenType string, secret string) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Extract token from header
		token := fetchTokenFromCookie(gctx, logger, tokenType)
		// Parse the token header with secret if the token is valid
		vrftoken, valerr := parseIntoJWT(gctx, logger, secret, token)
		if valerr != nil {
			// Abort the call chain for the request here as the user is unauthenticated
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Check the parsed token for validity
		if !vrftoken.Valid {
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Extract TokenUUID and UserID from token claims
		tokenclaims, ok := vrftoken.Claims.(jwt.MapClaims)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error during asserting jwt.Claims to jwt.MapClaims in AuthMiddleware")
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		tokenUUID, ok := tokenclaims[tokenType+"_uuid"].(string)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error during asserting jwt.MapClaims to string in AuthMiddleware")
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Successfully saved UserID is stored in float64 format even though type uint64 is passed during signing
		// Need to convert to uint64 later
		username, ok := tokenclaims["username"].(string)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error during asserting jwt.MapClaims to string in AuthMiddleware")
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Verify if TokenUUID:UserID is available in DB
		valid, dberr := authRepo.HasToken(gctx, logger, tokenUUID, username)
		if dberr != nil {
			// Error in TokenExists
			gctx.AbortWithStatus(http.StatusInternalServerError)
			return
		} else if !valid {
			// token missing in DB or mismatch with UserID
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// In case of tokenType = "refresh_token", delete the previous refresh_token first
		if tokenType == "refresh_token" {
			dberr = authRepo.DelToken(gctx, logger, tokenUUID)
			if dberr != nil {
				// Error in DelToken
				err, ok := dberr.(errors.ErrorResponse)
				if !ok || err.Status != 404 {
					// Error during DB interaction
					gctx.AbortWithStatus(http.StatusInternalServerError)
				}
				// Maybe the key wasn't present in the DB at all
				gctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		}
		// Set User in request's context
		// This object will be used further down in the handler chain
		user, dberr := userRepo.GetUser(gctx, logger, username)
		if dberr != nil {
			// Error during DB interaction
			gctx.AbortWithStatus(http.StatusInternalServerError)
		}
		gctx.Set("User", user)
		// Set User's accessToken which might be useful during logout
		if tokenType == "access_token" {
			gctx.Set("access_token", tokenUUID)
		}
		gctx.Next()
	}
}

// Helper to fetch token string from Header.
func fetchTokenFromCookie(gctx *gin.Context, logger log.Logger, tokenType string) string {
	var token *http.Cookie
	var err error
	if tokenType == "access_token" {
		token, err = gctx.Request.Cookie("access_token")
	} else {
		token, err = gctx.Request.Cookie("refresh_token")
	}
	if err != nil {
		logger.WithCtx(gctx).Error().Err(err).Msg("Error occured during fetching token from Cookie")
		return ""
	}
	return token.Value
}

// Helper to parse and return token string fetched from header.
// secret can be either Access-Secret for accessToken parsing or Refresh-Secret for refreshToken.
func parseIntoJWT(gctx *gin.Context, logger log.Logger, secret string, token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		// Check the signing method
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			err := errors.New(fmt.Sprintf("Unexpected signing method found: %s", t.Header["alg"]))
			return nil, err
		}
		return []byte(secret), nil
	})
}
