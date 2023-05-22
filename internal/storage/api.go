// Exposes all of the REST APIs related to Content upload in Popcorn.

package storage

import (
	"Popcorn/pkg/log"

	"github.com/gin-gonic/gin"
	tusd "github.com/tus/tusd/pkg/handler"
)

func APIHandlers(router *gin.Engine, storage_handler *tusd.UnroutedHandler, authWithAcc, validateAdmin gin.HandlerFunc, logger log.Logger) {
	router.POST("/api/upload_content", authWithAcc, validateAdmin, gin.WrapF(storage_handler.PostFile))
	router.GET("/api/upload_content/:id", authWithAcc, validateAdmin, gin.WrapF(storage_handler.GetFile))
	router.HEAD("/api/upload_content/:id", authWithAcc, validateAdmin, gin.WrapF(storage_handler.HeadFile))
	router.PATCH("/api/upload_content/:id", authWithAcc, validateAdmin, gin.WrapF(storage_handler.PatchFile))
	router.DELETE("/api/upload_content/:id", authWithAcc, validateAdmin, gin.WrapF(storage_handler.DelFile))
}
