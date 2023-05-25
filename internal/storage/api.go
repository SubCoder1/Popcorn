// Exposes all of the REST APIs related to Content upload in Popcorn.

package storage

import (
	"Popcorn/pkg/log"

	"github.com/gin-gonic/gin"
	tusd "github.com/tus/tusd/pkg/handler"
)

func APIHandlers(router *gin.Engine, storage_handler *tusd.UnroutedHandler, authWithAcc, validateAdmin gin.HandlerFunc, logger log.Logger) {
	uploadGroup := router.Group("/api/upload_content", authWithAcc, validateAdmin)
	{
		uploadGroup.POST("", gin.WrapF(storage_handler.PostFile))
		uploadGroup.GET("/:id", gin.WrapF(storage_handler.GetFile))
		uploadGroup.HEAD("/:id", gin.WrapF(storage_handler.HeadFile))
		uploadGroup.PATCH("/:id", gin.WrapF(storage_handler.PatchFile))
		uploadGroup.DELETE("/:id", gin.WrapF(storage_handler.DelFile))
	}
}
