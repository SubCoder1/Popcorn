// Exposes all of the REST APIs related to Content upload in Popcorn.

package storage

import (
	"Popcorn/internal/gang"
	"Popcorn/pkg/log"

	"github.com/gin-gonic/gin"
)

func APIHandlers(router *gin.Engine, gangRepo gang.Repository, authWithAcc gin.HandlerFunc, logger log.Logger) {
	storage_handler := GetTusdStorageHandler(gangRepo, logger)
	router.POST("/api/upload_content", authWithAcc, gin.WrapF(storage_handler.PostFile))
	router.GET("/api/upload_content/:id", authWithAcc, gin.WrapF(storage_handler.GetFile))
	router.HEAD("/api/upload_content/:id", authWithAcc, gin.WrapF(storage_handler.HeadFile))
	router.PATCH("/api/upload_content/:id", authWithAcc, gin.WrapF(storage_handler.PatchFile))
}
