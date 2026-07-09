package cli

import (
	"errors"
	"os"
	"path/filepath"
)

func findFirstFileByName(
	rootDir, fileName string,
	onWalkError func(path string, err error),
	onWalkFailed func(rootDir string, err error),
) string {
	var found string
	walkErr := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if onWalkError != nil {
				onWalkError(path, err)
			}
			return nil
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if info.Name() == fileName && found == "" {
			found = path
			return errWalkStop
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errWalkStop) && onWalkFailed != nil {
		onWalkFailed(rootDir, walkErr)
	}
	return found
}
