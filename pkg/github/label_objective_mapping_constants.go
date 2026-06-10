package github

// Objective label/value constants used by the objective-mapping feature.
//
// Note: keep these values in sync with DefaultObjectiveMapping (pkg/github/label_objective_mapping.go)
// and/or the repository-level .github/objective-mapping.json to avoid divergent scoring semantics.
// These mappings reflect the actual work domains and priorities:
//   - Safety/Reliability: Safe outputs, testing, reliability = critical
//   - Core engine: Compilation, parsing, workflow execution = critical
//   - Integration: MCP tools, GitHub Actions, CLI = important
//   - Quality: Bug fixes, performance, linting = important
//   - Enhancement: New features, documentation = valuable but lower impact
//
// To customize these mappings:
//  1. Create .github/objective-mapping.json in your repository root
//  2. Set OBJECTIVE_MAPPING_JSON environment variable (JSON string or file path)
//  3. See specs/objective-mapping-portfolio-reporting.md for configuration details
//
// Critical Priority Labels
const (
	ObjectiveLabelCritical = "critical"
	ObjectiveLabelP0       = "p0"
	ObjectiveValueCritical = 100
	ObjectiveValueP0       = 100
)

// Safety-Critical Work (safe outputs, test failures)
const (
	ObjectiveLabelTesting     = "testing"
	ObjectiveLabelReliability = "reliability"
	ObjectiveValueTesting     = 50
	ObjectiveValueReliability = 50
)

// Core Engine & Compilation
const (
	ObjectiveLabelWorkflow = "workflow"
	ObjectiveLabelEngine   = "engine"
	ObjectiveValueWorkflow = 45
	ObjectiveValueEngine   = 40
)

// Integration Points
const (
	ObjectiveLabelMCP     = "mcp"
	ObjectiveLabelActions = "actions"
	ObjectiveLabelCLI     = "cli"
	ObjectiveValueMCP     = 45
	ObjectiveValueActions = 40
	ObjectiveValueCLI     = 40
)

// Bug Fixes (especially core path)
const (
	ObjectiveLabelBug = "bug"
	ObjectiveValueBug = 60
)

// Security
const (
	ObjectiveLabelSecurityFix = "security-fix"
	ObjectiveValueSecurityFix = 70
)

// Copilot-Specific Optimizations
const (
	ObjectiveLabelCopilotOpt = "copilot-opt"
	ObjectiveValueCopilotOpt = 75
)

// High Priority Work
const (
	ObjectiveLabelHighPriority = "high-priority"
	ObjectiveLabelP1           = "p1"
	ObjectiveValueHighPriority = 35
	ObjectiveValueP1           = 35
)

// Code Quality
const (
	ObjectiveLabelLintMonster = "lint-monster"
	ObjectiveValueLintMonster = 25
	ObjectiveLabelPerformance = "performance"
	ObjectiveValuePerformance = 30
)

// Medium Priority Work
const (
	ObjectiveLabelMediumPriority = "medium-priority"
	ObjectiveLabelP2             = "p2"
	ObjectiveValueMediumPriority = 20
	ObjectiveValueP2             = 20
)

// Dependency Management
const (
	ObjectiveLabelDependencies = "dependencies"
	ObjectiveValueDependencies = 10
)

// Low Priority Work
const (
	ObjectiveLabelLowPriority = "low-priority"
	ObjectiveLabelP3          = "p3"
	ObjectiveValueLowPriority = 10
	ObjectiveValueP3          = 10
)

// Enhancement & Documentation
const (
	ObjectiveLabelEnhancement   = "enhancement"
	ObjectiveValueEnhancement   = 15
	ObjectiveLabelDocumentation = "documentation"
	ObjectiveValueDocumentation = 5
)

// Workflow/Automation Labels (no objective value)
const (
	ObjectiveLabelAIGenerated  = "ai-generated"
	ObjectiveValueAIGenerated  = 0
	ObjectiveLabelAIInspected  = "ai-inspected"
	ObjectiveValueAIInspected  = 0
	ObjectiveLabelSmokeCopilot = "smoke-copilot"
	ObjectiveValueSmokeCopilot = 0
)

// Question & Community Labels (no objective value)
const (
	ObjectiveLabelQuestion       = "question"
	ObjectiveValueQuestion       = 0
	ObjectiveLabelGoodFirstIssue = "good first issue"
	ObjectiveValueGoodFirstIssue = 0
)

// Combination logic options
const (
	MultiLabelLogicMax   = "max"   // Use highest value (default)
	MultiLabelLogicSum   = "sum"   // Add all values
	MultiLabelLogicFirst = "first" // Use first in priority order
)
