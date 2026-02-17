package workflow

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/stringutil"
)

const (
	// Unversioned lock files are treated as legacy schema v0 for backward compatibility.
	legacyLockSchemaVersion  = 0
	currentLockSchemaVersion = 1

	lockSchemaVersionCommentPrefix = "# gh-aw-lock-schema-version:"
)

type lockSchemaCompatibility struct {
	MinReadable int
	MaxReadable int
}

var defaultLockSchemaCompatibility = lockSchemaCompatibility{
	MinReadable: legacyLockSchemaVersion,
	MaxReadable: currentLockSchemaVersion,
}

func prependLockSchemaVersionHeader(yamlContent string) string {
	if strings.HasPrefix(strings.TrimSpace(yamlContent), lockSchemaVersionCommentPrefix) {
		return yamlContent
	}

	return fmt.Sprintf("%s %d\n%s", lockSchemaVersionCommentPrefix, currentLockSchemaVersion, yamlContent)
}

func parseLockSchemaVersion(content []byte) (int, error) {
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if !strings.HasPrefix(trimmed, "#") {
			break
		}

		if strings.HasPrefix(trimmed, lockSchemaVersionCommentPrefix) {
			rawVersion := strings.TrimSpace(strings.TrimPrefix(trimmed, lockSchemaVersionCommentPrefix))
			if rawVersion == "" {
				return 0, fmt.Errorf("missing lock schema version value")
			}

			version, err := strconv.Atoi(rawVersion)
			if err != nil {
				return 0, fmt.Errorf("invalid lock schema version %q", rawVersion)
			}
			if version < 0 {
				return 0, fmt.Errorf("invalid lock schema version %d", version)
			}

			return version, nil
		}
	}

	return legacyLockSchemaVersion, nil
}

func ValidateLockSchemaCompatibility(lockFilePath string, content []byte) error {
	lockSchemaVersion, err := parseLockSchemaVersion(content)
	if err != nil {
		return fmt.Errorf(
			"failed to read lock schema version for '%s': %w. Regenerate with 'gh aw compile %s'",
			lockFilePath,
			err,
			stringutil.NormalizeWorkflowName(filepath.Base(lockFilePath)),
		)
	}

	if lockSchemaVersion < defaultLockSchemaCompatibility.MinReadable ||
		lockSchemaVersion > defaultLockSchemaCompatibility.MaxReadable {
		return fmt.Errorf(
			"incompatible lock schema version %d in '%s' (supported read range: %d-%d). "+
				"Regenerate with 'gh aw compile %s' or upgrade gh-aw",
			lockSchemaVersion,
			lockFilePath,
			defaultLockSchemaCompatibility.MinReadable,
			defaultLockSchemaCompatibility.MaxReadable,
			stringutil.NormalizeWorkflowName(filepath.Base(lockFilePath)),
		)
	}

	return nil
}
