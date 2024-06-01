// Exposes all of the REST APIs related to SSE in Popcorn.

package sse

import (
	"Popcorn/internal/entity"
	"Popcorn/pkg/log"
	"Popcorn/pkg/middlewares"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Registers all of the REST API handlers related to internal package sse onto the gin server.
func APIHandlers(router *gin.Engine, service Service, authWithAcc, sseConnManager gin.HandlerFunc, logger log.Logger) {
	router.GET("/api/sse", authWithAcc, middlewares.SSECORSMiddleware(), sseConnManager, ssehandler())
}

func ssehandler() gin.HandlerFunc {
	return func(gctx *gin.Context) {
		v, ok := gctx.Get("SSE")
		if !ok {
			gctx.Status(http.StatusInternalServerError)
			return
		}
		client, ok := v.(entity.SSEClient)
		if !ok {
			gctx.Status(http.StatusInternalServerError)
			return
		}

		gctx.Stream(func(w io.Writer) bool {
			ticker := time.NewTicker(20 * time.Second)
			// Stream data to client
			for {
				select {
				// Send msg to the client
				case msg, ok := <-client.Channel:
					if !ok {
						ticker.Stop()
						return false
					}
					gctx.SSEvent(msg.Type, msg)
					return true
				// Send a ping to the client to ensure the connection isn't dropped by nginx
				case <-ticker.C:
					gctx.SSEvent("PING", "PING")
					return true
				// Client exit
				case <-gctx.Request.Context().Done():
					ticker.Stop()
					return false
				// Server force-close
				case <-quit:
					ticker.Stop()
					return false
				}
			}
		})
	}
}
