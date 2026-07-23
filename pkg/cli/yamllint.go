package cli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var yamllintLog = logger.New("cli:yamllint")

// yamllintDefaultConfig is the inline yamllint configuration used for lock files.
// It disables rules that produce excessive noise on generated YAML output.
const yamllintDefaultConfig = `{extends: default, rules: {line-length: disable, document-start: disable, truthy: {check-keys: false}, comments: {require-starting-space: true, min-spaces-from-content: 1}}}`

// yamllintIssue represents a single issue from yamllint parsable output.
type yamllintIssue struct {
	File    string
	Line    int
	Column  int
	Level   string
	Message string
	Rule    string
}

// yamllintParsableRegex matches a single line of yamllint --format parsable output:
//
//	{file}:{line}:{col}: [{level}] {message} ({rule})
var yamllintParsableRegex = regexp.MustCompile(`^(.+):(\d+):(\d+): \[(error|warning)\] (.+) \(([^)]+)\)$`)

// runYamllintOnFiles runs yamllint on one or more .lock.yml files using Docker.
func runYamllintOnFiles(lockFiles []string, verbose bool, strict bool) error {
	if len(lockFiles) == 0 {
		return nil
	}

	yamllintLog.Printf("Running yamllint on %d file(s): %v (verbose=%t, strict=%t)", len(lockFiles), lockFiles, verbose, strict)

	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	if !filepath.IsAbs(gitRoot) {
		return fmt.Errorf("git root must be an absolute path, got: %s", gitRoot)
	}

	relPaths := make([]string, 0, len(lockFiles))
	for _, lockFile := range lockFiles {
		relPath, err := filepath.Rel(gitRoot, lockFile)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", lockFile, err)
		}
		relPaths = append(relPaths, relPath)
	}

	// #nosec G204 -- gitRoot is validated as an absolute path (from git rev-parse, a trusted source).
	// relPaths are derived from filepath.Rel(gitRoot, lockFile), preventing path traversal.
	// exec.Command passes args directly to the OS (no shell), preventing injection.
	// yamllintDefaultConfig is a compile-time constant with no user-controlled content.
	dockerArgs := append([]string{
		"run",
		"--rm",
		"-v", gitRoot + ":/workdir",
		"-w", "/workdir",
		YamllintImage,
		"-d", yamllintDefaultConfig,
		"--format", "parsable",
	}, relPaths...)

	if len(lockFiles) == 1 {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running yamllint on "+relPaths[0]))
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("Running yamllint on %d files", len(lockFiles))))
	}

	if verbose {
		dockerCmd := fmt.Sprintf("docker run --rm -v \"%s:/workdir\" -w /workdir %s -d '%s' --format parsable %s",
			gitRoot, YamllintImage, yamllintDefaultConfig, strings.Join(relPaths, " "))
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run yamllint directly: "+dockerCmd))
	}

	cmd := exec.Command("docker", dockerArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	totalIssues, parseErr := parseAndDisplayYamllintOutput(stdout.String())
	if parseErr != nil {
		yamllintLog.Printf("Failed to parse yamllint output: %v", parseErr)
		if stdout.Len() > 0 {
			fmt.Fprint(os.Stderr, stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprint(os.Stderr, stderr.String())
		}
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode := exitErr.ExitCode()
			yamllintLog.Printf("yamllint exited with code %d (issues=%d)", exitCode, totalIssues)
			// Exit code 1 indicates lint findings
			if exitCode == 1 {
				if strict {
					fileDescription := "workflows"
					if len(lockFiles) == 1 {
						fileDescription = filepath.Base(lockFiles[0])
					}
					return fmt.Errorf("strict mode: yamllint found %d issue(s) in %s - workflows must have no yamllint issues in strict mode", totalIssues, fileDescription)
				}
				return nil
			}
			fileDescription := "workflows"
			if len(lockFiles) == 1 {
				fileDescription = filepath.Base(lockFiles[0])
			}
			return fmt.Errorf("yamllint failed with exit code %d on %s", exitCode, fileDescription)
		}
		return fmt.Errorf("yamllint failed: %w", err)
	}

	return nil
}

// parseAndDisplayYamllintOutput parses yamllint --format parsable output and displays findings.
// Returns the total number of issues found.
func parseAndDisplayYamllintOutput(stdout string) (int, error) {
	if strings.TrimSpace(stdout) == "" {
		return 0, nil
	}

	totalIssues := 0
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		issue, err := parseYamllintLine(line)
		if err != nil {
			yamllintLog.Printf("Failed to parse yamllint line %q: %v", line, err)
			fmt.Fprintln(os.Stderr, line)
			continue
		}

		totalIssues++

		errorType := "warning"
		if issue.Level == "error" {
			errorType = "error"
		}

		compilerErr := console.CompilerError{
			Position: console.ErrorPosition{
				File:   issue.File,
				Line:   issue.Line,
				Column: issue.Column,
			},
			Type:    errorType,
			Message: fmt.Sprintf("[%s] %s (%s)", issue.Level, issue.Message, issue.Rule),
		}

		fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
	}

	if err := scanner.Err(); err != nil {
		return totalIssues, fmt.Errorf("failed to scan yamllint output: %w", err)
	}

	return totalIssues, nil
}

// parseYamllintLine parses a single line of yamllint --format parsable output.
// Expected format: {file}:{line}:{col}: [{level}] {message} ({rule})
func parseYamllintLine(line string) (yamllintIssue, error) {
	matches := yamllintParsableRegex.FindStringSubmatch(line)
	if matches == nil {
		return yamllintIssue{}, fmt.Errorf("line does not match yamllint parsable format: %q", line)
	}

	lineNum, err := strconv.Atoi(matches[2])
	if err != nil {
		return yamllintIssue{}, fmt.Errorf("failed to parse line number %q: %w", matches[2], err)
	}

	colNum, err := strconv.Atoi(matches[3])
	if err != nil {
		return yamllintIssue{}, fmt.Errorf("failed to parse column number %q: %w", matches[3], err)
	}

	return yamllintIssue{
		File:    matches[1],
		Line:    lineNum,
		Column:  colNum,
		Level:   matches[4],
		Message: matches[5],
		Rule:    matches[6],
	}, nil
}
