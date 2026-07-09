package workflow

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/setutil"
)

var frontmatterLog = logger.New("workflow:frontmatter_extraction")

// indentYAMLLines adds indentation to all lines of a multi-line YAML string except the first
func (c *Compiler) indentYAMLLines(yamlContent, indent string) string {
	if yamlContent == "" {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	if len(lines) <= 1 {
		return yamlContent
	}

	// First line doesn't get additional indentation
	var result strings.Builder
	result.WriteString(lines[0])
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			result.WriteString("\n" + indent + lines[i])
		} else {
			// Emit a bare newline for blank/whitespace-only lines so we don't
			// carry the surrounding indentation as trailing whitespace, which
			// yamllint flags as trailing-spaces.
			result.WriteString("\n")
		}
	}

	return result.String()
}

// marshalTopLevelYAMLValue marshals a top-level YAML section value with appropriate options.
func marshalTopLevelYAMLValue(key string, value any) ([]byte, error) {
	marshalOptions := DefaultMarshalOptions
	if key == "on" {
		// Indent sequence items under their parent key so that yamllint's default
		// indentation rule (indent-sequences: true) is satisfied.
		marshalOptions = append(append([]yaml.EncodeOption{}, DefaultMarshalOptions...), yaml.IndentSequence(true))
	}
	if valueMap, ok := value.(map[string]any); ok {
		orderedValue := OrderMapFields(valueMap, []string{})
		wrappedData := yaml.MapSlice{{Key: key, Value: orderedValue}}
		return yaml.MarshalWithOptions(wrappedData, marshalOptions...)
	}
	return yaml.MarshalWithOptions(map[string]any{key: value}, marshalOptions...)
}

// extractTopLevelYAMLSection extracts a top-level YAML section from frontmatter
func (c *Compiler) extractTopLevelYAMLSection(frontmatter map[string]any, key string) string {
	value, exists := frontmatter[key]
	if !exists {
		return ""
	}
	frontmatterLog.Printf("Extracting YAML section: %s", key)

	yamlBytes, err := marshalTopLevelYAMLValue(key, value)
	if err != nil {
		return ""
	}

	yamlStr := strings.TrimSuffix(string(yamlBytes), "\n")
	// Post-process YAML to ensure cron expressions are quoted.
	yamlStr = parser.QuoteCronExpressions(yamlStr)
	// For top-level env values, quote plain scalars containing ": ".
	if key == "env" {
		yamlStr = quoteEnvValuesContainingColonSpace(yamlStr)
	}
	// Clean up null values — replace `: null` with `:` for cleaner output.
	yamlStr = CleanYAMLNullValues(yamlStr)
	// Clean up quoted keys — replace "key": with key: at the start of a line.
	if key != "on" {
		yamlStr = UnquoteYAMLKey(yamlStr, key)
	}
	if key == "on" {
		yamlStr = c.commentOutProcessedFieldsInOnSection(yamlStr, frontmatter)
		yamlStr = c.addZizmorIgnoreForWorkflowRun(yamlStr)
		yamlStr = c.addFriendlyScheduleComments(yamlStr, frontmatter)
	}
	return yamlStr
}

// detectNativeLabelFilterSections scans frontmatter for event sections that use native label
// filtering (marked with __gh_aw_native_label_filter__: true). Those sections must NOT have their
// "names" fields commented out in the compiled YAML output.
func detectNativeLabelFilterSections(frontmatter map[string]any) map[string]struct{} {
	sections := make(map[string]struct{})
	onValue, exists := frontmatter["on"]
	if !exists {
		return sections
	}
	onMap, ok := onValue.(map[string]any)
	if !ok {
		return sections
	}
	for _, sectionKey := range []string{"issues", "pull_request", "discussion", "issue_comment"} {
		sectionValue, hasSec := onMap[sectionKey]
		if !hasSec {
			continue
		}
		sectionMap, ok := sectionValue.(map[string]any)
		if !ok {
			continue
		}
		marker, hasMarker := sectionMap["__gh_aw_native_label_filter__"]
		if !hasMarker {
			continue
		}
		if useNative, ok := marker.(bool); ok && useNative {
			sections[sectionKey] = struct{}{}
			frontmatterLog.Printf("Section %s uses native label filtering", sectionKey)
		}
	}
	return sections
}

// onSectionFilter holds the state for the line-by-line comment-out pass over the "on" section.
type onSectionFilter struct {
	// Current event section tracking (pull_request / issues / discussion / issue_comment)
	inPullRequest        bool
	inIssues             bool
	inDiscussion         bool
	inIssueComment       bool
	currentSection       string
	currentSectionIndent int

	// Other event sections
	inDeploymentStatus     bool
	deploymentStatusIndent int
	inWorkflowRun          bool
	workflowRunIndent      int

	// workflow_run sub-state
	inWorkflowRunConclusionArray bool

	// pull_request sub-arrays
	inForksArray bool

	// Top-level on: extension arrays (not inside event sections)
	inSkipRolesArray bool
	inSkipBotsArray  bool
	inRolesArray     bool
	inBotsArray      bool
	inLabelsArray    bool
	inNeedsArray     bool

	// Top-level on: extension block fields
	inSkipIfMatch            bool
	inSkipIfNoMatch          bool
	inSkipIfCheckFailing     bool
	inSkipAuthorAssociations bool
	inGitHubApp              bool
	inOnSteps                bool
	inOnPermissions          bool

	// Sections that use native label filtering (names field must not be commented out)
	nativeLabelFilterSections map[string]struct{}

	// Accumulated output lines; kept for the names-array lookback.
	result []string
}

func newOnSectionFilter(nativeLabelFilterSections map[string]struct{}) *onSectionFilter {
	return &onSectionFilter{
		currentSectionIndent:      -1,
		deploymentStatusIndent:    -1,
		workflowRunIndent:         -1,
		nativeLabelFilterSections: nativeLabelFilterSections,
	}
}

// activateSection resets all event-section flags and then activates the selected section.
// It also clears every top-level on: extension-array tracker before entering the new section.
// This reset is required because each activateSection call ends with an early return in
// processLine (skipping the indent-based deactivation logic). Without the explicit reset, a
// stale flag from a preceding block would cause sibling items to be incorrectly commented out.
func (s *onSectionFilter) activateSection(section string, indent int) {
	s.inSkipRolesArray = false
	s.inSkipBotsArray = false
	s.inRolesArray = false
	s.inBotsArray = false
	s.inLabelsArray = false
	s.inNeedsArray = false
	s.inSkipIfMatch = false
	s.inSkipIfNoMatch = false
	s.inSkipIfCheckFailing = false
	s.inSkipAuthorAssociations = false

	s.inPullRequest = section == "pull_request"
	s.inIssues = section == "issues"
	s.inDiscussion = section == "discussion"
	s.inIssueComment = section == "issue_comment"
	s.inDeploymentStatus = section == "deployment_status"
	s.inWorkflowRun = section == "workflow_run"
	s.inWorkflowRunConclusionArray = false
	s.inForksArray = false

	switch section {
	case "pull_request", "issues", "discussion", "issue_comment":
		s.currentSection = section
		s.currentSectionIndent = indent
	default:
		s.currentSection = ""
		s.currentSectionIndent = -1
	}

	if section == "deployment_status" {
		s.deploymentStatusIndent = indent
	} else {
		s.deploymentStatusIndent = -1
	}
	if section == "workflow_run" {
		s.workflowRunIndent = indent
	} else {
		s.workflowRunIndent = -1
	}
}

// checkSectionEnter detects pull_request/issues/discussion/issue_comment/deployment_status/workflow_run
// lines and activates the appropriate section state. Returns true if the line was handled (caller
// should skip remaining processing for this line).
func (s *onSectionFilter) checkSectionEnter(line, trimmedLine string, lineIndent int) bool {
	if s.inOnPermissions || s.inOnSteps || s.inSkipAuthorAssociations {
		return false
	}
	if lineIndent != 2 && lineIndent != 4 {
		return false
	}
	var section string
	switch trimmedLine {
	case "pull_request:":
		section = "pull_request"
	case "issues:":
		section = "issues"
	case "discussion:":
		section = "discussion"
	case "issue_comment:":
		section = "issue_comment"
	case "deployment_status:":
		section = "deployment_status"
	case "workflow_run:":
		section = "workflow_run"
	}
	if section == "" {
		return false
	}
	s.activateSection(section, lineIndent)
	s.result = append(s.result, line)
	return true
}

// checkSectionExit updates section-exit state when the current line is at or above the section's
// indentation level (indicating we've left the section).
func (s *onSectionFilter) checkSectionExit(trimmedLine string, lineIndent int) {
	isRealLine := trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#")

	if (s.inPullRequest || s.inIssues || s.inDiscussion || s.inIssueComment) && isRealLine &&
		s.currentSectionIndent >= 0 && lineIndent <= s.currentSectionIndent {
		s.inPullRequest = false
		s.inIssues = false
		s.inDiscussion = false
		s.inIssueComment = false
		s.inForksArray = false
		s.currentSection = ""
		s.currentSectionIndent = -1
	}

	if s.inDeploymentStatus && isRealLine &&
		s.deploymentStatusIndent >= 0 && lineIndent <= s.deploymentStatusIndent {
		s.inDeploymentStatus = false
		s.deploymentStatusIndent = -1
	}

	if s.inWorkflowRun && isRealLine &&
		s.workflowRunIndent >= 0 && lineIndent <= s.workflowRunIndent {
		s.inWorkflowRun = false
		s.inWorkflowRunConclusionArray = false
		s.workflowRunIndent = -1
	}
}

// updateEnterSimpleState detects entry into simple array/block state fields.
func (s *onSectionFilter) updateEnterSimpleState(trimmedLine string, lineIndent int) {
	inEventSection := s.inPullRequest || s.inIssues || s.inDiscussion || s.inIssueComment

	if s.inPullRequest && strings.HasPrefix(trimmedLine, "forks:") {
		s.inForksArray = true
	}
	if !inEventSection && strings.HasPrefix(trimmedLine, "skip-roles:") {
		s.inSkipRolesArray = true
	}
	if !inEventSection && strings.HasPrefix(trimmedLine, "skip-bots:") {
		s.inSkipBotsArray = true
	}
	if !inEventSection && strings.HasPrefix(trimmedLine, "roles:") {
		s.inRolesArray = true
	}
	if !inEventSection && strings.HasPrefix(trimmedLine, "bots:") {
		s.inBotsArray = true
	}
	if !inEventSection && !s.inOnSteps && !s.inOnPermissions && lineIndent == 2 && trimmedLine == "labels:" {
		s.inLabelsArray = true
	}
	if !inEventSection && !s.inOnSteps && !s.inOnPermissions && lineIndent == 2 && strings.HasPrefix(trimmedLine, "needs:") {
		s.inNeedsArray = true
	}
	if !inEventSection && strings.HasPrefix(trimmedLine, "steps:") {
		s.inOnSteps = true
	}
}

// updateEnterComplexState detects entry into block extension fields that span multiple lines.
func (s *onSectionFilter) updateEnterComplexState(trimmedLine string) {
	inEventSection := s.inPullRequest || s.inIssues || s.inDiscussion || s.inIssueComment
	if inEventSection {
		return
	}
	if !s.inOnPermissions && strings.HasPrefix(trimmedLine, "permissions:") {
		s.inOnPermissions = true
	}
	if !s.inSkipIfMatch && (trimmedLine == "skip-if-match:" ||
		(strings.HasPrefix(trimmedLine, "# skip-if-match:") && strings.Contains(trimmedLine, "pre-activation job"))) {
		s.inSkipIfMatch = true
	}
	if !s.inSkipIfNoMatch && (trimmedLine == "skip-if-no-match:" ||
		(strings.HasPrefix(trimmedLine, "# skip-if-no-match:") && strings.Contains(trimmedLine, "pre-activation job"))) {
		s.inSkipIfNoMatch = true
	}
	if !s.inSkipIfCheckFailing && (trimmedLine == "skip-if-check-failing:" ||
		(strings.HasPrefix(trimmedLine, "# skip-if-check-failing:") && strings.Contains(trimmedLine, "pre-activation job"))) {
		s.inSkipIfCheckFailing = true
	}
	if !s.inSkipAuthorAssociations && trimmedLine == "skip-author-associations:" {
		s.inSkipAuthorAssociations = true
	}
	if !s.inGitHubApp && (trimmedLine == "github-app:" ||
		(strings.HasPrefix(trimmedLine, "# github-app:") && strings.Contains(trimmedLine, "pre-activation job"))) {
		s.inGitHubApp = true
	}
}

// updateExitComplexState detects when we leave extension block fields.
func (s *onSectionFilter) updateExitComplexState(trimmedLine string, lineIndent int) {
	isReal := trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#")

	if s.inSkipIfMatch && isReal && !strings.HasPrefix(trimmedLine, "skip-if-match:") &&
		!strings.HasPrefix(trimmedLine, "# skip-if-match:") && lineIndent == 2 {
		s.inSkipIfMatch = false
	}
	if s.inSkipIfNoMatch && isReal && !strings.HasPrefix(trimmedLine, "skip-if-no-match:") &&
		!strings.HasPrefix(trimmedLine, "# skip-if-no-match:") && lineIndent == 2 {
		s.inSkipIfNoMatch = false
	}
	if s.inSkipIfCheckFailing && isReal && !strings.HasPrefix(trimmedLine, "skip-if-check-failing:") &&
		!strings.HasPrefix(trimmedLine, "# skip-if-check-failing:") && lineIndent == 2 {
		s.inSkipIfCheckFailing = false
	}
	if s.inSkipAuthorAssociations && isReal && !strings.HasPrefix(trimmedLine, "skip-author-associations:") &&
		!strings.HasPrefix(trimmedLine, "# skip-author-associations:") && lineIndent == 2 {
		s.inSkipAuthorAssociations = false
	}
	if s.inGitHubApp && isReal && !strings.HasPrefix(trimmedLine, "github-app:") &&
		!strings.HasPrefix(trimmedLine, "# github-app:") && lineIndent == 2 {
		s.inGitHubApp = false
	}
}

// updateExitSimpleState detects when we leave simple array/block state fields.
func (s *onSectionFilter) updateExitSimpleState(trimmedLine string, lineIndent int) {
	isReal := trimmedLine != ""

	if s.inForksArray && s.inPullRequest && isReal && lineIndent == 4 &&
		!strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "forks:") {
		s.inForksArray = false
	}
	if s.inSkipRolesArray && isReal && lineIndent == 2 &&
		!strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "skip-roles:") && !strings.HasPrefix(trimmedLine, "#") {
		s.inSkipRolesArray = false
	}
	if s.inSkipBotsArray && isReal && lineIndent == 2 &&
		!strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "skip-bots:") && !strings.HasPrefix(trimmedLine, "#") {
		s.inSkipBotsArray = false
	}
	if s.inRolesArray && isReal && lineIndent == 2 &&
		!strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "roles:") && !strings.HasPrefix(trimmedLine, "#") {
		s.inRolesArray = false
	}
	if s.inBotsArray && isReal && lineIndent == 2 &&
		!strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "bots:") && !strings.HasPrefix(trimmedLine, "#") {
		s.inBotsArray = false
	}
	if s.inLabelsArray && isReal && lineIndent == 2 &&
		!strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "labels:") && !strings.HasPrefix(trimmedLine, "#") {
		s.inLabelsArray = false
	}
	if s.inNeedsArray && isReal && lineIndent == 2 &&
		!strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "needs:") && !strings.HasPrefix(trimmedLine, "#") {
		s.inNeedsArray = false
	}
	if s.inOnSteps && isReal && lineIndent == 2 &&
		!strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "steps:") && !strings.HasPrefix(trimmedLine, "#") {
		s.inOnSteps = false
	}
	if s.inOnPermissions && isReal && !strings.HasPrefix(trimmedLine, "permissions:") &&
		!strings.HasPrefix(trimmedLine, "# permissions:") && lineIndent == 2 && !strings.HasPrefix(trimmedLine, "#") {
		s.inOnPermissions = false
	}
}

// computeTopLevelSkipAndRolesComment returns shouldComment and commentReason for skip/role/bot
// top-level extension fields (when NOT inside an event section).
func (s *onSectionFilter) computeTopLevelSkipAndRolesComment(trimmedLine string) (bool, string) {
	switch {
	case strings.HasPrefix(trimmedLine, "manual-approval:"):
		return true, " # Manual approval processed as environment field in activation job"
	case strings.HasPrefix(trimmedLine, "stop-after:"):
		return true, " # Stop-after processed as stop-time check in pre-activation job"
	case strings.HasPrefix(trimmedLine, "skip-if-match:"):
		return true, " # Skip-if-match processed as search check in pre-activation job"
	case s.inSkipIfMatch && (strings.HasPrefix(trimmedLine, "query:") || strings.HasPrefix(trimmedLine, "max:") || strings.HasPrefix(trimmedLine, "scope:")):
		return true, ""
	case strings.HasPrefix(trimmedLine, "skip-if-no-match:"):
		return true, " # Skip-if-no-match processed as search check in pre-activation job"
	case s.inSkipIfNoMatch && (strings.HasPrefix(trimmedLine, "query:") || strings.HasPrefix(trimmedLine, "min:") || strings.HasPrefix(trimmedLine, "scope:")):
		return true, ""
	case strings.HasPrefix(trimmedLine, "skip-if-check-failing:"):
		return true, " # Skip-if-check-failing processed as check status gate in pre-activation job"
	case s.inSkipIfCheckFailing && (strings.HasPrefix(trimmedLine, "include:") || strings.HasPrefix(trimmedLine, "exclude:") || strings.HasPrefix(trimmedLine, "branch:") || strings.HasPrefix(trimmedLine, "allow-pending:") || strings.HasPrefix(trimmedLine, "-")):
		return true, ""
	case strings.HasPrefix(trimmedLine, "skip-author-associations:"):
		return true, " # Skip-author-associations compiled into pre-activation job if condition"
	case strings.HasPrefix(trimmedLine, "skip-roles:"):
		return true, " # Skip-roles processed as role check in pre-activation job"
	case s.inSkipRolesArray && strings.HasPrefix(trimmedLine, "-"):
		return true, " # Skip-roles processed as role check in pre-activation job"
	case strings.HasPrefix(trimmedLine, "skip-bots:"):
		return true, " # Skip-bots processed as bot check in pre-activation job"
	case s.inSkipBotsArray && strings.HasPrefix(trimmedLine, "-"):
		return true, " # Skip-bots processed as bot check in pre-activation job"
	case strings.HasPrefix(trimmedLine, "roles:"):
		return true, " # Roles processed as role check in pre-activation job"
	case s.inRolesArray && strings.HasPrefix(trimmedLine, "-"):
		return true, " # Roles processed as role check in pre-activation job"
	case strings.HasPrefix(trimmedLine, "bots:"):
		return true, " # Bots processed as bot check in pre-activation job"
	case s.inBotsArray && strings.HasPrefix(trimmedLine, "-"):
		return true, " # Bots processed as bot check in pre-activation job"
	}
	return false, ""
}

// computeTopLevelLabelsAndStepsComment returns shouldComment and commentReason for labels/needs/
// steps/permissions/reaction/github-app/stale-check top-level fields.
func (s *onSectionFilter) computeTopLevelLabelsAndStepsComment(trimmedLine string, lineIndent int) (bool, string) {
	switch {
	case s.inSkipAuthorAssociations && lineIndent > 2:
		return true, ""
	case !s.inOnSteps && !s.inOnPermissions && lineIndent == 2 && strings.HasPrefix(trimmedLine, "labels:"):
		return true, " # Label filtering applied via job conditions"
	case s.inLabelsArray && strings.HasPrefix(trimmedLine, "-"):
		return true, " # Label filtering applied via job conditions"
	case !s.inOnSteps && !s.inOnPermissions && lineIndent == 2 && strings.HasPrefix(trimmedLine, "needs:"):
		return true, " # Needs processed as dependency in pre-activation job"
	case s.inNeedsArray && strings.HasPrefix(trimmedLine, "-"):
		return true, " # Needs processed as dependency in pre-activation job"
	case strings.HasPrefix(trimmedLine, "restore-memory:"):
		return true, " # Restore-memory enables pre-activation memory restore"
	case strings.HasPrefix(trimmedLine, "steps:"):
		return true, " # Steps injected into pre-activation job"
	case s.inOnSteps:
		return true, ""
	case strings.HasPrefix(trimmedLine, "permissions:"):
		return true, " # Permissions applied to pre-activation job"
	case s.inOnPermissions:
		return true, ""
	case strings.HasPrefix(trimmedLine, "reaction:"):
		return true, " # Reaction processed as activation job step"
	case strings.HasPrefix(trimmedLine, "github-token:"):
		return true, " # GitHub token used for reactions and status comments in activation"
	case strings.HasPrefix(trimmedLine, "github-app:"):
		return true, " # GitHub App used to mint token for reactions and status comments in activation"
	case s.inGitHubApp && isGitHubAppNestedField(trimmedLine):
		return true, ""
	case strings.HasPrefix(trimmedLine, "stale-check:"):
		return true, " # Stale-check processed as frontmatter hash check step in activation job"
	}
	return false, ""
}

// computeEventSectionComment returns shouldComment and commentReason for lines inside event
// sections (pull_request, issues, discussion, issue_comment, deployment_status, workflow_run).
func (s *onSectionFilter) computeEventSectionComment(line, trimmedLine string) (bool, string) {
	inEventSection := s.inPullRequest || s.inIssues || s.inDiscussion || s.inIssueComment
	switch {
	case s.inPullRequest && strings.Contains(trimmedLine, "draft:"):
		return true, " # Draft filtering applied via job conditions"
	case s.inPullRequest && strings.HasPrefix(trimmedLine, "forks:"):
		return true, " # Fork filtering applied via job conditions"
	case s.inForksArray && strings.HasPrefix(trimmedLine, "-"):
		return true, " # Fork filtering applied via job conditions"
	case s.inDeploymentStatus && strings.HasPrefix(trimmedLine, "state:"):
		return true, " # State filtering compiled into if condition"
	case s.inDeploymentStatus && strings.HasPrefix(trimmedLine, "-"):
		return true, " # State filtering compiled into if condition"
	case s.inWorkflowRun && strings.HasPrefix(trimmedLine, "conclusion:"):
		s.inWorkflowRunConclusionArray = true
		return true, " # Conclusion filtering compiled into if condition"
	case s.inWorkflowRunConclusionArray && strings.HasPrefix(trimmedLine, "-"):
		return true, " # Conclusion filtering compiled into if condition"
	case s.inWorkflowRun && !strings.HasPrefix(trimmedLine, "-") && strings.Contains(trimmedLine, ":"):
		s.inWorkflowRunConclusionArray = false
		return false, ""
	case inEventSection && strings.HasPrefix(trimmedLine, "lock-for-agent:"):
		return true, " # Lock-for-agent processed as issue locking in activation job"
	case inEventSection && strings.HasPrefix(trimmedLine, "names:"):
		if !setutil.Contains(s.nativeLabelFilterSections, s.currentSection) {
			return true, " # Label filtering applied via job conditions"
		}
		return false, ""
	case inEventSection && line != "":
		return s.computeNamesArrayComment(trimmedLine)
	}
	return false, ""
}

// computeNamesArrayComment checks whether the current array-item line falls inside a commented
// names: block (via backward lookback) and should therefore also be commented out.
func (s *onSectionFilter) computeNamesArrayComment(trimmedLine string) (bool, string) {
	if setutil.Contains(s.nativeLabelFilterSections, s.currentSection) || !strings.HasPrefix(trimmedLine, "-") {
		return false, ""
	}
	for i := range slices.Backward(s.result) {
		prevLine := s.result[i]
		prevTrimmed := strings.TrimSpace(prevLine)
		if prevTrimmed == "" {
			continue
		}
		if strings.Contains(prevTrimmed, "names:") && strings.Contains(prevTrimmed, "# Label filtering") {
			return true, " # Label filtering applied via job conditions"
		}
		if !strings.HasPrefix(prevTrimmed, "#") || !strings.Contains(prevTrimmed, "Label filtering") {
			break
		}
		if strings.HasPrefix(prevTrimmed, "# -") && strings.Contains(prevTrimmed, "Label filtering") {
			return true, " # Label filtering applied via job conditions"
		}
		break
	}
	return false, ""
}

// processLine processes one YAML line through the comment-out filter and appends the result to
// s.result.
func (s *onSectionFilter) processLine(line string) {
	trimmedLine := strings.TrimSpace(line)
	lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))

	// Check for event section entry (returns immediately if section transition).
	if s.checkSectionEnter(line, trimmedLine, lineIndent) {
		return
	}

	// Update section exit state.
	s.checkSectionExit(trimmedLine, lineIndent)

	// Skip marker lines (not included in output).
	inEventSection := s.inPullRequest || s.inIssues || s.inDiscussion || s.inIssueComment
	if inEventSection && strings.Contains(trimmedLine, "__gh_aw_native_label_filter__:") {
		return
	}

	// Update enter/exit state for extension fields (order must match original).
	s.updateEnterSimpleState(trimmedLine, lineIndent)
	s.updateEnterComplexState(trimmedLine)
	s.updateExitComplexState(trimmedLine, lineIndent)
	s.updateExitSimpleState(trimmedLine, lineIndent)

	// Compute shouldComment.
	var shouldComment bool
	var commentReason string

	if !inEventSection {
		if ok, reason := s.computeTopLevelSkipAndRolesComment(trimmedLine); ok {
			shouldComment, commentReason = ok, reason
		} else {
			shouldComment, commentReason = s.computeTopLevelLabelsAndStepsComment(trimmedLine, lineIndent)
		}
	}
	if !shouldComment {
		shouldComment, commentReason = s.computeEventSectionComment(line, trimmedLine)
	}

	if shouldComment {
		indentation := ""
		trimmed := strings.TrimLeft(line, " \t")
		if len(line) > len(trimmed) {
			indentation = line[:len(line)-len(trimmed)]
		}
		commentedLine := indentation + "# " + trimmed + commentReason
		if trimmed == "" {
			commentedLine = strings.TrimRight(commentedLine, " \t")
		}
		s.result = append(s.result, commentedLine)
	} else {
		s.result = append(s.result, line)
	}
}

// commentOutProcessedFieldsInOnSection comments out fields in the on: section that are processed
// separately (draft, fork, forks, names, labels, manual-approval, stop-after, skip-if-match,
// skip-if-no-match, skip-if-check-failing, skip-roles, reaction, lock-for-agent, steps,
// permissions, needs, restore-memory, and stale-check). Sections with
// __gh_aw_native_label_filter__ are not modified.
func (c *Compiler) commentOutProcessedFieldsInOnSection(yamlStr string, frontmatter map[string]any) string {
	frontmatterLog.Print("Processing 'on' section to comment out processed fields")
	nativeLabelFilterSections := detectNativeLabelFilterSections(frontmatter)
	filter := newOnSectionFilter(nativeLabelFilterSections)
	for line := range strings.SplitSeq(yamlStr, "\n") {
		filter.processLine(line)
	}
	return strings.Join(filter.result, "\n")
}

// addZizmorIgnoreForWorkflowRun adds a zizmor ignore comment for workflow_run triggers
// The comment is added after the workflow_run: line to suppress dangerous-triggers warnings
// since the compiler adds proper role and fork validation to secure these triggers
func (c *Compiler) addZizmorIgnoreForWorkflowRun(yamlStr string) string {
	// Check if the YAML contains workflow_run trigger
	if !strings.Contains(yamlStr, "workflow_run:") {
		return yamlStr
	}
	frontmatterLog.Print("Adding zizmor ignore annotation for workflow_run trigger")

	lines := strings.Split(yamlStr, "\n")
	var result []string
	annotationAdded := false // Track if we've already added the annotation

	for _, line := range lines {
		result = append(result, line)

		// Skip if we've already added the annotation (prevents duplicates)
		if annotationAdded {
			continue
		}

		// Check if this is a non-comment workflow_run: key at the correct YAML level
		trimmedLine := strings.TrimSpace(line)

		// Skip if the line is a comment
		if strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Match lines that are only 'workflow_run:' (possibly with trailing whitespace or a comment)
		// e.g., 'workflow_run:', 'workflow_run: # comment', '  workflow_run:'
		// But not 'someworkflow_run:', 'workflow_run: value', etc.
		if idx := strings.Index(trimmedLine, "workflow_run:"); idx == 0 {
			after := strings.TrimSpace(trimmedLine[len("workflow_run:"):])
			// Only allow if nothing or only a comment follows
			if after == "" || strings.HasPrefix(after, "#") {
				// Get the indentation of the workflow_run line
				indentation := ""
				if len(line) > len(trimmedLine) {
					indentation = line[:len(line)-len(trimmedLine)]
				}

				// Add zizmor ignore comment with proper indentation
				// The comment explains that the trigger is secured with role and fork validation
				comment := indentation + "  # zizmor: ignore[dangerous-triggers] - workflow_run trigger is secured with role and fork validation"
				result = append(result, comment)
				annotationAdded = true
			}
		}
	}

	return strings.Join(result, "\n")
}

// extractPermissions extracts permissions from frontmatter using the permission parser
func (c *Compiler) extractPermissions(frontmatter map[string]any) string {
	permissionsValue, exists := frontmatter["permissions"]
	if !exists {
		frontmatterLog.Print("No permissions field found in frontmatter")
		return ""
	}

	// Check if this is an "all: read" case by using the parser
	parser := NewPermissionsParserFromValue(permissionsValue)

	// If it's "all: read", use the parser to expand it
	if parser.hasAll && parser.allLevel == "read" {
		frontmatterLog.Print("Expanding 'all: read' permissions to individual scopes")
		permissions := parser.ToPermissions()
		yaml := permissions.RenderToYAML()

		// Adjust indentation from 6 spaces to 2 spaces for workflow-level permissions
		// RenderToYAML uses 6 spaces for job-level rendering
		lines := strings.Split(yaml, "\n")
		for i := 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "      ") {
				lines[i] = "  " + lines[i][6:]
			}
		}
		return strings.Join(lines, "\n")
	}

	// For all other cases, use standard extraction
	return c.extractTopLevelYAMLSection(frontmatter, "permissions")
}

// extractIfCondition extracts the if condition from frontmatter, returning just the expression
// without the "if: " prefix. Also merges any condition derived from on.deployment_status.state
// and on.workflow_run.conclusion.
func (c *Compiler) extractIfCondition(frontmatter map[string]any) (string, error) {
	var ifExpr string
	if value, exists := frontmatter["if"]; exists {
		if strValue, ok := value.(string); ok {
			// Strip "if: " prefix and ${{ }} wrapper to get a bare expression for safe merging
			ifExpr = stripExpressionWrapper(c.extractExpressionFromIfString(strValue))
			frontmatterLog.Printf("Extracted if condition from frontmatter: %s", ifExpr)
		}
	}

	// Merge any condition generated from on.deployment_status.state
	stateCondition := extractDeploymentStatusStateCondition(frontmatter)
	if stateCondition != "" {
		frontmatterLog.Printf("Merging deployment_status state condition: %s", stateCondition)
		if ifExpr != "" {
			ifExpr = "(" + ifExpr + ") && (" + stateCondition + ")"
		} else {
			ifExpr = stateCondition
		}
	}

	// Merge any condition generated from on.workflow_run.conclusion
	conclusionCondition, err := extractWorkflowRunConclusionCondition(frontmatter)
	if err != nil {
		return "", err
	}
	if conclusionCondition != "" {
		frontmatterLog.Printf("Merging workflow_run conclusion condition: %s", conclusionCondition)
		if ifExpr != "" {
			ifExpr = "(" + ifExpr + ") && (" + conclusionCondition + ")"
		} else {
			ifExpr = conclusionCondition
		}
	}

	return ifExpr, nil
}

// extractOnTriggerValue returns the raw value for on.<trigger> when the frontmatter
// contains an "on" map with that trigger configured.
func extractOnTriggerValue(frontmatter map[string]any, trigger string) (any, bool) {
	onMap, ok := frontmatter["on"].(map[string]any)
	if !ok {
		return nil, false
	}
	value, ok := onMap[trigger]
	return value, ok
}

// extractOnTriggerMap returns the on.<trigger> value as a map when the configured
// trigger uses object syntax.
func extractOnTriggerMap(frontmatter map[string]any, trigger string) (map[string]any, bool) {
	value, ok := extractOnTriggerValue(frontmatter, trigger)
	if !ok {
		return nil, false
	}
	triggerMap, ok := value.(map[string]any)
	return triggerMap, ok
}

// normalizeStringOrStringSlice converts a string or string-like array value into a
// []string, ignoring non-string array elements.
func normalizeStringOrStringSlice(raw any) []string {
	if s, ok := raw.(string); ok {
		return []string{s}
	}
	return parseStringSliceAny(raw, nil)
}

// extractDeploymentStatusStateCondition reads on.deployment_status.state and converts it
// into a GitHub Actions expression string (without ${{ }} wrappers). Returns "" if not set.
func extractDeploymentStatusStateCondition(frontmatter map[string]any) string {
	dsMap, ok := extractOnTriggerMap(frontmatter, "deployment_status")
	if !ok {
		return ""
	}
	stateValue, ok := dsMap["state"]
	if !ok {
		return ""
	}

	// GitHub Actions allows state as a single string or an array
	states := normalizeStringOrStringSlice(stateValue)

	if len(states) == 0 {
		return ""
	}

	parts := make([]string, 0, len(states))
	for _, s := range states {
		parts = append(parts, "github.event.deployment_status.state == '"+s+"'")
	}
	stateExpr := strings.Join(parts, " || ")

	// Guard the state check with an event_name test so the condition remains true
	// when the workflow is triggered by other events (e.g. workflow_dispatch).
	// Without the guard, a non-deployment_status event would see the state as
	// empty/undefined and the entire activation condition would evaluate to false.
	return "github.event_name != 'deployment_status' || (" + stateExpr + ")"
}

// validWorkflowRunConclusions is the exhaustive list of conclusion values that GitHub
// Actions emits for workflow_run events.  Values outside this set are rejected at
// compile time to prevent expression injection (a raw value is interpolated directly
// into a GitHub Actions expression string).
var validWorkflowRunConclusions = []string{
	"success",
	"failure",
	"neutral",
	"cancelled",
	"skipped",
	"timed_out",
	"action_required",
	"stale",
}

// isValidWorkflowRunConclusion reports whether v is a recognised conclusion value.
func isValidWorkflowRunConclusion(v string) bool {
	return slices.Contains(validWorkflowRunConclusions, v)
}

// extractWorkflowRunConclusionCondition reads on.workflow_run.conclusion and converts it
// into a GitHub Actions expression string (without ${{ }} wrappers). Returns "" if not set.
func extractWorkflowRunConclusionCondition(frontmatter map[string]any) (string, error) {
	wrMap, ok := extractOnTriggerMap(frontmatter, "workflow_run")
	if !ok {
		return "", nil
	}
	conclusionValue, ok := wrMap["conclusion"]
	if !ok {
		return "", nil
	}

	conclusions := normalizeStringOrStringSlice(conclusionValue)

	if len(conclusions) == 0 {
		return "", nil
	}

	for _, c := range conclusions {
		if !isValidWorkflowRunConclusion(c) {
			return "", fmt.Errorf("invalid on.workflow_run.conclusion value %q: must be one of %s",
				c, strings.Join(validWorkflowRunConclusions, ", "))
		}
	}

	parts := make([]string, 0, len(conclusions))
	for _, c := range conclusions {
		parts = append(parts, "github.event.workflow_run.conclusion == '"+c+"'")
	}
	conclusionExpr := strings.Join(parts, " || ")

	// Guard the conclusion check with an event_name test so the condition remains true
	// when the workflow is triggered by other events (e.g. workflow_dispatch).
	// Without the guard, a non-workflow_run event would see conclusion as
	// empty/undefined and the entire activation condition would evaluate to false.
	return "github.event_name != 'workflow_run' || (" + conclusionExpr + ")", nil
}

// extractExpressionFromIfString extracts the expression part from a string that might
// contain "if: expression" or just "expression", returning just the expression
func (c *Compiler) extractExpressionFromIfString(ifString string) string {
	if ifString == "" {
		return ""
	}

	// Check if the string starts with "if: " and strip it
	if strings.HasPrefix(ifString, "if: ") {
		expr := strings.TrimSpace(ifString[4:]) // Remove "if: " prefix
		frontmatterLog.Printf("Stripped 'if: ' prefix from if condition: %s", expr)
		return expr
	}

	// Return the string as-is (it's just the expression)
	return ifString
}

// extractCommandConfig extracts command configuration from frontmatter including name, events,
// centralized routing strategy, and optional footer placeholder for slash_command.
func (c *Compiler) extractCommandConfig(frontmatter map[string]any) (commandNames []string, commandEvents []string, commandCentralized bool, commandPlaceholder string) {
	frontmatterLog.Print("Extracting command configuration from frontmatter")
	// Check new format: on.slash_command or on.slash_command.name (preferred)
	// Also check legacy format: on.command or on.command.name (deprecated)
	commandValue, hasCommand := extractOnTriggerValue(frontmatter, "slash_command")
	isDeprecated := false
	if !hasCommand {
		commandValue, hasCommand = extractOnTriggerValue(frontmatter, "command")
		isDeprecated = hasCommand
	}
	if hasCommand {
		// Show deprecation warning if using old field name
		if isDeprecated {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("The 'command:' trigger field is deprecated. Please use 'slash_command:' instead."))
			c.IncrementWarningCount()
		}

		// Check if command is a string (shorthand format)
		if commandStr, ok := commandValue.(string); ok {
			frontmatterLog.Printf("Extracted command name (shorthand): %s", commandStr)
			return []string{commandStr}, nil, false, "" // nil means default (all events)
		}
		// Check if command is a map with a name key (object format)
		if commandMap, ok := commandValue.(map[string]any); ok {
			var events []string
			centralized := false
			placeholder := ""
			names := normalizeStringOrStringSlice(commandMap["name"])

			// Extract events field
			if eventsValue, hasEvents := commandMap["events"]; hasEvents {
				events = ParseCommandEvents(eventsValue)
			}

			if strategyRaw, hasStrategy := commandMap["strategy"]; hasStrategy {
				if strategy, ok := strategyRaw.(string); ok && strings.EqualFold(strings.TrimSpace(strategy), "centralized") {
					centralized = true
				}
			}

			// Extract optional placeholder for footer hint text
			if placeholderRaw, hasPlaceholder := commandMap["placeholder"]; hasPlaceholder {
				if placeholderStr, ok := placeholderRaw.(string); ok {
					if trimmed := strings.TrimSpace(placeholderStr); trimmed != "" {
						placeholder = trimmed
					}
				}
			}

			frontmatterLog.Printf("Extracted command config: names=%v, events=%v, centralized=%v, placeholder=%q", names, events, centralized, placeholder)
			return names, events, centralized, placeholder
		}
	}

	return nil, nil, false, ""
}

// extractLabelCommandConfig extracts the label-command configuration from frontmatter
// including label name(s), the events field, strategy, and the remove_label flag.
// It reads on.label_command which can be:
//   - a string: label name directly (e.g. label_command: "deploy")
//   - a map with "name" or "names", optional "events", optional "strategy", and optional "remove_label" fields
//
// Returns (labelNames, labelEvents, decentralized, removeLabel) where labelEvents is nil for default (all events)
// and removeLabel defaults to true when not specified.
func (c *Compiler) extractLabelCommandConfig(frontmatter map[string]any) (labelNames []string, labelEvents []string, decentralized bool, removeLabel bool) {
	frontmatterLog.Print("Extracting label-command configuration from frontmatter")
	labelCommandValue, hasLabelCommand := extractOnTriggerValue(frontmatter, "label_command")
	if !hasLabelCommand {
		return nil, nil, false, true
	}

	// Simple string form: label_command: "my-label"
	if nameStr, ok := labelCommandValue.(string); ok {
		frontmatterLog.Printf("Extracted label-command name (shorthand): %s", nameStr)
		return []string{nameStr}, nil, false, true
	}

	// Map form: label_command: {name: "...", names: [...], events: [...], remove_label: bool}
	if lcMap, ok := labelCommandValue.(map[string]any); ok {
		var events []string
		decentralized := false
		removeLabelVal := true // default to true
		names := normalizeStringOrStringSlice(lcMap["name"])

		if namesVal, hasNames := lcMap["names"]; hasNames {
			names = append(names, normalizeStringOrStringSlice(namesVal)...)
		}

		if eventsVal, hasEvents := lcMap["events"]; hasEvents {
			events = ParseCommandEvents(eventsVal)
		}

		if strategyVal, hasStrategy := lcMap["strategy"]; hasStrategy {
			if strategy, ok := strategyVal.(string); ok && strings.EqualFold(strings.TrimSpace(strategy), "decentralized") {
				decentralized = true
			}
		}

		if removeLabelField, hasRemoveLabel := lcMap["remove_label"]; hasRemoveLabel {
			if b, ok := removeLabelField.(bool); ok {
				removeLabelVal = b
			}
		}

		frontmatterLog.Printf("Extracted label-command config: names=%v, events=%v, decentralized=%v, remove_label=%v", names, events, decentralized, removeLabelVal)
		return names, events, decentralized, removeLabelVal
	}

	return nil, nil, false, true
}

// isGitHubAppNestedField returns true if the trimmed YAML line represents a known
// nested field or array item inside an on.github-app object.
func isGitHubAppNestedField(trimmedLine string) bool {
	githubAppFields := []string{"app-id:", "client-id:", "private-key:", "ignore-if-missing:", "owner:", "repositories:"}
	for _, field := range githubAppFields {
		if strings.HasPrefix(trimmedLine, field) {
			return true
		}
	}
	// Array items (repositories list)
	return strings.HasPrefix(trimmedLine, "-")
}
