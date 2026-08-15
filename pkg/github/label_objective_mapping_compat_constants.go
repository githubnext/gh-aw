package github

// Deprecated: compatibility constants preserved for external consumers that
// previously referenced ObjectiveLabel*/ObjectiveValue* identifiers.
//
// These constants are not used by runtime scoring. Runtime defaults are defined
// solely by DefaultObjectiveMapping().

// Critical Priority Labels.
const (
	ObjectiveLabelCritical = "critical"
	ObjectiveLabelP0       = "p0"
	ObjectiveValueCritical = 100
	ObjectiveValueP0       = 100
)

// Safety-Critical Work.
const (
	ObjectiveLabelTesting     = "testing"
	ObjectiveLabelReliability = "reliability"
	ObjectiveValueTesting     = 50
	ObjectiveValueReliability = 50
)

// Core Engine & Compilation.
const (
	ObjectiveLabelWorkflow = "workflow"
	ObjectiveLabelEngine   = "engine"
	ObjectiveValueWorkflow = 45
	ObjectiveValueEngine   = 40
)

// Integration Points.
const (
	ObjectiveLabelMCP     = "mcp"
	ObjectiveLabelActions = "actions"
	ObjectiveLabelCLI     = "cli"
	ObjectiveValueMCP     = 45
	ObjectiveValueActions = 40
	ObjectiveValueCLI     = 40
)

// Bug Fixes.
const (
	ObjectiveLabelBug = "bug"
	ObjectiveValueBug = 60
)

// Security.
const (
	ObjectiveLabelSecurityFix = "security-fix"
	ObjectiveValueSecurityFix = 70
)

// Copilot-Specific Optimizations.
const (
	ObjectiveLabelCopilotOpt = "copilot-opt"
	ObjectiveValueCopilotOpt = 75
)

// High Priority Work.
const (
	ObjectiveLabelHighPriority = "high-priority"
	ObjectiveLabelP1           = "p1"
	ObjectiveValueHighPriority = 35
	ObjectiveValueP1           = 35
)

// Code Quality.
const (
	ObjectiveLabelLintMonster = "lint-monster"
	ObjectiveValueLintMonster = 25
	ObjectiveLabelPerformance = "performance"
	ObjectiveValuePerformance = 30
)

// Medium Priority Work.
const (
	ObjectiveLabelMediumPriority = "medium-priority"
	ObjectiveLabelP2             = "p2"
	ObjectiveValueMediumPriority = 20
	ObjectiveValueP2             = 20
)

// Dependency Management.
const (
	ObjectiveLabelDependencies = "dependencies"
	ObjectiveValueDependencies = 10
)

// Low Priority Work.
const (
	ObjectiveLabelLowPriority = "low-priority"
	ObjectiveLabelP3          = "p3"
	ObjectiveValueLowPriority = 10
	ObjectiveValueP3          = 10
)

// Enhancement & Documentation.
const (
	ObjectiveLabelEnhancement   = "enhancement"
	ObjectiveValueEnhancement   = 15
	ObjectiveLabelDocumentation = "documentation"
	ObjectiveValueDocumentation = 5
)

// Workflow/Automation Labels (no objective value).
const (
	ObjectiveLabelAIGenerated  = "ai-generated"
	ObjectiveValueAIGenerated  = 0
	ObjectiveLabelAIInspected  = "ai-inspected"
	ObjectiveValueAIInspected  = 0
	ObjectiveLabelSmokeCopilot = "smoke-copilot"
	ObjectiveValueSmokeCopilot = 0
)

// Question & Community Labels (no objective value).
const (
	ObjectiveLabelQuestion       = "question"
	ObjectiveValueQuestion       = 0
	ObjectiveLabelGoodFirstIssue = "good first issue"
	ObjectiveValueGoodFirstIssue = 0
)
