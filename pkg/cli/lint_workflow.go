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
			threat := ""
			if diagnostic.ThreatRuleID != "" {
				threat = " (" + diagnostic.ThreatRuleID + ")"
			}
			if _, err := fmt.Fprintf(output, "%s: %s [%s]%s %s at %s (script line %d)\n",
				diagnostic.File, diagnostic.Severity, diagnostic.RuleID, threat, diagnostic.Message, diagnostic.Path, diagnostic.Line); err != nil {
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
	input := workflowlint.Input{}
	if len(javascript) > 0 {
		input.JavaScript = javascript[0]
	}
	if strings.HasSuffix(file, ".md") {
		input.Document, err = aw.Parse(string(content))
		if err != nil {
			return nil, errors.New("could not parse workflow AW AST")
		}
		return runner.Run(ctx, input)
	}
	var failures []error
	input.Workflow, err = actions.Parse(content)
	if err != nil {
		// Parser errors can contain YAML scalar values or credentials.
		failures = append(failures, errors.New("could not parse workflow Actions AST"))
	}
	if base, paired := strings.CutSuffix(file, ".lock.yml"); paired {
		source, readErr := os.ReadFile(base + ".md")
		switch {
		case readErr == nil:
			document, parseErr := aw.Parse(string(source))
			if parseErr == nil {
				input.Document = document
			}
		case !errors.Is(readErr, os.ErrNotExist):
			failures = append(failures, fmt.Errorf("reading companion AW document: %w", readErr))
		}
	}
	if input.Workflow == nil && input.Document == nil {
		return nil, errors.Join(failures...)
	}
	diagnostics, err := runner.Run(ctx, input)
	return diagnostics, errors.Join(append(failures, err)...)
}
