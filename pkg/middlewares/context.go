package middlewares

import (
	"Popcorn/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
)

// This middleware will be used to populate every incoming request's context with an Unique CorrelationID.
// Which will help to debug an issue which happened between a chain of events during handling a request.
func CorrelationMiddleware(logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		correlationID := xid.New().String()
		// Setting the correlationID in request's context
		gctx.Set("correlation_id", correlationID)
		// Setting the correlationID to response header
		gctx.Writer.Header().Set("X-Correlation-ID", correlationID)
	}
}
