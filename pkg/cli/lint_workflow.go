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

type workflowLintFile struct {
	name  string
	input workflowlint.Input
}

func runWorkflowLint(ctx context.Context, files []string, runner *workflowlint.Runner, output io.Writer, jsonOutput bool, options ...workflowLintOptions) error {
	var configuration workflowLintOptions
	if len(options) > 0 {
		configuration = options[0]
	}
	diagnostics := make([]workflowLintDiagnostic, 0)
	var failures []error
	inputs, loadErr := prepareWorkflowLintFiles(ctx, files, runner, configuration)
	if errors.Is(loadErr, context.Canceled) || errors.Is(loadErr, context.DeadlineExceeded) {
		return loadErr
	}
	failures = append(failures, loadErr)
	for _, prepared := range inputs {
		if err := ctx.Err(); err != nil {
			failures = append(failures, err)
			break
		}
		file, input := prepared.name, prepared.input
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

func prepareWorkflowLintFiles(ctx context.Context, files []string, runner *workflowlint.Runner, configuration workflowLintOptions) ([]workflowLintFile, error) {
	inputs := make([]workflowLintFile, 0, len(files))
	var documents []*aw.Document
	var failures []error
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		input, err := loadWorkflowLintInput(file, configuration.JavaScript)
		inputs = append(inputs, workflowLintFile{name: file, input: input})
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", file, err))
		}
		if input.Document != nil {
			documents = append(documents, input.Document)
		}
	}
	checksJavaScript := slices.ContainsFunc(runner.Rules(), func(rule workflowlint.Rule) bool { return rule.ID == "secret-leak" })
	if checksJavaScript && configuration.JavaScript != nil && configuration.JavaScript.Mode == taint.JavaScriptFast {
		err := taint.PrimeAuthoredJavaScript(ctx, documents, taint.SecretLeakRule(), *configuration.JavaScript)
		failures = append(failures, err)
	}
	return inputs, errors.Join(failures...)
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
	input := workflowlint.Input{}
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
			} else {
				failures = append(failures, errors.New("could not parse companion AW AST"))
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
