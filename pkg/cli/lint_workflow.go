package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
	"github.com/github/gh-aw/pkg/workflow/taint"
)

type workflowLintDiagnostic struct {
	File string `json:"file"`
	workflowlint.Diagnostic
}

type workflowLintOptions struct {
	JavaScript *taint.JavaScriptOptions
	Debug      bool
}

func runWorkflowLint(ctx context.Context, files []string, runner *workflowlint.Runner, output io.Writer, jsonOutput bool, options ...workflowLintOptions) error {
	var configuration workflowLintOptions
	if len(options) > 0 {
		configuration = options[0]
	}
	diagnostics := make([]workflowLintDiagnostic, 0)
	var failures []error
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			failures = append(failures, err)
			break
		}
		current, err := lintWorkflowFile(ctx, file, runner, configuration.JavaScript)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", file, err))
		}
		for _, diagnostic := range current {
			if diagnostic.RuleID == "secret-leak" && diagnostic.Severity == workflowlint.Warning && !configuration.Debug {
				continue
			}
			diagnostics = append(diagnostics, workflowLintDiagnostic{File: file, Diagnostic: diagnostic})
		}
	}
	findings := 0
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == workflowlint.Error {
			findings++
		}
		if !jsonOutput {
			if _, err := fmt.Fprintf(output, "%s: %s [%s] %s at %s (script line %d)\n",
				diagnostic.File, diagnostic.Severity, diagnostic.RuleID, diagnostic.Message, diagnostic.Path, diagnostic.Line); err != nil {
				return err
			}
		}
	}
	if jsonOutput {
		if err := json.NewEncoder(output).Encode(diagnostics); err != nil {
			return err
		}
	}
	if findings > 0 {
		failures = append(failures, fmt.Errorf("workflow lint found %d error(s)", findings))
	}
	return errors.Join(failures...)
}

func lintWorkflowFile(ctx context.Context, file string, runner *workflowlint.Runner, javascript ...*taint.JavaScriptOptions) ([]workflowlint.Diagnostic, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	workflow, err := actions.Parse(content)
	if err != nil {
		// Parser errors can contain YAML scalar values. Keep lint diagnostics
		// free of values that may themselves be credentials.
		return nil, errors.New("could not parse workflow Actions AST")
	}
	input := workflowlint.Input{Workflow: workflow}
	if len(javascript) > 0 {
		input.JavaScript = javascript[0]
	}
	// A companion AW source adds prompt analysis without recompiling or changing
	// the on-disk execution graph. Native Actions files need no companion.
	source, readErr := os.ReadFile(strings.TrimSuffix(file, ".lock.yml") + ".md")
	if readErr == nil {
		document, parseErr := aw.Parse(string(source))
		if parseErr != nil {
			return nil, errors.New("could not parse companion AW document")
		}
		input.Document = document
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return nil, fmt.Errorf("reading companion AW document: %w", readErr)
	}
	return runner.Run(ctx, input)
}
