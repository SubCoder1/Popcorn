// Server Side Events (SSE) middleware used to populate request context with client SSE channel.

package sse

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SSEConnMiddleware(service Service, sseRepo Repository, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the joingang service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in SSEConnMiddleware")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		// Add the username into DB SSE Bucket
		dberr := sseRepo.AddClient(gctx, logger, user.Username)
		if dberr != nil {
			// Issue in AddClient
			gctx.AbortWithStatus(http.StatusInternalServerError)
		}
		// Initialize client
		client := entity.SSEClient{
			ID:      user.Username,
			Channel: make(chan entity.SSEData),
		}

		// Send new connection to event to store
		service.GetOrSetEvent(gctx).NewClients <- client

		defer func() {
			// Send closed connection to event server
			if service.GetOrSetEvent(gctx).TotalClients[client.ID] != nil {
				logger.WithCtx(gctx).Info().Msg(fmt.Sprintf("Closing SSE connection : %s", client.ID))
				service.GetOrSetEvent(gctx).ClosedClients <- client
			}
		}()

		gctx.Set("SSE", client)
		gctx.Next()
	}
}
