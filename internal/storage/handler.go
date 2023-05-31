// External pkg tusd to handle large file upload with resumable feature and file chunking.

package storage

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/gang"
	"Popcorn/internal/sse"
	"Popcorn/pkg/log"
	"context"
	"errors"
	"os"

	"github.com/h2non/filetype"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
)

var (
	store         filestore.FileStore
	composer      *tusd.StoreComposer
	handler       *tusd.UnroutedHandler
	tusderr       error
	content_types map[string]string = map[string]string{"video/mp4": "mp4", "video/x-msvideo": "avi", "video/x-matroska": "mkv"}
	ctx           context.Context   = context.Background()
	content_dir   string            = "./uploads"
)

// Returns a fresh or existing Tusd Unrouted handler to help in gang content upload
func GetTusdStorageHandler(gangRepo gang.Repository, sseService sse.Service, logger log.Logger) *tusd.UnroutedHandler {
	// Check if upload directory exists, if not make one
	if _, err := os.Stat(content_dir); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(content_dir, 0777)
		if err != nil {
			logger.Fatal().Err(err).Msg("Error during creating upload directory for tusd storage")
		}
	}

	store = filestore.FileStore{Path: content_dir}

	composer = tusd.NewStoreComposer()
	store.UseIn(composer)

	handler, tusderr = tusd.NewUnroutedHandler(tusd.Config{
		BasePath:                "/api/upload_content",
		MaxSize:                 524288000,
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		NotifyTerminatedUploads: true,
		DisableDownload:         true,
		PreUploadCreateCallback: func(hook tusd.HookEvent) error {
			// Validate metadata attached with the upload request
			user := hook.HTTPRequest.Header.Get("User")
			gangKey := "gang:" + user
			available, dberr := gangRepo.HasGang(ctx, logger, gangKey, "")
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
			filepath := content_dir + "/" + hook.Upload.ID
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
			dberr := gangRepo.UpdateGangContentData(ctx, logger, user, hook.Upload.MetaData["filename"], hook.Upload.ID, false)
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
	// Start a goroutine for receiving events from the handler whenever
	// an upload is completed. The event will contains details about the upload
	// itself and the relevant HTTP request.
	go func() {
		for {
			event := <-handler.CompleteUploads
			logger.Info().Msgf("Upload %s finished", event.Upload.ID)
			// Send notifications to gang Members about the updates
			user := event.HTTPRequest.Header.Get("User")
			members, _ := gangRepo.GetGangMembers(ctx, logger, user)
			for _, member := range members {
				member := member
				go func() {
					data := entity.SSEData{
						Data: nil,
						Type: "gangUpdate",
						To:   member,
					}
					sseService.GetOrSetEvent(ctx).Message <- data
				}()
			}
		}
	}()
	// Start another goroutine for receiving events from the handler whenever
	// an upload is terminated.
	go func() {
		for {
			event := <-handler.TerminatedUploads
			logger.Info().Msgf("Upload %s terminated", event.Upload.ID)
			// Send notifications to gang Members about the updates
			user := event.HTTPRequest.Header.Get("User")
			members, _ := gangRepo.GetGangMembers(ctx, logger, user)
			for _, member := range members {
				member := member
				go func() {
					data := entity.SSEData{
						Data: nil,
						Type: "gangUpdate",
						To:   member,
					}
					sseService.GetOrSetEvent(ctx).Message <- data
				}()
			}
		}
	}()

	return handler
}
