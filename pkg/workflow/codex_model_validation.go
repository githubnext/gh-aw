package workflow

import (
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
)

func supportedCodexModels() []string {
	def, err := getBuiltinEngineDefinition(string(constants.CodexEngine))
	if err == nil && len(def.Models.Supported) > 0 {
		return append([]string(nil), def.Models.Supported...)
	}
	return []string{constants.CodexDefaultModel}
}

func isCodexAlphaSnapshotModel(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(normalized, "gpt-5-codex-alpha")
}

func validateCodexModelName(model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil
	}
	if isCodexAlphaSnapshotModel(model) {
		return fmt.Errorf("codex model %q is not supported. Alpha snapshot names like gpt-5-codex-alpha-* have been decommissioned; set engine.model to one of: %s",
			model, strings.Join(supportedCodexModels(), ", "))
	}
	if slices.Contains(supportedCodexModels(), model) {
		return nil
	}
	if suggestion := parser.FindClosestMatches(model, supportedCodexModels(), 1); len(suggestion) > 0 {
		return fmt.Errorf("codex model %q is not supported. Set engine.model to one of: %s. Did you mean %q?",
			model, strings.Join(supportedCodexModels(), ", "), suggestion[0])
	}
	return fmt.Errorf("codex model %q is not supported. Set engine.model to one of: %s",
		model, strings.Join(supportedCodexModels(), ", "))
}

func buildCodexModelValidationCommand(modelEnvVar string) string {
	supported := supportedCodexModels()
	if len(supported) == 0 {
		return ""
	}
	allowedPatterns := strings.Join(supported, "|")
	supportedList := strings.Join(supported, ", ")
	return fmt.Sprintf(`case "$%[1]s" in %[2]s) ;; *gpt-5-codex-alpha*) printf '%%s\n' "Invalid Codex model \"$%[1]s\". Alpha snapshot names like gpt-5-codex-alpha-* have been decommissioned. Set engine.model in workflow frontmatter (or the variable that feeds it) to one of: %[3]s." >&2; exit 1 ;; *) printf '%%s\n' "Invalid Codex model \"$%[1]s\". Set engine.model in workflow frontmatter (or the variable that feeds it) to one of: %[3]s." >&2; exit 1 ;; esac && `,
		modelEnvVar, allowedPatterns, supportedList)
}
