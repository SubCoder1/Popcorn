package cleanup

import (
	"Popcorn/pkg/log"
	"os"
)

// Helper method to delete file due to any issues found during or post upload
func DeleteContentFiles(filepath string, logger log.Logger) {
	if len(filepath) != 0 {
		ext := []string{"", ".info"}
		for i := 0; i < len(ext); i++ {
			oserr := os.Remove(filepath + ext[i])
			if oserr != nil {
				logger.Error().Err(oserr).Msgf("Error occured during deleting content file - %s", filepath+ext[i])
			}
		}
	}
}
