package cli

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/timeutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// TaskDomainInfo describes the dominant task type inferred for a workflow run.
type TaskDomainInfo struct {
	Name   string `json:"name"`
	Label  string `json:"label"`
	Reason string `json:"reason,omitempty"`
}

// BehaviorFingerprint summarizes the run's execution profile in compact dimensions.
type BehaviorFingerprint struct {
	ExecutionStyle  string `json:"execution_style"`
	ToolBreadth     string `json:"tool_breadth"`
	ActuationStyle  string `json:"actuation_style"`
	ResourceProfile string `json:"resource_profile"`
	DispatchMode    string `json:"dispatch_mode"`
}

// AgenticAssessment captures an actionable judgment about the run's behavior.
type AgenticAssessment struct {
	Kind           string `json:"kind"`
	Severity       string `json:"severity"`
	Summary        string `json:"summary"`
	Evidence       string `json:"evidence,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

func buildToolUsageInfo(metrics LogMetrics) []ToolUsageInfo {
	toolStats := make(map[string]*ToolUsageInfo)

	for _, toolCall := range metrics.ToolCalls {
		displayKey := workflow.PrettifyToolName(toolCall.Name)
		if existing, exists := toolStats[displayKey]; exists {
			existing.CallCount += toolCall.CallCount
			if toolCall.MaxInputSize > existing.MaxInputSize {
				existing.MaxInputSize = toolCall.MaxInputSize
			}
			if toolCall.MaxOutputSize > existing.MaxOutputSize {
				existing.MaxOutputSize = toolCall.MaxOutputSize
			}
			if toolCall.MaxDuration > 0 {
				maxDuration := timeutil.FormatDuration(toolCall.MaxDuration)
				if existing.MaxDuration == "" || toolCall.MaxDuration > parseDurationString(existing.MaxDuration) {
					existing.MaxDuration = maxDuration
				}
			}
			continue
		}

		info := &ToolUsageInfo{
			Name:          displayKey,
			CallCount:     toolCall.CallCount,
			MaxInputSize:  toolCall.MaxInputSize,
			MaxOutputSize: toolCall.MaxOutputSize,
		}
		if toolCall.MaxDuration > 0 {
			info.MaxDuration = timeutil.FormatDuration(toolCall.MaxDuration)
		}
		toolStats[displayKey] = info
	}

	toolUsage := make([]ToolUsageInfo, 0, len(toolStats))
	for _, info := range toolStats {
		toolUsage = append(toolUsage, *info)
	}

	slices.SortFunc(toolUsage, func(a, b ToolUsageInfo) int {
		if a.CallCount != b.CallCount {
			return b.CallCount - a.CallCount
		}
		return strings.Compare(a.Name, b.Name)
	})

	return toolUsage
}

func deriveRunAgenticAnalysis(processedRun ProcessedRun, metrics LogMetrics) (*AwContext, []ToolUsageInfo, []CreatedItemReport, *TaskDomainInfo, *BehaviorFingerprint, []AgenticAssessment) {
	var awContext *AwContext
	if processedRun.AwContext != nil {
		awContext = processedRun.AwContext
	} else if processedRun.Run.LogsPath != "" {
		awInfoPath := filepath.Join(processedRun.Run.LogsPath, "aw_info.json")
		if info, err := parseAwInfo(awInfoPath, false); err == nil && info != nil {
			awContext = info.Context
		}
	}

	toolUsage := buildToolUsageInfo(metrics)
	createdItems := extractCreatedItemsFromManifest(processedRun.Run.LogsPath)
	metricsData := MetricsData{
		TokenUsage:    processedRun.Run.TokenUsage,
		EstimatedCost: processedRun.Run.EstimatedCost,
		Turns:         processedRun.Run.Turns,
		ErrorCount:    processedRun.Run.ErrorCount,
		WarningCount:  processedRun.Run.WarningCount,
	}

	taskDomain := detectTaskDomain(processedRun, createdItems, toolUsage, awContext)
	behaviorFingerprint := buildBehaviorFingerprint(processedRun, metricsData, toolUsage, createdItems, awContext)
	agenticAssessments := buildAgenticAssessments(processedRun, metricsData, toolUsage, createdItems, taskDomain, behaviorFingerprint, awContext)

	return awContext, toolUsage, createdItems, taskDomain, behaviorFingerprint, agenticAssessments
}

func detectTaskDomain(processedRun ProcessedRun, createdItems []CreatedItemReport, toolUsage []ToolUsageInfo, awContext *AwContext) *TaskDomainInfo {
	combined := strings.ToLower(strings.Join([]string{
		processedRun.Run.WorkflowName,
		processedRun.Run.WorkflowPath,
		processedRun.Run.Event,
	}, " "))

	createdTypes := make([]string, 0, len(createdItems))
	for _, item := range createdItems {
		createdTypes = append(createdTypes, strings.ToLower(item.Type))
	}
	createdJoined := strings.Join(createdTypes, " ")

	toolNames := make([]string, 0, len(toolUsage))
	for _, tool := range toolUsage {
		toolNames = append(toolNames, strings.ToLower(tool.Name))
	}
	toolJoined := strings.Join(toolNames, " ")

	switch {
	case containsAny(combined, "release", "deploy", "publish", "backport", "changelog"):
		return &TaskDomainInfo{Name: "release_ops", Label: "Release / Ops", Reason: "Workflow metadata matches release or operational automation."}
	case containsAny(combined, "research", "investigat", "analysis", "analy", "report", "audit"):
		return &TaskDomainInfo{Name: "research", Label: "Research", Reason: "Workflow naming and instructions suggest exploratory analysis or reporting."}
	case containsAny(combined, "triage", "label", "classif", "route") || containsAny(createdJoined, "add_labels", "remove_labels", "set_issue_type"):
		return &TaskDomainInfo{Name: "triage", Label: "Triage", Reason: "The run focused on classification, routing, or issue state updates."}
	case containsAny(combined, "fix", "patch", "repair", "refactor", "swe", "code", "review") || containsAny(createdJoined, "create_pull_request_review_comment", "submit_pull_request_review"):
		return &TaskDomainInfo{Name: "code_fix", Label: "Code Fix", Reason: "The workflow appears oriented toward code changes or pull request review."}
	case containsAny(combined, "cleanup", "maint", "update", "deps", "sync", "housekeeping"):
		return &TaskDomainInfo{Name: "repo_maintenance", Label: "Repo Maintenance", Reason: "Workflow metadata matches repository maintenance or update work."}
	case containsAny(combined, "issue", "discussion", "comment", "support", "reply") || containsAny(createdJoined, "add_comment", "create_discussion"):
		return &TaskDomainInfo{Name: "issue_response", Label: "Issue Response", Reason: "The run is primarily interacting with issue, discussion, or comment threads."}
	case awContext != nil:
		return &TaskDomainInfo{Name: "delegated_automation", Label: "Delegated Automation", Reason: "The run was dispatched from an upstream workflow and is acting as a delegated task."}
	case containsAny(toolJoined, "github_issue_read", "github-discussion-query"):
		return &TaskDomainInfo{Name: "issue_response", Label: "Issue Response", Reason: "Tool usage centers on repository conversations and issue context."}
	default:
		return &TaskDomainInfo{Name: "general_automation", Label: "General Automation", Reason: "The run does not strongly match a narrower workflow domain yet."}
	}
}

func buildBehaviorFingerprint(processedRun ProcessedRun, metrics MetricsData, toolUsage []ToolUsageInfo, createdItems []CreatedItemReport, awContext *AwContext) *BehaviorFingerprint {
	toolTypes := len(toolUsage)
	writeCount := len(createdItems) + processedRun.Run.SafeItemsCount

	executionStyle := "directed"
	switch {
	case metrics.Turns >= 10 || toolTypes >= 6:
		executionStyle = "exploratory"
	case metrics.Turns >= 5 || toolTypes >= 4:
		executionStyle = "adaptive"
	}

	toolBreadth := "narrow"
	switch {
	case toolTypes >= 6:
		toolBreadth = "broad"
	case toolTypes >= 3:
		toolBreadth = "moderate"
	}

	actuationStyle := "read_only"
	switch {
	case writeCount >= 6:
		actuationStyle = "write_heavy"
	case writeCount > 0:
		actuationStyle = "selective_write"
	}

	resourceProfile := "lean"
	switch {
	case processedRun.Run.Duration >= 15*time.Minute || metrics.Turns >= 12 || toolTypes >= 6 || writeCount >= 8:
		resourceProfile = "heavy"
	case processedRun.Run.Duration >= 5*time.Minute || metrics.Turns >= 6 || toolTypes >= 4 || writeCount >= 3:
		resourceProfile = "moderate"
	}

	dispatchMode := "standalone"
	if awContext != nil {
		dispatchMode = "delegated"
	}

	return &BehaviorFingerprint{
		ExecutionStyle:  executionStyle,
		ToolBreadth:     toolBreadth,
		ActuationStyle:  actuationStyle,
		ResourceProfile: resourceProfile,
		DispatchMode:    dispatchMode,
	}
}

func buildAgenticAssessments(processedRun ProcessedRun, metrics MetricsData, toolUsage []ToolUsageInfo, createdItems []CreatedItemReport, domain *TaskDomainInfo, fingerprint *BehaviorFingerprint, awContext *AwContext) []AgenticAssessment {
	if domain == nil || fingerprint == nil {
		return nil
	}

	assessments := make([]AgenticAssessment, 0, 4)
	toolTypes := len(toolUsage)
	frictionEvents := len(processedRun.MissingTools) + len(processedRun.MCPFailures) + len(processedRun.MissingData)
	writeCount := len(createdItems) + processedRun.Run.SafeItemsCount

	if fingerprint.ResourceProfile == "heavy" {
		severity := "medium"
		if metrics.Turns >= 14 || toolTypes >= 7 || processedRun.Run.Duration >= 20*time.Minute {
			severity = "high"
		}
		assessments = append(assessments, AgenticAssessment{
			Kind:           "resource_heavy_for_domain",
			Severity:       severity,
			Summary:        fmt.Sprintf("This %s run consumed a heavy execution profile for its task shape.", domain.Label),
			Evidence:       fmt.Sprintf("turns=%d tool_types=%d duration=%s write_actions=%d", metrics.Turns, toolTypes, formatAssessmentDuration(processedRun.Run.Duration), writeCount),
			Recommendation: "Compare this run to similar successful runs and trim unnecessary turns, tools, or write actions.",
		})
	}

	if (domain.Name == "triage" || domain.Name == "repo_maintenance" || domain.Name == "issue_response") && fingerprint.ResourceProfile == "lean" && fingerprint.ExecutionStyle == "directed" && fingerprint.ToolBreadth == "narrow" {
		assessments = append(assessments, AgenticAssessment{
			Kind:           "overkill_for_agentic",
			Severity:       "low",
			Summary:        fmt.Sprintf("This %s run looks stable enough that deterministic automation may be a simpler fit.", domain.Label),
			Evidence:       fmt.Sprintf("turns=%d tool_types=%d actuation=%s", metrics.Turns, toolTypes, fingerprint.ActuationStyle),
			Recommendation: "Consider whether a scripted rule or deterministic workflow step could replace this agentic path.",
		})
	}

	if frictionEvents >= 3 || (frictionEvents > 0 && writeCount >= 3) || ((domain.Name == "triage" || domain.Name == "repo_maintenance" || domain.Name == "issue_response") && fingerprint.ExecutionStyle == "exploratory") {
		severity := "medium"
		if frictionEvents >= 4 || (frictionEvents > 0 && fingerprint.ActuationStyle == "write_heavy") {
			severity = "high"
		}
		assessments = append(assessments, AgenticAssessment{
			Kind:           "poor_agentic_control",
			Severity:       severity,
			Summary:        "The run showed signs of broad or weakly controlled agentic behavior.",
			Evidence:       fmt.Sprintf("friction=%d execution=%s actuation=%s", frictionEvents, fingerprint.ExecutionStyle, fingerprint.ActuationStyle),
			Recommendation: "Tighten instructions, reduce unnecessary tools, or delay write actions until the workflow has stronger evidence.",
		})
	}

	if awContext != nil {
		assessments = append(assessments, AgenticAssessment{
			Kind:           "delegated_context_present",
			Severity:       "info",
			Summary:        "The run preserved upstream dispatch context, which helps trace multi-workflow episodes.",
			Evidence:       fmt.Sprintf("workflow_call_id=%s event_type=%s", awContext.WorkflowCallID, awContext.EventType),
			Recommendation: "Use this context when comparing downstream runs so follow-up workflows are evaluated as part of one task chain.",
		})
	}

	return assessments
}

func generateAgenticAssessmentFindings(assessments []AgenticAssessment) []Finding {
	findings := make([]Finding, 0, len(assessments))
	for _, assessment := range assessments {
		category := "agentic"
		impact := "Review recommended"
		switch assessment.Kind {
		case "resource_heavy_for_domain":
			category = "performance"
			impact = "Higher cost and latency than a comparable well-behaved run"
		case "overkill_for_agentic":
			category = "optimization"
			impact = "A deterministic implementation may be cheaper and easier to govern"
		case "poor_agentic_control":
			category = "agentic"
			impact = "Broad or weakly controlled behavior can reduce trust even when the run succeeds"
		case "delegated_context_present":
			category = "coordination"
			impact = "Context continuity improves downstream debugging and auditability"
		}
		findings = append(findings, Finding{
			Category:    category,
			Severity:    assessment.Severity,
			Title:       prettifyAssessmentKind(assessment.Kind),
			Description: assessment.Summary,
			Impact:      impact,
		})
	}
	return findings
}

func generateAgenticAssessmentRecommendations(assessments []AgenticAssessment) []Recommendation {
	recommendations := make([]Recommendation, 0, len(assessments))
	for _, assessment := range assessments {
		if assessment.Recommendation == "" || assessment.Severity == "info" {
			continue
		}
		priority := "medium"
		if assessment.Severity == "high" {
			priority = "high"
		}
		recommendations = append(recommendations, Recommendation{
			Priority: priority,
			Action:   assessment.Recommendation,
			Reason:   assessment.Summary,
		})
	}
	return recommendations
}

func containsAny(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

func prettifyAssessmentKind(kind string) string {
	switch kind {
	case "resource_heavy_for_domain":
		return "Resource Heavy For Domain"
	case "overkill_for_agentic":
		return "Potential Deterministic Alternative"
	case "poor_agentic_control":
		return "Weak Agentic Control"
	case "delegated_context_present":
		return "Dispatch Context Preserved"
	default:
		return strings.ReplaceAll(kind, "_", " ")
	}
}

func formatAssessmentDuration(duration time.Duration) string {
	if duration <= 0 {
		return "n/a"
	}
	return duration.String()
}
