package middlewares

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
