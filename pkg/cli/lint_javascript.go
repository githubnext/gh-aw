package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/taint"
	"github.com/spf13/cobra"
)

func registerJavaScriptLintFlags(cmd *cobra.Command) {
	cmd.Flags().String("javascript-module", "", "Explicit directory containing the pinned @typescript/native-preview npm dependency (optional native JavaScript AST analysis)")
	cmd.Flags().String("javascript-mode", "fast", "JavaScript analysis mode: fast (authored scripts and reviewed API summaries) or deep (traverse runtime sources; requires --javascript-module)")
	cmd.Flags().String("javascript-root", "", "Repository source root for JavaScript entry points and relative imports (requires --javascript-module)")
	cmd.Flags().StringArray("javascript-runtime-path", nil, "Verified runner directory=repository directory mapping; repeat for additional asset roots (requires --javascript-root)")
}

func javascriptLintOptions(cmd *cobra.Command) (*taint.JavaScriptOptions, func() error, error) {
	module, _ := cmd.Flags().GetString("javascript-module")
	mode, _ := cmd.Flags().GetString("javascript-mode")
	root, _ := cmd.Flags().GetString("javascript-root")
	mappings, _ := cmd.Flags().GetStringArray("javascript-runtime-path")
	closeFiles := func() error { return nil }
	if mode != string(taint.JavaScriptFast) && mode != string(taint.JavaScriptDeep) {
		return nil, closeFiles, errors.New("--javascript-mode must be fast or deep")
	}
	if module == "" {
		if cmd.Flags().Changed("javascript-mode") {
			return nil, closeFiles, errors.New("--javascript-mode requires --javascript-module")
		}
		if root != "" || len(mappings) > 0 {
			return nil, closeFiles, errors.New("--javascript-root and --javascript-runtime-path require --javascript-module")
		}
		return nil, closeFiles, nil
	}
	if root == "" && len(mappings) > 0 {
		return nil, closeFiles, errors.New("--javascript-runtime-path requires --javascript-root")
	}
	module, err := filepath.Abs(module)
	if err != nil {
		return nil, closeFiles, errors.New("could not resolve --javascript-module")
	}
	options := &taint.JavaScriptOptions{
		Context: cmd.Context(), Parser: &taint.TypeScriptParser{ModuleDirectory: module},
		RuntimePaths: make(map[string]string),
		Mode:         taint.JavaScriptMode(mode),
	}
	for _, mapping := range mappings {
		from, to, ok := strings.Cut(mapping, "=")
		from = strings.TrimSuffix(from, "/")
		if !ok || from == "" || !fs.ValidPath(to) {
			return nil, closeFiles, errors.New("--javascript-runtime-path requires runner-directory=repository-relative-directory")
		}
		if _, exists := options.RuntimePaths[from]; exists {
			return nil, closeFiles, errors.New("--javascript-runtime-path contains a duplicate runner directory")
		}
		options.RuntimePaths[from] = to
	}
	if root != "" {
		opened, err := os.OpenRoot(root)
		if err != nil {
			return nil, closeFiles, errors.New("could not open --javascript-root")
		}
		options.Files = opened.FS()
		closeFiles = opened.Close
	}
	return options, closeFiles, nil
}
