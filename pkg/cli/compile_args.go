// This file provides argument preprocessing for the compile command.
//
// It handles expansion of directory paths and GitHub URLs into their
// constituent workflow .md files so that the rest of the compilation
// pipeline only needs to deal with concrete file paths.
//
// # Key Functions
//
//   - resolveCompileArgs() - Expand a list of compile arguments
//   - expandCompileArg()   - Expand a single argument (URL, directory, or file)
//   - expandURLArg()       - Parse a GitHub URL and resolve its local path
//   - expandDirectoryArg() - Return all .md workflow files inside a directory

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var compileArgsLog = logger.New("cli:compile_args")

// resolveCompileArgs preprocesses compile command arguments to handle
// directory paths and GitHub URLs. When an argument is a directory or a
// GitHub URL pointing to a folder, it is expanded to all .md workflow files
// in that directory.
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
//   - GitHub URLs (http:// or https://) pointing to a directory: expand to all .md files
//   - GitHub URLs pointing to a specific .md file: return the extracted local path
//   - Local directory paths: expand to all .md files in that directory
//   - Everything else: return as-is for the existing resolver to handle
func expandCompileArg(arg string, verbose bool) ([]string, error) {
	compileArgsLog.Printf("Processing compile argument: %s", arg)

	// Handle GitHub URLs
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		return expandURLArg(arg, verbose)
	}

	// Handle local directory paths
	info, err := os.Stat(arg)
	if err == nil && info.IsDir() {
		return expandDirectoryArg(arg, verbose)
	}

	// Return as-is (regular file path or workflow name)
	return []string{arg}, nil
}

// expandURLArg handles a GitHub URL argument for the compile command.
// For tree (directory) URLs, it compiles all workflows in the corresponding
// local directory. For blob/raw (file) URLs, it returns the extracted local
// file path so that the standard resolver can locate the file.
func expandURLArg(urlArg string, verbose bool) ([]string, error) {
	compileArgsLog.Printf("Parsing GitHub URL argument: %s", urlArg)

	components, err := parser.ParseGitHubURL(urlArg)
	if err != nil {
		compileArgsLog.Printf("Failed to parse URL %s: %v - using as-is", urlArg, err)
		// Return the URL as-is; the standard resolver will produce a clear error
		return []string{urlArg}, nil
	}

	localPath := components.Path
	if localPath == "" {
		compileArgsLog.Printf("No path extracted from URL %s", urlArg)
		return []string{urlArg}, nil
	}

	compileArgsLog.Printf("Extracted local path from URL: %s (type=%s)", localPath, components.Type)

	// For tree (directory) URLs, compile all .md files in that directory
	if components.Type == parser.URLTypeTree {
		return expandDirectoryArg(localPath, verbose)
	}

	// For blob/raw (file) URLs, return the extracted local path
	return []string{localPath}, nil
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
