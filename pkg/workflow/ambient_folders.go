package workflow

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var ambientFoldersLog = logger.New("workflow:ambient_folders")

var ambientFolderPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

func resolveAmbientFolders(frontmatter map[string]any, importsResult *parser.ImportsResult) ([]string, error) {
	var merged []string
	if importsResult != nil {
		merged = append(merged, importsResult.MergedAmbientFolders...)
		ambientFoldersLog.Printf("Merged %d ambient folders from imports", len(importsResult.MergedAmbientFolders))
	}
	main, err := extractAmbientFolders(frontmatter)
	if err != nil {
		return nil, err
	}
	merged = append(merged, main...)
	normalized, err := normalizeAmbientFolders(merged)
	if err != nil {
		return nil, err
	}
	ambientFoldersLog.Printf("Resolved %d ambient folders", len(normalized))
	return normalized, nil
}

func extractAmbientFolders(frontmatter map[string]any) ([]string, error) {
	if frontmatter == nil {
		return nil, nil
	}
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
			return nil, errors.New("ambient-folders has an unsupported type, expected an array of folder path strings. Example: ambient-folders: [docs, src/lib]")
		}
	}
	folders := make([]string, 0, len(values))
	for _, value := range values {
		folder, ok := value.(string)
		if !ok {
			return nil, errors.New("ambient-folders entry has an unsupported type, expected a string folder path. Example: ambient-folders: [docs, src/lib]")
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
			return nil, errors.New("ambient-folders entry is empty, expected a non-empty relative folder path. Example: ambient-folders: [docs, src/lib]")
		}
		clean := filepath.ToSlash(filepath.Clean(value))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
			return nil, fmt.Errorf("ambient-folders entry %q is not a relative path within the repository, expected a relative folder path without '..' or a leading '/'. Example: ambient-folders: [docs, src/lib]", folder)
		}
		if !ambientFolderPattern.MatchString(clean) {
			return nil, fmt.Errorf("ambient-folders entry %q contains unsupported characters, expected only letters, digits, '.', '_', '-', and '/'. Example: ambient-folders: [docs, src/lib]", folder)
		}
		if _, exists := seen[clean]; exists {
			ambientFoldersLog.Printf("Skipping duplicate ambient folder: %s", clean)
			continue
		}
		seen[clean] = struct{}{}
		normalized = append(normalized, clean)
	}
	return normalized, nil
}

func generateStageAmbientFoldersStep(data *WorkflowData) []string {
	if data == nil || len(data.AmbientFolders) == 0 {
		return nil
	}
	folders := strings.Join(data.AmbientFolders, " ")
	return []string{
		"      - name: " + constants.ActivationStageAmbientFoldersStepName + "\n",
		"        env:\n",
		fmt.Sprintf("          GH_AW_AMBIENT_FOLDERS: \"%s\"\n", folders),
		"        run: |\n",
		"          mkdir -p /tmp/gh-aw/ambient-folders\n",
		"          for folder in $GH_AW_AMBIENT_FOLDERS; do\n",
		"            src=\"$GITHUB_WORKSPACE/$folder\"\n",
		"            dst=\"/tmp/gh-aw/ambient-folders/$folder\"\n",
		"            if [ -e \"$src\" ]; then\n",
		"              mkdir -p \"$(dirname \"$dst\")\"\n",
		"              rm -rf \"$dst\"\n",
		"              cp -a \"$src\" \"$dst\"\n",
		"            fi\n",
		"          done\n",
	}
}

func generateRestoreAmbientFoldersStep(yaml *strings.Builder, data *WorkflowData) {
	for _, line := range restoreAmbientFoldersSteps(data) {
		yaml.WriteString(line)
		yaml.WriteByte('\n')
	}
}

func restoreAmbientFoldersSteps(data *WorkflowData) GitHubActionStep {
	if data == nil || len(data.AmbientFolders) == 0 {
		return nil
	}
	return GitHubActionStep{
		"      - name: Restore ambient folders from activation artifact",
		"        env:",
		fmt.Sprintf("          GH_AW_AMBIENT_FOLDERS: \"%s\"", strings.Join(data.AmbientFolders, " ")),
		"        run: |",
		"          for folder in $GH_AW_AMBIENT_FOLDERS; do",
		"            src=\"/tmp/gh-aw/ambient-folders/$folder\"",
		"            dst=\"$GITHUB_WORKSPACE/$folder\"",
		"            if [ -e \"$src\" ]; then",
		"              mkdir -p \"$(dirname \"$dst\")\"",
		"              rm -rf \"$dst\"",
		"              cp -a \"$src\" \"$dst\"",
		"            fi",
		"          done",
	}
}
