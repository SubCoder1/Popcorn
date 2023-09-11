package cleanup

import (
	"Popcorn/pkg/log"
	"os"
)

var UPLOAD_PATH string = os.Getenv("UPLOAD_PATH")

// Helper method to delete file due to any issues found during or post upload
func DeleteContentFiles(contentID string, logger log.Logger) {
	if len(contentID) != 0 {
		ext := []string{"", ".info"}
		for i := 0; i < len(ext); i++ {
			oserr := os.Remove(UPLOAD_PATH + contentID + ext[i])
			if oserr != nil {
				logger.Error().Err(oserr).Msgf("Error occured during deleting content file - %s", UPLOAD_PATH+contentID+ext[i])
			}
		}
	}
}
