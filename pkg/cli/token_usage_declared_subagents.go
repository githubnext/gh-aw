package cli

import (
	"encoding/json"
	"os"
	"slices"
	"strconv"
	"strings"
)

type declaredSubagentModel struct {
	Name     string   `json:"name"`
	Model    string   `json:"model"`
	Patterns []string `json:"patterns"`
}

// readDeclaredSubagentModels returns the declared sub-agent models and the main model
// recorded in aw_info.json.
func readDeclaredSubagentModels(runDir string, summary *TokenUsageSummary) ([]declaredSubagentModel, string) {
	infoPath := findAwInfoPath(runDir)
	if infoPath == "" {
		return nil, ""
	}
	content, err := os.ReadFile(infoPath)
	if err != nil {
		addTokenUsageWarning(summary, "cannot read declared sub-agent models: "+err.Error())
		return nil, ""
	}
	var info struct {
		Model  string                  `json:"model"`
		Models []declaredSubagentModel `json:"sub_agent_models"`
	}
	if err := json.Unmarshal(content, &info); err != nil {
		addTokenUsageWarning(summary, "cannot parse declared sub-agent models: "+err.Error())
		return nil, ""
	}
	return info.Models, strings.TrimSpace(info.Model)
}

func augmentDeclaredSubagentModels(runDir string, summary *TokenUsageSummary) {
	models, mainModel := readDeclaredSubagentModels(runDir, summary)
	if len(models) == 0 {
		return
	}
	resolver := newModelIdentityResolver(runDir)
	parentModels := effectiveParentModels(summary, mainModel)
	observedModels := subagentObservedModels(summary)
	observedNames := make([]string, 0, len(observedModels))
	for name := range observedModels {
		observedNames = append(observedNames, name)
	}
	slices.Sort(observedNames)
	for _, model := range models {
		row, sessionObserved := declaredSessionModelRow(model, summary.SubagentModelRequests, resolver, observedNames, parentModels)
		if !sessionObserved {
			row = declaredTrafficModelRow(model, observedModels, observedNames, resolver, parentModels)
		}
		summary.DeclaredSubagentModels = append(summary.DeclaredSubagentModels, row)
	}
}

// effectiveParentModels returns the models the main agent ran on, which an inherited
// sub-agent model uses: the main agent's served session models when the session records
// them, otherwise the main model from aw_info.json.
func effectiveParentModels(summary *TokenUsageSummary, mainModel string) []string {
	var models []string
	for _, agent := range summary.AgentUsage {
		if agent.AgentType != "main" {
			continue
		}
		for _, usage := range agent.Models {
			if usage.Requests > 0 {
				models = appendUnique(models, usage.Model)
			}
		}
		for _, served := range agent.ServedModels {
			models = appendUnique(models, served)
		}
	}
	if len(models) == 0 && mainModel != "" {
		models = append(models, mainModel)
	}
	return models
}

// isInheritedSubagentModel reports whether a declared model is an inheritance instruction
// (Claude's `inherit`), which means the sub-agent runs on the parent model.
func isInheritedSubagentModel(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "inherit", "inherited":
		return true
	}
	return false
}

func declaredSessionModelRow(model declaredSubagentModel, requests []SubagentModelRequest, resolver *modelIdentityResolver, observedNames []string, parentModels []string) (SubagentModelRequest, bool) {
	row := SubagentModelRequest{AgentName: model.Name, RequestedModel: model.Model}
	for _, request := range requests {
		if !strings.EqualFold(request.AgentName, model.Name) {
			continue
		}
		row.InvocationCount += request.InvocationCount
		row.CompletedCount += request.CompletedCount
		row.FailedCount += request.FailedCount
		row.IncompleteCount += request.IncompleteCount
		row.Effort = combineSubagentEffort(row.Effort, request.Effort)
		if row.Error == "" {
			row.Error = request.Error
		}
		if request.EffectiveModel == "" {
			continue
		}
		for _, served := range request.ServedModels {
			row.ServedModels = appendUnique(row.ServedModels, served)
		}
		row.ServedModels = appendUnique(row.ServedModels, request.EffectiveModel)
		if declaredModelMatches(model, request.EffectiveModel, "", resolver, parentModels) {
			row.EffectiveModel = request.EffectiveModel
			row.ResolvedModel = firstNonEmptyModel(request.ResolvedModel, resolver.resolve(request.EffectiveModel, "", observedNames))
		}
	}
	if row.InvocationCount == 0 {
		return row, false
	}
	if row.EffectiveModel == "" {
		if row.FailedCount > 0 && row.CompletedCount == 0 {
			row.ReasonCode = modelMismatchReasonSubagentFailed
		} else {
			row.ReasonCode = modelMismatchReasonModelNotObserved
		}
	}
	return row, true
}

func combineSubagentEffort(current, next string) string {
	if current == "mixed" || next == "mixed" {
		return "mixed"
	}
	if current == "" {
		return next
	}
	if next != "" && current != next {
		return "mixed"
	}
	return current
}

func declaredTrafficModelRow(model declaredSubagentModel, observedModels map[string]*ModelTokenUsage, observedNames []string, resolver *modelIdentityResolver, parentModels []string) SubagentModelRequest {
	row := SubagentModelRequest{AgentName: model.Name, RequestedModel: model.Model}
	for _, observed := range observedNames {
		usage := observedModels[observed]
		if usage == nil || usage.Requests == 0 {
			continue
		}
		if declaredModelMatches(model, observed, usage.Provider, resolver, parentModels) {
			row.EffectiveModel = normalizeModelIdentity(observed)
			row.ResolvedModel = row.EffectiveModel
			row.ServedModels = []string{observed}
			break
		}
	}
	if row.EffectiveModel == "" {
		row.ReasonCode = modelMismatchReasonModelNotObserved
		if len(observedModels) == 0 {
			row.ReasonCode = modelMismatchReasonTokenUsageMissing
		}
	}
	return row
}

// declaredModelMatches reports whether an observed model satisfies a declaration. An
// inherited declaration matches the effective parent models.
func declaredModelMatches(model declaredSubagentModel, observed, provider string, resolver *modelIdentityResolver, parentModels []string) bool {
	patterns := model.Patterns
	if isInheritedSubagentModel(model.Model) {
		patterns = parentModels
	} else if len(patterns) == 0 {
		patterns = []string{model.Model}
	}
	for _, pattern := range patterns {
		if resolver.matches(pattern, observed, provider) {
			return true
		}
	}
	return false
}

func subagentObservedModels(summary *TokenUsageSummary) map[string]*ModelTokenUsage {
	if summary.agentModels != nil {
		return summary.agentModels
	}
	return summary.ByModel
}

func generateSubagentModelFindings(summary *TokenUsageSummary) []AuditFinding {
	if summary == nil {
		return nil
	}
	var findings []AuditFinding
	for _, model := range summary.SubagentModelRequests {
		if model.FailedCount == 0 {
			continue
		}
		description := "Sub-agent " + model.AgentName + " failed " + strconv.Itoa(model.FailedCount) + " invocation(s)"
		if model.Error != "" {
			description += ": " + model.Error
		}
		findings = append(findings, AuditFinding{
			Code: AuditFindingSubagentFailed, Category: "tooling", Severity: "high",
			Title: "Sub-agent Failed", Description: description,
			Impact: "The delegated task failed and may have reduced the completeness or correctness of the workflow result.",
		})
	}
	for _, model := range summary.DeclaredSubagentModels {
		if model.ReasonCode != modelMismatchReasonModelNotObserved {
			continue
		}
		findings = append(findings, AuditFinding{
			Code:     AuditFindingSubagentModelNotObserved,
			Category: "tooling", Severity: "medium",
			Title:       "Declared Sub-agent Model Not Observed",
			Description: "No observed model requests match sub-agent " + model.AgentName + " model " + model.RequestedModel,
			Impact:      "The sub-agent may not have been invoked or its model was not honored. Model presence alone does not prove delegation.",
		})
	}
	return findings
}
