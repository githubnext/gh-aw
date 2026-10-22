package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/aw"
)

func expandLintDirectory(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("reading lint directory: %w", err)
	}
	names := make(map[string]struct{})
	for _, entry := range entries {
		if !entry.IsDir() {
			names[entry.Name()] = struct{}{}
		}
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		file := filepath.Join(directory, entry.Name())
		switch filepath.Ext(file) {
		case ".yml", ".yaml":
			files = append(files, file)
		case ".md":
			if _, paired := names[strings.TrimSuffix(entry.Name(), ".md")+".lock.yml"]; paired {
				continue
			}
			content, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("reading AW lint candidate: %w", err)
			}
			document, err := aw.Parse(string(content))
			if err != nil || document.Location.FrontmatterStart > 0 || len(document.Frontmatter) > 0 {
				files = append(files, file)
			}
		}
	}
	return files, nil
}
