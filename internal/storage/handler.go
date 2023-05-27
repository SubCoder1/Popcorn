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
	ffmpeg "github.com/u2takey/ffmpeg-go"
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
			// Start encoding the uploaded content
			go encodeContentIntoH264(logger, event.Upload.ID)
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

// Encodes successfully uploaded content into proper format
func encodeContentIntoH264(logger log.Logger, content_ID string) {
	logger.Info().Msgf("Starting encoding of uploaded content - %s using ffmpeg", content_ID)
	input_path := content_dir + "/" + content_ID
	vid_output_path := content_dir + "/" + content_ID + ".h264"
	aud_output_path := content_dir + "/" + content_ID + ".ogg"

	err := ffmpeg.Input(input_path).Output(vid_output_path, ffmpeg.KwArgs{
		"c:v": "libx264", "bsf:v": "h264_mp4toannexb", "b:v": "2M",
		"pix_fmt": "yuv420p", "x264-params": "keyint=120",
		"max_delay": 0, "loglevel": "warning", "movflags": "+faststart",
		"c:a": "libopus", "b:a": "128k", "page_duration": 20000, "vn": aud_output_path,
	}).OverWriteOutput().ErrorToStdOut().Run()

	if err != nil {
		// Error occured during encoding content
		logger.Error().Err(err).Msgf("Error occured while encoding content - %s", content_ID)
		// Print the info file of the content for better debugging
		content_info, oserr := os.ReadFile(input_path + ".info")
		if oserr != nil {
			// Could not read .info file (maybe missing or corrupted?)
			logger.Error().Err(oserr).Msgf("Could not read file - %s", input_path)
		}
		logger.Info().Msg(string(content_info))
	}
}
