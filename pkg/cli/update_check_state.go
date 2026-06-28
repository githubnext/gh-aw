package cli

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

func getUpdateCheckFilePathFor(fileName string, log *logger.Logger) string {
	tmpDir := os.TempDir()
	if tmpDir == "" {
		log.Print("Could not determine temp directory")
		return ""
	}

	ghAwTmpDir := filepath.Join(tmpDir, "gh-aw")
	if err := os.MkdirAll(ghAwTmpDir, constants.DirPermPublic); err != nil {
		log.Printf("Error creating gh-aw temp directory: %v", err)
		return ""
	}

	return filepath.Join(ghAwTmpDir, fileName)
}

func shouldRunUpdateCheckAtPath(lastCheckFile string, interval time.Duration, label string, log *logger.Logger) bool {
	if lastCheckFile == "" {
		log.Printf("Could not determine %s file path", label)
		return false
	}

	data, err := os.ReadFile(lastCheckFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading %s file: %v", label, err)
		}
		return true
	}

	lastCheck, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		log.Printf("Error parsing %s time: %v", label, err)
		return true
	}

	elapsed := time.Since(lastCheck)
	if elapsed < interval {
		log.Printf("Last %s was %v ago, skipping", label, elapsed)
		return false
	}

	return true
}

func writeUpdateCheckTime(path string, perm os.FileMode, label string, log *logger.Logger) {
	if path == "" {
		return
	}

	timestamp := time.Now().Format(time.RFC3339)
	if err := os.WriteFile(path, []byte(timestamp), perm); err != nil {
		log.Printf("Error writing %s time: %v", label, err)
	}
}
