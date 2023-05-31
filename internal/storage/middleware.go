// Middlewares needed by tus content handling service are defined here.

package storage

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/internal/gang"
	"Popcorn/pkg/log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// As only gang admin can do anything regarding content (upload / update / delete),
// This middleware is needed to validate incoming tus requests.
func ValidateGangAdminMiddleware(logger log.Logger, gangRepo gang.Repository) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the gang admin
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in createGang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		gangKey := "gang:" + user.Username
		available, dberr := gangRepo.HasGang(ctx, logger, gangKey, "")
		if dberr != nil || !available {
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if gctx.Request.Method == "DELETE" {
			// Erase content ID and filename from DB
			defer func() {
				gangRepo.UpdateGangContentData(gctx, logger, user.Username, "", "", false)
			}()
		}
		gctx.Request.Header.Add("User", user.Username) // to be used in tusd callbacks
		gctx.Next()
	}
}
