package cli

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/scanfindings"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

type scannerDiagnostic struct {
	Finding  scanfindings.Finding
	Identity string
}

// scannerFindingsError distinguishes findings from execution and parsing failures.
type scannerFindingsError struct {
	err         error
	diagnostics []scannerDiagnostic
}

func (e *scannerFindingsError) Error() string { return e.err.Error() }
func (e *scannerFindingsError) Unwrap() error { return e.err }

func withScannerFindings(err error, findings []scanfindings.Finding) error {
	if err == nil || len(findings) == 0 {
		return err
	}
	diagnostics := make([]scannerDiagnostic, 0, len(findings))
	for _, finding := range findings {
		diagnostics = append(diagnostics, scannerDiagnostic{
			Finding: finding, Identity: scannerFindingIdentity(finding),
		})
	}
	return &scannerFindingsError{err: err, diagnostics: diagnostics}
}

type dryRunScannerBaseline struct {
	results map[string]error
}

// DryRunScannerDifferential reports the finding-level comparison used by the gate.
type DryRunScannerDifferential struct {
	Baseline int `json:"baseline"`
	Existing int `json:"existing"`
	Added    int `json:"added"`
	Ignored  int `json:"ignored"`
}

func (b *dryRunScannerBaseline) filter(tool string, err error) (*DryRunScannerDifferential, error) {
	if baselineErr := b.results[tool]; baselineErr != nil {
		var baseline *scannerFindingsError
		if !errors.As(baselineErr, &baseline) {
			return nil, errors.Join(fmt.Errorf("%s baseline scan failed: %w", tool, baselineErr), err)
		}
	}
	var current *scannerFindingsError
	if err != nil && !errors.As(err, &current) {
		return nil, err
	}
	diff := &DryRunScannerDifferential{}
	counts := make(map[string]int)
	var baseline *scannerFindingsError
	if errors.As(b.results[tool], &baseline) {
		diff.Baseline = len(baseline.diagnostics)
		for _, diagnostic := range baseline.diagnostics {
			counts[diagnostic.Identity]++
		}
	}
	var added []scanfindings.Finding
	if current != nil {
		for _, diagnostic := range current.diagnostics {
			finding := diagnostic.Finding
			if finding.Severity == scanfindings.SeverityLow || finding.Severity == scanfindings.SeverityInfo {
				diff.Ignored++
			} else if counts[diagnostic.Identity] > 0 {
				counts[diagnostic.Identity]--
				diff.Existing++
			} else {
				diff.Added++
				added = append(added, finding)
			}
		}
	}
	if len(added) > 0 {
		return diff, withScannerFindings(fmt.Errorf("%s found %d added medium-or-higher or unclassified finding(s)", tool, len(added)), added)
	}
	return diff, nil
}

func prepareDryRunScannerBaseline(ctx context.Context, config *CompileConfig, files []string, workflowDir string) error {
	if !config.DryRun {
		return nil
	}
	baseline := &dryRunScannerBaseline{results: make(map[string]error)}
	config.dryRunBaseline = baseline
	var lockFiles []string
	for _, file := range files {
		resolved, err := resolveWorkflowFile(file, false)
		if err != nil {
			continue // The ordinary compile pipeline reports source resolution errors.
		}
		lockFile := stringutil.MarkdownToLockFile(resolved)
		info, err := os.Stat(lockFile)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("cannot read dry-run baseline %s: %w", lockFile, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("dry-run baseline %s must be a regular lock file", lockFile)
		}
		lockFiles = append(lockFiles, lockFile)
	}
	if len(lockFiles) == 0 {
		return nil
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr("Scanning pre-compilation lock files for the dry-run warning baseline"))
	baselineConfig := *config
	baselineConfig.DryRun = false
	baselineConfig.Strict = true
	baselineConfig.JSONOutput = true
	opts := batchToolsOptions{
		workflowDir: workflowDir, dryRunZizmor: true,
		lockFilesForActionlint: lockFiles, lockFilesForZizmor: lockFiles,
		lockFilesForDirTools: lockFiles, lockFilesForSyft: lockFiles,
		lockFilesForGrype: lockFiles, lockFilesForGrant: lockFiles,
		lockFilesForYamllint: lockFiles, lockFilesForShellcheck: lockFiles,
		reportResult: func(tool string, _ int, err error) { baseline.results[tool] = err },
	}
	var results []ValidationResult
	_, err := runBatchExternalTools(ctx, baselineConfig, opts, &CompilationStats{}, &results)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// Findings and scanner failures are retained separately for the final gate.
	compileDevelopmentLog.Printf("Captured dry-run scanner baseline: tools=%d, diagnostics=%t", len(baseline.results), err != nil)
	return nil
}

var scannerStepIndexPattern = regexp.MustCompile(`\.steps\[(\d+)\]`)

func scannerFindingIdentity(finding scanfindings.Finding) string {
	file := filepath.Clean(finding.File)
	if abs, err := filepath.Abs(file); err == nil {
		file = abs
	}
	anchor := strconv.Itoa(finding.Line) + ":" + strconv.Itoa(finding.Column)
	content, err := os.ReadFile(file)
	if err == nil && (strings.HasSuffix(file, ".yml") || strings.HasSuffix(file, ".yaml")) {
		if parsed, err := parser.ParseBytes(content, 0); err == nil {
			visitor := &scannerLocationVisitor{line: finding.Line}
			for _, doc := range parsed.Docs {
				ast.Walk(visitor, doc.Body)
			}
			lines := strings.Split(string(content), "\n")
			lineIndex := finding.Line - 1
			if visitor.path != "" && lineIndex >= 0 && lineIndex < len(lines) {
				path := scannerStepIndexPattern.ReplaceAllStringFunc(visitor.path, func(match string) string {
					prefix := visitor.path[:strings.Index(visitor.path, match)] + match
					for _, field := range []string{"id", "name", "uses", "run"} {
						p, pathErr := yaml.PathString(prefix + "." + field)
						if pathErr != nil {
							continue
						}
						var value string
						if p.Read(strings.NewReader(string(content)), &value) == nil && value != "" {
							return fmt.Sprintf(".steps[%s:%x]", field, sha256.Sum256([]byte(value)))
						}
					}
					return match
				})
				anchor = path + "\x00" + strings.TrimSpace(lines[lineIndex])
			}
		}
	}
	return strings.Join([]string{file, finding.RuleID, finding.Severity.String(), finding.Message, anchor}, "\x00")
}

type scannerLocationVisitor struct {
	line int
	path string
	row  int
}

func (v *scannerLocationVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil || node.GetToken() == nil {
		return v
	}
	row := node.GetToken().Position.Line
	if row <= v.line && (row > v.row || row == v.row && len(node.GetPath()) > len(v.path)) {
		v.row = row
		v.path = node.GetPath()
	}
	return v
}
