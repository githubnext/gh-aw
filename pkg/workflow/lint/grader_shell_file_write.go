package lint

import (
	"context"
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/workflow/gradershell"
)

func GraderShellFileWriteRule() Rule {
	return Rule{
		ID: "grader-shell-file-write", Description: "Detect direct file writes and unchecked shell execution in operational-value Bash graders (not a sandbox)",
		Severity: Error, Targets: AWAST, Check: checkGraderShellFileWrite,
	}
}

func checkGraderShellFileWrite(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	if input.Document == nil {
		return nil, nil
	}
	graders, err := policyMapping(input.Document.Frontmatter["graders"], "graders")
	if err != nil {
		return nil, err
	}
	fields, ok := graders["operational-value"].(map[string]any)
	if !ok {
		return nil, nil
	}
	value, exists := fields["run"]
	if !exists {
		return nil, nil
	}
	const path = "graders.operational-value.run"
	runPath, ok := value.(string)
	if !ok {
		return nil, errors.New("shell grader run requires a literal .sh path at " + path)
	}
	source, err := gradershell.Load(input.SourcePath, runPath)
	if err != nil {
		return nil, fmt.Errorf("cannot analyze %s: %w", path, err)
	}
	findings, err := gradershell.Check(ctx, source)
	diagnostics := make([]Diagnostic, 0, len(findings))
	for _, finding := range findings {
		diagnostics = append(diagnostics, Diagnostic{Path: path, Line: finding.Line, Message: finding.Message})
	}
	if err != nil {
		return diagnostics, fmt.Errorf("cannot analyze %s: %w", path, err)
	}
	return diagnostics, nil
}
