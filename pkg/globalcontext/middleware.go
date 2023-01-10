// Context middleware is used in gin to populate request context with unique ID.
// This ID will be helpful in debugging issues happening for a request in handler chain.

package globalcontext

import (
	"Popcorn/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// This middleware will be used to populate every incoming request's context with an Unique UUID.
// This middleware will be used as a global one.
func UniqueIDMiddleware(logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		rqId, uuiderr := uuid.NewRandom()
		if uuiderr != nil {
			logger.Error().Err(uuiderr).Msg("Error during generating UUID for ReqID.")
		} else {
			gctx.Set("ReqID", rqId.String())
		}
	}
}
