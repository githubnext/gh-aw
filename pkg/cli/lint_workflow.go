package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
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
	if err := primeWorkflowLintJavaScript(ctx, files, runner, configuration); err != nil {
		return err
	}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			failures = append(failures, err)
			break
		}
		input, loadErr := loadWorkflowLintInput(file, configuration.JavaScript)
		if loadErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", file, loadErr))
		}
		if input.Document == nil && input.Workflow == nil {
			continue
		}
		current, err := runner.Run(ctx, input)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", file, err))
		}
		for _, diagnostic := range current {
			if diagnostic.RuleID == "secret-leak" && diagnostic.Severity == workflowlint.Warning && !configuration.Debug && configuration.JavaScript == nil {
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

func primeWorkflowLintJavaScript(ctx context.Context, files []string, runner *workflowlint.Runner, configuration workflowLintOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if configuration.JavaScript == nil || configuration.JavaScript.Mode != taint.JavaScriptFast {
		return nil
	}
	if _, ok := configuration.JavaScript.Parser.(taint.JavaScriptBatchParser); !ok {
		return nil
	}
	checksJavaScript := slices.ContainsFunc(runner.Rules(), func(rule workflowlint.Rule) bool { return rule.ID == "secret-leak" })
	if !checksJavaScript {
		return nil
	}
	var documents []*aw.Document
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		sourceFile := file
		if !strings.HasSuffix(file, ".md") {
			base, paired := strings.CutSuffix(file, ".lock.yml")
			if !paired {
				continue
			}
			sourceFile = base + ".md"
		}
		source, err := os.ReadFile(sourceFile)
		if err != nil {
			continue
		}
		document, err := aw.Parse(string(source))
		if err == nil {
			documents = append(documents, document)
		}
	}
	// Priming is only an optimization; per-file analysis reports failures.
	_ = taint.PrimeAuthoredJavaScript(ctx, documents, taint.SecretLeakRule(), *configuration.JavaScript)
	return ctx.Err()
}

func lintWorkflowFile(ctx context.Context, file string, runner *workflowlint.Runner, javascript ...*taint.JavaScriptOptions) ([]workflowlint.Diagnostic, error) {
	input, loadErr := loadWorkflowLintInput(file, javascript...)
	if input.Workflow == nil && input.Document == nil {
		return nil, loadErr
	}
	diagnostics, err := runner.Run(ctx, input)
	return diagnostics, errors.Join(loadErr, err)
}

func loadWorkflowLintInput(file string, javascript ...*taint.JavaScriptOptions) (workflowlint.Input, error) {
	input := workflowlint.Input{SourcePath: file}
	content, err := os.ReadFile(file)
	if err != nil {
		return input, err
	}
	if len(javascript) > 0 {
		input.JavaScript = javascript[0]
	}
	if strings.HasSuffix(file, ".md") {
		input.Document, err = aw.Parse(string(content))
		if err != nil {
			return input, errors.New("could not parse workflow AW AST")
		}
		return input, nil
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
				input.SourcePath = base + ".md"
			}
		case !errors.Is(readErr, os.ErrNotExist):
			failures = append(failures, fmt.Errorf("reading companion AW document: %w", readErr))
		}
	}
	if input.Workflow == nil && input.Document == nil {
		return input, errors.Join(failures...)
	}
	return input, errors.Join(failures...)
}
