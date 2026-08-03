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
	onMap := ensureOnMap(frontmatter)
	if onMap == nil {
		return nil, nil
	}
	raw, exists := onMap["ambient-folders"]
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
			return nil, errors.New("on.ambient-folders must be an array of folder paths")
		}
	}
	folders := make([]string, 0, len(values))
	for _, value := range values {
		folder, ok := value.(string)
		if !ok {
			return nil, errors.New("on.ambient-folders entries must be strings")
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
			return nil, errors.New("on.ambient-folders entries cannot be empty")
		}
		clean := filepath.ToSlash(filepath.Clean(value))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
			return nil, fmt.Errorf("on.ambient-folders entry %q must be a relative folder path within the repository", folder)
		}
		if !ambientFolderPattern.MatchString(clean) {
			return nil, fmt.Errorf("on.ambient-folders entry %q contains unsupported characters", folder)
		}
		if _, exists := seen[clean]; exists {
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
		"      - name: Stage ambient folders for activation artifact\n",
		"        env:\n",
		fmt.Sprintf("          GH_AW_AMBIENT_FOLDERS: \"%s\"\n", folders),
		"        # poutine:ignore untrusted_checkout_exec\n",
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
		"        # poutine:ignore untrusted_checkout_exec",
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
