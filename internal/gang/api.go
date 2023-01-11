// Exposes all of the REST APIs related to Gang creation in Popcorn.

package gang

import (
	"Popcorn/pkg/log"

	"github.com/gin-gonic/gin"
)

// Registers all of the REST API handlers related to internal package gang onto the gin server.
func GangHandlers(router *gin.Engine, service Service, AuthWithAcc gin.HandlerFunc, logger log.Logger) {

}
