package cli

import (
	"fmt"
	"path/filepath"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
)

var postStepsSafeOutputsCodemodLog = logger.New("cli:codemod_post_steps_safe_outputs")

// getPostStepsSafeOutputsGuidedErrorCodemod creates a codemod that emits a guided error when
// post-steps directly access safe-outputs files (such as $GH_AW_SAFE_OUTPUTS or agent_output.json)
// instead of using the supported safe-outputs mechanism.
func getPostStepsSafeOutputsGuidedErrorCodemod() Codemod {
	return Codemod{
		ID:           "post-steps-safe-outputs-guided-error",
		Name:         "Detect post-steps accessing safe-outputs file directly (manual fix required)",
		Description:  "Detects post-steps that read the safe-outputs output file directly instead of using supported safe-outputs mechanisms, and emits a guided error pointing users to the safe-outputs documentation.",
		IntroducedIn: "1.5.0",
		Guided:       true,
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			return applyPostStepsSafeOutputsCheck(content, frontmatter, "")
		},
		ApplyWithContext: func(content string, frontmatter map[string]any, filePath string) (string, bool, error) {
			return applyPostStepsSafeOutputsCheck(content, frontmatter, filePath)
		},
	}
}

// applyPostStepsSafeOutputsCheck is the shared implementation used by both Apply and
// ApplyWithContext. When filePath is non-empty, effective post-steps are resolved from imports
// so that violations introduced via shared imports are also caught.
func applyPostStepsSafeOutputsCheck(content string, frontmatter map[string]any, filePath string) (string, bool, error) {
	allPostSteps, err := resolveEffectivePostSteps(content, frontmatter, filePath)
	if err != nil {
		postStepsSafeOutputsCodemodLog.Printf("Failed to resolve effective post-steps: %v", err)
		if rawSteps, ok := frontmatter["post-steps"].([]any); ok {
			allPostSteps = rawSteps
		}
	}

	if len(allPostSteps) == 0 {
		return content, false, nil
	}

	if err := workflow.ValidatePostStepsSafeOutputs(allPostSteps); err != nil {
		postStepsSafeOutputsCodemodLog.Printf("Detected safe-outputs violation in post-steps: %v", err)
		return content, false, err
	}

	return content, false, nil
}

// resolveEffectivePostSteps returns all post-steps for the workflow, including any imported
// post-steps when a file path is available.
func resolveEffectivePostSteps(content string, frontmatter map[string]any, filePath string) ([]any, error) {
	var allPostSteps []any
	if rawSteps, ok := frontmatter["post-steps"].([]any); ok {
		allPostSteps = append(allPostSteps, rawSteps...)
	}

	if filePath == "" {
		return allPostSteps, nil
	}

	if _, hasImports := frontmatter["imports"]; !hasImports {
		return allPostSteps, nil
	}

	baseDir := filepath.Dir(filePath)
	importCache := parser.NewImportCache("")
	importsResult, err := parser.ProcessImportsFromFrontmatterWithSource(frontmatter, baseDir, importCache, filePath, content)
	if err != nil {
		return nil, fmt.Errorf("resolving imports: %w", err)
	}

	if importsResult != nil && importsResult.MergedPostSteps != "" {
		var importedPostSteps []any
		if err := yaml.Unmarshal([]byte(importsResult.MergedPostSteps), &importedPostSteps); err != nil {
			return nil, fmt.Errorf("parsing imported post-steps: %w", err)
		}
		allPostSteps = append(allPostSteps, importedPostSteps...)
	}

	return allPostSteps, nil
}
