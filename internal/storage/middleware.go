// Middlewares needed by tus content handling service are defined here.

package storage

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/internal/gang"
	"Popcorn/internal/metrics"
	"Popcorn/pkg/log"
	"context"
	"net/http"
	"strconv"
	"syscall"

	"github.com/gin-gonic/gin"
)

// As only gang admin can do anything regarding content (upload / update / delete),
// This middleware is needed to validate incoming tus requests.
func ContentStorageMiddleware(logger log.Logger, livekit_config entity.LivekitConfig, metricsService metrics.Service, gangRepo gang.Repository) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the gang admin
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in ContentStorageMiddleware")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}

		metrics, dberr := metricsService.GetMetrics(gctx)
		if dberr != nil {
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		} else if metrics.IngressQuotaExceeded || metrics.ActiveIngress+1 >= livekit_config.MaxConcurrentIngressLimit {
			// We cannot accept any uploads if Ingress quota or max concurrent ingress limit exceeds
			gctx.AbortWithStatus(http.StatusBadRequest)
			return
		}

		gangKey := "gang:" + user.Username
		gang, dberr := gangRepo.GetGang(gctx, logger, gangKey, user.Username, false)
		if dberr != nil {
			// Error occured, might be validation or server error
			err, ok := dberr.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		} else if gang.Admin == "" {
			gctx.AbortWithStatus(http.StatusUnauthorized)
			return
		} else if gang.Streaming || gang.ContentURL != "" || gang.ContentScreenShare {
			// Cannot upload files while other streaming options are filled up
			gctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		// Check if enough disk space is available to accept another content
		// Convert MAX_UPLOAD_SIZE to int64
		contentUploadSize, err := strconv.ParseInt(MAX_UPLOAD_SIZE, 10, 64)
		if err != nil {
			// Set default to 524MBs
			contentUploadSize = 524288000
		}
		diskSpaceAvail, err := getAvailableDiskSpace(gctx, logger)
		if err != nil {
			gctx.AbortWithStatus(http.StatusInternalServerError)
			return
		} else if diskSpaceAvail-52428800 < uint64(contentUploadSize) {
			// Not enough space available
			gctx.AbortWithStatus(http.StatusInsufficientStorage)
			return
		}
		if gctx.Request.Method == "DELETE" {
			// Erase content ID and filename from DB
			defer func() {
				gangRepo.UpdateGangContentData(gctx, logger, user.Username, "", "", "", false, false)
			}()
		}
		gctx.Request.Header.Add("User", user.Username) // to be used in tusd callbacks
		gctx.Next()
	}
}

// Helper method to get available disk space
func getAvailableDiskSpace(ctx context.Context, logger log.Logger) (uint64, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(UPLOAD_PATH, &fs)
	if err != nil {
		// Error occured in Statfs()
		logger.WithCtx(ctx).Error().Err(err).Msg("Error occured while trying to fetch available disk space")
		return 0, err
	}
	return fs.Bfree * uint64(fs.Bsize), err
}
