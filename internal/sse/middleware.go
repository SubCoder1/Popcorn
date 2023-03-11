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

func SSEConnMiddleware(stream *entity.SSE, sseRepo Repository, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the joingang service
		username, ok := gctx.Value("Username").(string)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in SSEConnMiddleware")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		// Initialize client
		client := entity.SSEClient{
			ID:      username,
			Channel: make(chan entity.SSEData),
		}
		// Add the username into DB SSE Bucket
		dberr := sseRepo.AddClient(gctx, logger, username)
		if dberr != nil {
			// Issue in AddClient
			gctx.AbortWithStatus(http.StatusInternalServerError)
		}

		// Send new connection to event to store
		stream.NewClients <- client

		defer func() {
			// Send closed connection to event server
			logger.WithCtx(gctx).Info().Msg(fmt.Sprintf("Closing SSE connection : %s", client.ID))
			stream.ClosedClients <- client
		}()

		gctx.Set("SSE", client)
		gctx.Next()
	}
}
