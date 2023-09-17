// External pkg tusd to handle large file upload with resumable feature and file chunking.

package storage

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/gang"
	"Popcorn/internal/sse"
	"Popcorn/pkg/cleanup"
	"Popcorn/pkg/log"
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/h2non/filetype"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
)

var (
	store           filestore.FileStore
	composer        *tusd.StoreComposer
	handler         *tusd.UnroutedHandler
	tusderr         error
	content_types   map[string]string = map[string]string{"video/mp4": "mp4", "video/x-matroska": "mkv"}
	ctx             context.Context   = context.Background()
	UPLOAD_PATH     string            = os.Getenv("UPLOAD_PATH")
	MAX_UPLOAD_SIZE string            = os.Getenv("MAX_UPLOAD_SIZE")
)

// Returns a fresh or existing Tusd Unrouted handler to help in gang content upload
func GetTusdStorageHandler(gangRepo gang.Repository, sseService sse.Service, logger log.Logger) *tusd.UnroutedHandler {
	// Check if upload directory exists, if not make one
	if _, err := os.Stat(UPLOAD_PATH); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(UPLOAD_PATH, 0777)
		if err != nil {
			logger.WithCtx(ctx).Fatal().Err(err).Msg("Error during creating upload directory for tusd storage")
		}
	}
	// Convert MAX_UPLOAD_SIZE to int64
	contentUploadSize, err := strconv.ParseInt(MAX_UPLOAD_SIZE, 10, 64)
	if err != nil {
		// Set default to 524MBs
		contentUploadSize = 524288000
	}

	store = filestore.FileStore{Path: UPLOAD_PATH}

	composer = tusd.NewStoreComposer()
	store.UseIn(composer)

	handler, tusderr = tusd.NewUnroutedHandler(tusd.Config{
		BasePath:                "/api/upload_content",
		MaxSize:                 contentUploadSize,
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		NotifyTerminatedUploads: true,
		DisableDownload:         false,
		RespectForwardedHeaders: true,
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
			user := hook.HTTPRequest.Header.Get("User")
			gangKey := "gang:" + user
			// Check if content URL is there already for this gang
			gang, dberr := gangRepo.GetGang(ctx, logger, gangKey, user, false)
			if dberr != nil {
				// Error occured in GetGang()
				return tusd.NewHTTPError(dberr, 500)
			} else if gang.ContentURL != "" {
				// cannot contain content file & URL at the same time
				return tusd.NewHTTPError(errors.New("cannot contain content file & URL at the same time"), 400)
			}
			// Validate uploaded file and add filename and ID into gang data upon success
			filepath := UPLOAD_PATH + hook.Upload.ID
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

			dberr = gangRepo.UpdateGangContentData(ctx, logger, user, hook.Upload.MetaData["filename"], hook.Upload.ID, "", false)
			if dberr != nil {
				// Error occured in UpdateGangContentData()
				return tusd.NewHTTPError(dberr, 500)
			}

			// The uploaded file should be deleted if not streamed under 10mins as storage is limited
			time.AfterFunc(10*time.Minute, func() {
				gang, _ := gangRepo.GetGang(ctx, logger, gangKey, user, false)

				if len(gang.Name) != 0 && !gang.Streaming {
					logger.Info().Msgf("Deleting unstreamed content files for: %s", gangKey)
					// Delete gang content files
					cleanup.DeleteContentFiles(gang.ContentID, logger)
					// Erase gang content data from DB
					gangRepo.UpdateGangContentData(ctx, logger, user, "", "", "", false)
					// Notify the members that stream has stopped
					members, _ := gangRepo.GetGangMembers(ctx, logger, user)
					for _, member := range members {
						go func(member string) {
							data := entity.SSEData{
								Data: nil,
								Type: "gangUpdate",
								To:   member,
							}
							sseService.GetOrSetEvent(ctx).Message <- data
						}(member)
					}
				}
			})

			diskSpaceAvail, _ := getAvailableDiskSpace(ctx, logger)
			logger.WithCtx(ctx).Info().Msgf("Available disk space - %d", diskSpaceAvail)

			return nil
		},
	})
	if tusderr != nil {
		logger.WithCtx(ctx).Fatal().Err(tusderr).Msg("Unable to create tusd handler")
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
