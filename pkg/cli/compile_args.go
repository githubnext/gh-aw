// This file provides argument preprocessing for the compile command.
//
// It handles expansion of local directory paths into their constituent
// workflow .md files so that the rest of the compilation pipeline only
// needs to deal with concrete file paths.
//
// # Key Functions
//
//   - resolveCompileArgs() - Expand a list of compile arguments
//   - expandCompileArg()   - Expand a single argument (directory or file)
//   - expandDirectoryArg() - Return all .md workflow files inside a directory

package cli

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var compileArgsLog = logger.New("cli:compile_args")

// resolveCompileArgs preprocesses compile command arguments to handle
// local directory paths. When an argument is a directory it is expanded
// to all .md workflow files in that directory.
func resolveCompileArgs(args []string, verbose bool) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}

	var result []string
	for _, arg := range args {
		expanded, err := expandCompileArg(arg, verbose)
		if err != nil {
			return nil, err
		}
		result = append(result, expanded...)
	}
	return result, nil
}

// expandCompileArg expands a single compile argument:
//   - Local directory paths: expand to all .md files in that directory
//   - Everything else: return as-is for the existing resolver to handle
func expandCompileArg(arg string, verbose bool) ([]string, error) {
	compileArgsLog.Printf("Processing compile argument: %s", arg)

	// Handle local directory paths
	info, err := os.Stat(arg)
	if err == nil && info.IsDir() {
		return expandDirectoryArg(arg, verbose)
	}

	// Return as-is (regular file path or workflow name)
	return []string{arg}, nil
}

// expandDirectoryArg expands a directory path to all .md workflow files in it.
func expandDirectoryArg(dirPath string, verbose bool) ([]string, error) {
	compileArgsLog.Printf("Expanding directory argument: %s", dirPath)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Compiling all workflows in directory: "+dirPath))
	}

	mdFiles, err := getMarkdownWorkflowFiles(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find workflow files in %s: %w", dirPath, err)
	}

	mdFiles, err = filterMarkdownFilesWithFrontmatter(mdFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to filter workflow files in %s: %w", dirPath, err)
	}

	if len(mdFiles) == 0 {
		return nil, fmt.Errorf("no workflow markdown files found in %s (workflow files must start with a frontmatter opener on the first line)", dirPath)
	}

	compileArgsLog.Printf("Found %d workflow files in directory %s", len(mdFiles), dirPath)
	return mdFiles, nil
}
