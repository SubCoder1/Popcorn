// External pkg tusd to handle large file upload with resumable feature and file chunking.

package storage

import (
	"Popcorn/internal/gang"
	"Popcorn/pkg/log"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/h2non/filetype"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
)

var (
	store         filestore.FileStore
	composer      *tusd.StoreComposer
	handler       *tusd.UnroutedHandler
	tusderr       error
	once          sync.Once
	content_types map[string]string = map[string]string{"video/mp4": "mp4", "video/x-msvideo": "avi", "video/x-matroska": "mkv"}
)

// Returns a fresh or existing Tusd Unrouted handler to help in gang content upload
func GetTusdStorageHandler(gangRepo gang.Repository, logger log.Logger) *tusd.UnroutedHandler {
	// Check if upload directory exists, if not make one
	path := "./uploads"
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			logger.Fatal().Err(err).Msg("Error during creating upload directory for tusd storage")
		}
	}

	once.Do(func() {
		store = filestore.FileStore{Path: path}

		composer = tusd.NewStoreComposer()
		store.UseIn(composer)

		handler, tusderr = tusd.NewUnroutedHandler(tusd.Config{
			BasePath:              "/api/upload_content",
			MaxSize:               524288000,
			StoreComposer:         composer,
			NotifyCompleteUploads: true,
			DisableDownload:       true,
			PreUploadCreateCallback: func(hook tusd.HookEvent) error {
				// Validate metadata attached with the upload request
				user := hook.HTTPRequest.Header.Get("User")
				gangKey := "gang:" + user
				available, dberr := gangRepo.HasGang(context.Background(), logger, gangKey, "")
				if dberr != nil || !available {
					return tusd.ErrUploadStoppedByServer
				}
				filetype := hook.Upload.MetaData["filetype"]
				if _, ok := content_types[filetype]; !ok {
					// invalid filetype
					return tusd.ErrInvalidContentType
				}
				if len(hook.Upload.MetaData["filename"]) == 0 {
					// filename cannot be blank
					return tusd.ErrNotFound
				}
				return nil
			},
			PreFinishResponseCallback: func(hook tusd.HookEvent) error {
				// Validate uploaded file and add filename and ID into gang data upon success
				filepath := path + "/" + hook.Upload.ID
				file, oserr := os.Open(filepath)
				if oserr != nil {
					logger.Error().Err(oserr).Msg("Cannot open content - " + hook.Upload.ID)
					return tusd.ErrFileLocked
				}
				head := make([]byte, 261)
				file.Read(head)
				if !filetype.IsVideo(head) {
					// Filetype validation failed
					return tusd.ErrInvalidContentType
				}

				user := hook.HTTPRequest.Header.Get("User")
				fmt.Println(hook.Upload.MetaData["filename"], hook.Upload.ID)
				dberr := gangRepo.UpdateGangContentData(context.Background(), logger, user, hook.Upload.MetaData["filename"], hook.Upload.ID)
				if dberr != nil {
					// Error occured in EraseGangContentData()
					return tusd.NewHTTPError(errors.New("internal server error"), 500)
				}
				return nil
			},
		})
		if tusderr != nil {
			logger.Fatal().Err(tusderr).Msg("Unable to create tusd handler")
		}
		// Start another goroutine for receiving events from the handler whenever
		// an upload is completed. The event will contains details about the upload
		// itself and the relevant HTTP request.
		go func() {
			for {
				event := <-handler.CompleteUploads
				logger.Info().Msgf("Upload %s finished\n", event.Upload.ID)
			}
		}()
	})

	return handler
}
