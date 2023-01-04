// Auth middleware is used to validate JWT token sent via header.
// This verification is needed for endpoints which needs authenticated users.

package auth

import (
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// Client's JWT metadata
type JWTMetaData struct {
	AccessTokenUUID string `json:"access_token_uuid"`
	UserID          uint64 `json:"user_id"`
}

func AuthMiddleware(logger log.Logger, authrepo Repository, accSecret string) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Extract Access token from header
		token := func() string {
			tokenheader := gctx.Request.Header.Get("Authorization")
			// Token is in form Authorization: Bearer <token>
			bearertoken := strings.Split(tokenheader, " ")
			if len(bearertoken) == 2 {
				return bearertoken[1]
			}
			return ""
		}()
		// Parse the token header with AccessSecret if the token is valid
		vrftoken, valerr := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			// Check the signing method
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				err := errors.New(fmt.Sprintf("Unexpected signing method found: %s", t.Header["alg"]))
				logger.WithCtx(gctx).Error().Err(err)
				return nil, err
			}
			return []byte(accSecret), nil
		})
		if valerr != nil {
			// Abort the call chain for the request here as the user is unauthenticated
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Check the parsed token for validity
		if _, ok := vrftoken.Claims.(jwt.Claims); !ok || !vrftoken.Valid {

			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Extract AccessTokenUUID and UserID from token claims
		tokenclaims, ok := vrftoken.Claims.(jwt.MapClaims)
		if !ok {
			// Type assertion error
			gctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		AccessTokenUUID, ok := tokenclaims["access_token_uuid"]
		if !ok {
			// Type assertion error
			gctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		// Successfully saved UserID is getting stored in float64 format
		// Need to convert to uint64 later
		UserID, ok := tokenclaims["user_id"].(float64)
		if !ok {
			// Type assertion error
			gctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		// Verify if AccessTokenUUID:UserID is available in DB
		valid, dberr := authrepo.TokenExists(gctx, logger, AccessTokenUUID.(string), uint64(UserID))
		if dberr != nil {
			gctx.AbortWithStatus(http.StatusInternalServerError)
			return
		} else if !valid {
			// token missing in DB or mismatch with UserID
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Set UserID in the request's context
		gctx.Set("UserID", UserID)
	}
}
