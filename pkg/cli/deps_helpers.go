package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var depsHelpersLog = logger.New("cli:deps_helpers")

// findGoMod locates the go.mod file in the repository.
// It first checks the current directory, then falls back to the git root.
func findGoMod() (string, error) {
	// Try current directory first
	if _, err := os.Stat("go.mod"); err == nil {
		absPath, err := filepath.Abs("go.mod")
		if err == nil {
			depsHelpersLog.Printf("Found go.mod in current directory: %s", absPath)
		}
		return absPath, err
	}

	// Try git root
	root, err := findGitRoot()
	if err != nil {
		return "", errors.New("not in a Go module (no go.mod found)")
	}

	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		return "", errors.New("not in a Go module (no go.mod found)")
	}

	depsHelpersLog.Printf("Found go.mod at git root: %s", goModPath)
	return goModPath, nil
}

// parseGoModFile parses a go.mod file and returns all dependencies (direct and indirect).
// Callers can filter on the Indirect field as needed.
func parseGoModFile(path string) ([]DependencyInfoWithIndirect, error) {
	depsHelpersLog.Printf("Parsing go.mod file: %s", path)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var deps []DependencyInfoWithIndirect
	lines := strings.Split(string(content), "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Track require block
		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		// Parse dependency line
		if inRequire || strings.HasPrefix(line, "require ") {
			// Remove "require " prefix if present
			line = strings.TrimPrefix(line, "require ")

			// Check if indirect before splitting (preserve the comment)
			indirect := strings.Contains(line, "// indirect")

			parts := strings.Fields(line)
			if len(parts) >= 2 {
				deps = append(deps, DependencyInfoWithIndirect{
					DependencyInfo: DependencyInfo{
						Path:    parts[0],
						Version: parts[1],
					},
					Indirect: indirect,
				})
			}
		}
	}

	depsHelpersLog.Printf("Parsed %d dependencies from go.mod", len(deps))
	return deps, nil
}
