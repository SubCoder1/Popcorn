// Exposes all of the REST APIs related to SSE in Popcorn.

package sse

import (
	"Popcorn/internal/entity"
	"Popcorn/pkg/log"
	"Popcorn/pkg/middlewares"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Registers all of the REST API handlers related to internal package sse onto the gin server.
func APIHandlers(router *gin.Engine, service Service, authWithAcc gin.HandlerFunc, sseConn gin.HandlerFunc, logger log.Logger) {
	sseGroup := router.Group("/api/sse", authWithAcc, middlewares.SSEMiddleware(), sseConn)
	{
		sseGroup.GET("/stream", ssehandler(service, logger))
	}
}

func ssehandler(service Service, logger log.Logger) gin.HandlerFunc {
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
			// Stream data to client
			for {
				select {
				// Send msg to the client
				case msg, ok := <-client.Channel:
					if !ok {
						return false
					}
					gctx.SSEvent("message", msg)
					return true
				// Client exit
				case <-gctx.Request.Context().Done():
					return false
				}
			}
		})
	}
}
