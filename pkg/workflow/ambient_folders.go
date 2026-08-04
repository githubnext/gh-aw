package workflow

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/parser"
)

var ambientFolderPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

func resolveAmbientFolders(frontmatter map[string]any, importsResult *parser.ImportsResult) ([]string, error) {
	var merged []string
	if importsResult != nil {
		merged = append(merged, importsResult.MergedAmbientFolders...)
	}
	main, err := extractAmbientFolders(frontmatter)
	if err != nil {
		return nil, err
	}
	merged = append(merged, main...)
	return normalizeAmbientFolders(merged)
}

func extractAmbientFolders(frontmatter map[string]any) ([]string, error) {
	raw, exists := frontmatter["ambient-folders"]
	if !exists || raw == nil {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]string); ok {
			values = make([]any, 0, len(typed))
			for _, value := range typed {
				values = append(values, value)
			}
		} else {
			return nil, errors.New("ambient-folders must be an array of folder paths")
		}
	}
	folders := make([]string, 0, len(values))
	for _, value := range values {
		folder, ok := value.(string)
		if !ok {
			return nil, errors.New("ambient-folders entries must be strings")
		}
		folders = append(folders, folder)
	}
	return folders, nil
}

func normalizeAmbientFolders(folders []string) ([]string, error) {
	seen := make(map[string]struct{}, len(folders))
	normalized := make([]string, 0, len(folders))
	for _, folder := range folders {
		value := strings.TrimSpace(strings.ReplaceAll(folder, "\\", "/"))
		if value == "" {
			return nil, errors.New("ambient-folders entries cannot be empty")
		}
		clean := filepath.ToSlash(filepath.Clean(value))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
			return nil, fmt.Errorf("ambient-folders entry %q must be a relative folder path within the repository", folder)
		}
		if !ambientFolderPattern.MatchString(clean) {
			return nil, fmt.Errorf("ambient-folders entry %q contains unsupported characters", folder)
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		normalized = append(normalized, clean)
	}
	return normalized, nil
}
