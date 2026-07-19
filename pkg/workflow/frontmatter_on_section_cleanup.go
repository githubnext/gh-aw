package workflow

import (
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/setutil"
)

// commentOutProcessedFieldsInOnSection comments out draft, fork, forks, names, labels, manual-approval, stop-after, skip-if-match, skip-if-no-match, skip-roles, reaction, lock-for-agent, steps, permissions, needs, restore-memory, and stale-check fields in the on section
// These fields are processed separately and should be commented for documentation
// Exception: names fields in sections with __gh_aw_native_label_filter__ marker in frontmatter are NOT commented out
type onSectionCleanupState struct {
	result []string

	inPullRequest                bool
	inIssues                     bool
	inDiscussion                 bool
	inIssueComment               bool
	inDeploymentStatus           bool
	inWorkflowRun                bool
	inWorkflowRunConclusionArray bool
	inForksArray                 bool
	inSkipIfMatch                bool
	inSkipIfNoMatch              bool
	inSkipIfCheckFailing         bool
	inSkipAuthorAssociations     bool
	inSkipRolesArray             bool
	inSkipBotsArray              bool
	inRolesArray                 bool
	inBotsArray                  bool
	inLabelsArray                bool
	inNeedsArray                 bool
	inGitHubApp                  bool
	inOnSteps                    bool
	inOnPermissions              bool
	commentBlockIndent           string
	inCommentBlock               bool
	currentSection               string
	currentSectionIndent         int
	deploymentStatusIndent       int
	workflowRunIndent            int
	nativeLabelFilterSections    map[string]struct{}
}

func (c *Compiler) commentOutProcessedFieldsInOnSection(yamlStr string, frontmatter map[string]any) string {
	frontmatterLog.Print("Processing 'on' section to comment out processed fields")

	state := newOnSectionCleanupState(frontmatter)
	for _, line := range strings.Split(yamlStr, "\n") {
		state.processLine(line)
	}

	state.result = dedentTrailingOnCommentBlock(state.result)
	return strings.Join(state.result, "\n")
}

func newOnSectionCleanupState(frontmatter map[string]any) *onSectionCleanupState {
	return &onSectionCleanupState{
		currentSectionIndent:      -1,
		deploymentStatusIndent:    -1,
		workflowRunIndent:         -1,
		nativeLabelFilterSections: nativeLabelFilterSections(frontmatter),
	}
}

func nativeLabelFilterSections(frontmatter map[string]any) map[string]struct{} {
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
		sectionMap, ok := sectionValue.(map[string]any)
		if !hasSec || !ok {
			continue
		}
		marker, hasMarker := sectionMap["__gh_aw_native_label_filter__"]
		useNative, ok := marker.(bool)
		if hasMarker && ok && useNative {
			sections[sectionKey] = struct{}{}
			frontmatterLog.Printf("Section %s uses native label filtering", sectionKey)
		}
	}
	return sections
}

func (s *onSectionCleanupState) processLine(line string) {
	trimmedLine := strings.TrimSpace(line)
	lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))

	if s.tryEnterEventSection(line, trimmedLine, lineIndent) {
		return
	}
	s.updateEventSectionExits(line, trimmedLine, lineIndent)
	if s.shouldSkipNativeLabelMarker(trimmedLine) {
		return
	}
	s.updateEntryState(trimmedLine, lineIndent)
	s.updateExitState(line, trimmedLine)

	shouldComment, commentReason := s.commentDecision(line, trimmedLine, lineIndent)
	s.appendProcessedLine(line, shouldComment, commentReason)
}

func (s *onSectionCleanupState) tryEnterEventSection(line, trimmedLine string, lineIndent int) bool {
	if s.inOnPermissions || s.inOnSteps || s.inSkipAuthorAssociations {
		return false
	}
	if lineIndent != 2 && lineIndent != 4 {
		return false
	}

	section := ""
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

	s.activateEventSection(section, lineIndent)
	s.result = append(s.result, line)
	return true
}

func (s *onSectionCleanupState) activateEventSection(section string, indent int) {
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
	s.inCommentBlock = false
	s.commentBlockIndent = ""
	s.inPullRequest = section == "pull_request"
	s.inIssues = section == "issues"
	s.inDiscussion = section == "discussion"
	s.inIssueComment = section == "issue_comment"
	s.inDeploymentStatus = section == "deployment_status"
	s.inWorkflowRun = section == "workflow_run"
	s.inWorkflowRunConclusionArray = false
	s.inForksArray = false

	s.currentSection = ""
	s.currentSectionIndent = -1
	if section == "pull_request" || section == "issues" || section == "discussion" || section == "issue_comment" {
		s.currentSection = section
		s.currentSectionIndent = indent
	}
	s.deploymentStatusIndent = -1
	if section == "deployment_status" {
		s.deploymentStatusIndent = indent
	}
	s.workflowRunIndent = -1
	if section == "workflow_run" {
		s.workflowRunIndent = indent
	}
}

func (s *onSectionCleanupState) updateEventSectionExits(line, trimmedLine string, lineIndent int) {
	if (s.inPullRequest || s.inIssues || s.inDiscussion || s.inIssueComment) && isRealLineAtOrAbove(line, trimmedLine, lineIndent, s.currentSectionIndent) {
		s.inPullRequest = false
		s.inIssues = false
		s.inDiscussion = false
		s.inIssueComment = false
		s.inForksArray = false
		s.currentSection = ""
		s.currentSectionIndent = -1
	}
	if s.inDeploymentStatus && isRealLineAtOrAbove(line, trimmedLine, lineIndent, s.deploymentStatusIndent) {
		s.inDeploymentStatus = false
		s.deploymentStatusIndent = -1
	}
	if s.inWorkflowRun && isRealLineAtOrAbove(line, trimmedLine, lineIndent, s.workflowRunIndent) {
		s.inWorkflowRun = false
		s.inWorkflowRunConclusionArray = false
		s.workflowRunIndent = -1
	}
}

func isRealLineAtOrAbove(line, trimmedLine string, lineIndent, sectionIndent int) bool {
	return strings.TrimSpace(line) != "" && !strings.HasPrefix(trimmedLine, "#") && sectionIndent >= 0 && lineIndent <= sectionIndent
}

func (s *onSectionCleanupState) shouldSkipNativeLabelMarker(trimmedLine string) bool {
	return (s.inPullRequest || s.inIssues || s.inDiscussion || s.inIssueComment) && strings.Contains(trimmedLine, "__gh_aw_native_label_filter__:")
}

func (s *onSectionCleanupState) updateEntryState(trimmedLine string, lineIndent int) {
	s.updateTopLevelArrayEntryState(trimmedLine, lineIndent)
	s.updateObjectEntryState(trimmedLine)
}

func (s *onSectionCleanupState) updateTopLevelArrayEntryState(trimmedLine string, lineIndent int) {
	inEvent := s.inPullRequest || s.inIssues || s.inDiscussion || s.inIssueComment
	if s.inPullRequest && strings.HasPrefix(trimmedLine, "forks:") {
		s.inForksArray = true
	}
	if inEvent {
		return
	}
	if strings.HasPrefix(trimmedLine, "skip-roles:") {
		s.inSkipRolesArray = true
	}
	if strings.HasPrefix(trimmedLine, "skip-bots:") {
		s.inSkipBotsArray = true
	}
	if strings.HasPrefix(trimmedLine, "roles:") {
		s.inRolesArray = true
	}
	if strings.HasPrefix(trimmedLine, "bots:") {
		s.inBotsArray = true
	}
	if !s.inOnSteps && !s.inOnPermissions && lineIndent == 2 && trimmedLine == "labels:" {
		s.inLabelsArray = true
	}
	if !s.inOnSteps && !s.inOnPermissions && lineIndent == 2 && strings.HasPrefix(trimmedLine, "needs:") {
		s.inNeedsArray = true
	}
	if strings.HasPrefix(trimmedLine, "steps:") {
		s.inOnSteps = true
	}
	if !s.inOnPermissions && strings.HasPrefix(trimmedLine, "permissions:") {
		s.inOnPermissions = true
	}
}

func (s *onSectionCleanupState) updateObjectEntryState(trimmedLine string) {
	if s.inPullRequest || s.inIssues || s.inDiscussion || s.inIssueComment {
		return
	}
	if !s.inSkipIfMatch && ((strings.HasPrefix(trimmedLine, "skip-if-match:") && trimmedLine == "skip-if-match:") ||
		(strings.HasPrefix(trimmedLine, "# skip-if-match:") && strings.Contains(trimmedLine, "pre-activation job"))) {
		s.inSkipIfMatch = true
	}
	if !s.inSkipIfNoMatch && ((strings.HasPrefix(trimmedLine, "skip-if-no-match:") && trimmedLine == "skip-if-no-match:") ||
		(strings.HasPrefix(trimmedLine, "# skip-if-no-match:") && strings.Contains(trimmedLine, "pre-activation job"))) {
		s.inSkipIfNoMatch = true
	}
	if !s.inSkipIfCheckFailing && (trimmedLine == "skip-if-check-failing:" ||
		(strings.HasPrefix(trimmedLine, "# skip-if-check-failing:") && strings.Contains(trimmedLine, "pre-activation job"))) {
		s.inSkipIfCheckFailing = true
	}
	if !s.inSkipAuthorAssociations && strings.HasPrefix(trimmedLine, "skip-author-associations:") && trimmedLine == "skip-author-associations:" {
		s.inSkipAuthorAssociations = true
	}
	if !s.inGitHubApp && ((strings.HasPrefix(trimmedLine, "github-app:") && trimmedLine == "github-app:") ||
		(strings.HasPrefix(trimmedLine, "# github-app:") && strings.Contains(trimmedLine, "pre-activation job"))) {
		s.inGitHubApp = true
	}
}

func (s *onSectionCleanupState) updateExitState(line, trimmedLine string) {
	lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))
	s.updateObjectExitState(line, trimmedLine, lineIndent)
	s.updateArrayExitState(line, trimmedLine, lineIndent)
	s.updateOnStepsAndPermissionsExitState(line, trimmedLine, lineIndent)
}

func (s *onSectionCleanupState) updateObjectExitState(line, trimmedLine string, lineIndent int) {
	if s.inSkipIfMatch && shouldExitTopLevelObject(line, trimmedLine, lineIndent, "skip-if-match:") {
		s.inSkipIfMatch = false
	}
	if s.inSkipIfNoMatch && shouldExitTopLevelObject(line, trimmedLine, lineIndent, "skip-if-no-match:") {
		s.inSkipIfNoMatch = false
	}
	if s.inSkipIfCheckFailing && shouldExitTopLevelObject(line, trimmedLine, lineIndent, "skip-if-check-failing:") {
		s.inSkipIfCheckFailing = false
	}
	if s.inSkipAuthorAssociations && shouldExitTopLevelObject(line, trimmedLine, lineIndent, "skip-author-associations:") {
		s.inSkipAuthorAssociations = false
	}
	if s.inGitHubApp && shouldExitTopLevelObject(line, trimmedLine, lineIndent, "github-app:") {
		s.inGitHubApp = false
	}
}

func shouldExitTopLevelObject(line, trimmedLine string, lineIndent int, field string) bool {
	if strings.TrimSpace(line) == "" || strings.HasPrefix(trimmedLine, field) || strings.HasPrefix(trimmedLine, "# "+field) {
		return false
	}
	return lineIndent == 2 && !strings.HasPrefix(trimmedLine, "#")
}

func (s *onSectionCleanupState) updateArrayExitState(line, trimmedLine string, lineIndent int) {
	if s.inForksArray && s.inPullRequest && strings.TrimSpace(line) != "" &&
		lineIndent == 4 && !strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "forks:") {
		s.inForksArray = false
	}
	if s.inSkipRolesArray && shouldExitTopLevelArray(line, trimmedLine, lineIndent, "skip-roles:") {
		s.inSkipRolesArray = false
	}
	if s.inSkipBotsArray && shouldExitTopLevelArray(line, trimmedLine, lineIndent, "skip-bots:") {
		s.inSkipBotsArray = false
	}
	if s.inRolesArray && shouldExitTopLevelArray(line, trimmedLine, lineIndent, "roles:") {
		s.inRolesArray = false
	}
	if s.inBotsArray && shouldExitTopLevelArray(line, trimmedLine, lineIndent, "bots:") {
		s.inBotsArray = false
	}
	if s.inLabelsArray && shouldExitTopLevelArray(line, trimmedLine, lineIndent, "labels:") {
		s.inLabelsArray = false
	}
	if s.inNeedsArray && shouldExitTopLevelArray(line, trimmedLine, lineIndent, "needs:") {
		s.inNeedsArray = false
	}
}

func shouldExitTopLevelArray(line, trimmedLine string, lineIndent int, field string) bool {
	return strings.TrimSpace(line) != "" && lineIndent == 2 && !strings.HasPrefix(trimmedLine, "-") &&
		!strings.HasPrefix(trimmedLine, field) && !strings.HasPrefix(trimmedLine, "#")
}

func (s *onSectionCleanupState) updateOnStepsAndPermissionsExitState(line, trimmedLine string, lineIndent int) {
	if s.inOnSteps && strings.TrimSpace(line) != "" && lineIndent == 2 && !strings.HasPrefix(trimmedLine, "-") &&
		!strings.HasPrefix(trimmedLine, "steps:") && !strings.HasPrefix(trimmedLine, "#") {
		s.inOnSteps = false
	}
	if s.inOnPermissions && strings.TrimSpace(line) != "" && !strings.HasPrefix(trimmedLine, "permissions:") &&
		!strings.HasPrefix(trimmedLine, "# permissions:") && lineIndent == 2 && !strings.HasPrefix(trimmedLine, "#") {
		s.inOnPermissions = false
	}
}

func (s *onSectionCleanupState) commentDecision(line, trimmedLine string, lineIndent int) (bool, string) {
	if !s.inPullRequest && !s.inIssues && !s.inDiscussion && !s.inIssueComment {
		if shouldComment, reason := s.topLevelGateCommentDecision(trimmedLine, lineIndent); shouldComment {
			return true, reason
		}
		if shouldComment, reason := s.topLevelMetadataCommentDecision(trimmedLine, lineIndent); shouldComment {
			return true, reason
		}
	}
	return s.eventCommentDecision(line, trimmedLine)
}

func (s *onSectionCleanupState) topLevelGateCommentDecision(trimmedLine string, lineIndent int) (bool, string) {
	switch {
	case strings.HasPrefix(trimmedLine, "manual-approval:"):
		return true, " # Manual approval processed as environment field in activation job"
	case strings.HasPrefix(trimmedLine, "stop-after:"):
		return true, " # Stop-after processed as stop-time check in pre-activation job"
	case strings.HasPrefix(trimmedLine, "skip-if-match:"):
		return true, " # Skip-if-match processed as search check in pre-activation job"
	case s.inSkipIfMatch && hasAnyOnSectionPrefix(trimmedLine, "query:", "max:", "scope:"):
		return true, ""
	case strings.HasPrefix(trimmedLine, "skip-if-no-match:"):
		return true, " # Skip-if-no-match processed as search check in pre-activation job"
	case s.inSkipIfNoMatch && hasAnyOnSectionPrefix(trimmedLine, "query:", "min:", "scope:"):
		return true, ""
	case strings.HasPrefix(trimmedLine, "skip-if-check-failing:"):
		return true, " # Skip-if-check-failing processed as check status gate in pre-activation job"
	case s.inSkipIfCheckFailing && hasAnyOnSectionPrefix(trimmedLine, "include:", "exclude:", "branch:", "allow-pending:", "-"):
		return true, ""
	case strings.HasPrefix(trimmedLine, "skip-author-associations:"):
		return true, " # Skip-author-associations compiled into pre-activation job if condition"
	case s.inSkipAuthorAssociations && lineIndent > 2:
		return true, ""
	}
	return false, ""
}

func (s *onSectionCleanupState) topLevelMetadataCommentDecision(trimmedLine string, lineIndent int) (bool, string) {
	if shouldComment, reason := s.topLevelListCommentDecision(trimmedLine, lineIndent); shouldComment {
		return true, reason
	}
	switch {
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

func (s *onSectionCleanupState) topLevelListCommentDecision(trimmedLine string, lineIndent int) (bool, string) {
	switch {
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
	case !s.inOnSteps && !s.inOnPermissions && lineIndent == 2 && strings.HasPrefix(trimmedLine, "labels:"):
		return true, " # Label filtering applied via job conditions"
	case s.inLabelsArray && strings.HasPrefix(trimmedLine, "-"):
		return true, " # Label filtering applied via job conditions"
	case !s.inOnSteps && !s.inOnPermissions && lineIndent == 2 && strings.HasPrefix(trimmedLine, "needs:"):
		return true, " # Needs processed as dependency in pre-activation job"
	case s.inNeedsArray && strings.HasPrefix(trimmedLine, "-"):
		return true, " # Needs processed as dependency in pre-activation job"
	}
	return false, ""
}

func hasAnyOnSectionPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func (s *onSectionCleanupState) eventCommentDecision(line, trimmedLine string) (bool, string) {
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
	case s.isIssueLikeEvent() && strings.HasPrefix(trimmedLine, "lock-for-agent:"):
		return true, " # Lock-for-agent processed as issue locking in activation job"
	case s.isIssueLikeEvent() && strings.HasPrefix(trimmedLine, "names:"):
		return s.namesFieldCommentDecision()
	case s.isIssueLikeEvent() && line != "":
		return s.namesArrayItemCommentDecision(trimmedLine)
	}
	return false, ""
}

func (s *onSectionCleanupState) isIssueLikeEvent() bool {
	return s.inPullRequest || s.inIssues || s.inDiscussion || s.inIssueComment
}

func (s *onSectionCleanupState) namesFieldCommentDecision() (bool, string) {
	if setutil.Contains(s.nativeLabelFilterSections, s.currentSection) {
		return false, ""
	}
	return true, " # Label filtering applied via job conditions"
}

func (s *onSectionCleanupState) namesArrayItemCommentDecision(trimmedLine string) (bool, string) {
	if setutil.Contains(s.nativeLabelFilterSections, s.currentSection) || len(s.result) == 0 {
		return false, ""
	}
	for i := range slices.Backward(s.result) {
		prevTrimmed := strings.TrimSpace(s.result[i])
		if prevTrimmed == "" {
			continue
		}
		if strings.Contains(prevTrimmed, "names:") && strings.Contains(prevTrimmed, "# Label filtering") {
			return strings.HasPrefix(trimmedLine, "-"), " # Label filtering applied via job conditions"
		}
		if !strings.HasPrefix(prevTrimmed, "#") || !strings.Contains(prevTrimmed, "Label filtering") {
			break
		}
		if strings.HasPrefix(prevTrimmed, "# -") && strings.Contains(prevTrimmed, "Label filtering") {
			if strings.HasPrefix(trimmedLine, "-") {
				return true, " # Label filtering applied via job conditions"
			}
			continue
		}
		break
	}
	return false, ""
}

func (s *onSectionCleanupState) appendProcessedLine(line string, shouldComment bool, commentReason string) {
	if !shouldComment {
		s.inCommentBlock = false
		s.commentBlockIndent = ""
		s.result = append(s.result, line)
		return
	}

	trimmed := strings.TrimLeft(line, " \t")
	if s.inForksArray && strings.HasPrefix(trimmed, "-") {
		s.inCommentBlock = false
		s.commentBlockIndent = ""
	}
	if !s.inCommentBlock && trimmed != "" {
		s.commentBlockIndent = ""
		if len(line) > len(trimmed) {
			s.commentBlockIndent = line[:len(line)-len(trimmed)]
		}
		s.inCommentBlock = true
	}

	commentedLine := s.commentBlockIndent + "# " + trimmed + commentReason
	commentedLine = strings.TrimRight(commentedLine, " \t")
	s.result = append(s.result, commentedLine)
}

// dedentTrailingOnCommentBlock re-indents the final run of commented-out lines at the
// end of the `on:` section to column 0.
//
// A commented-out processed field (e.g. `# roles:` or `# state:`) that is the last
// thing in the `on:` section is followed, in the assembled workflow, by a top-level
// key (`permissions:`, `concurrency:`, …) at column 0. yamllint's comments-indentation
// rule flags a comment whose indentation matches neither the preceding content line nor
// the following one, so a trailing block indented under `on:` (indent 2 or deeper) is
// reported because the next real line sits at column 0. Aligning the trailing block to
// column 0 gives it a matching anchor and clears the warning.
//
// Only the final block is adjusted. Commented blocks with real `on:` content after them
// already have a matching indentation anchor below and are left untouched.
func dedentTrailingOnCommentBlock(lines []string) []string {
	// Locate the last non-blank line; the trailing block must end in a comment.
	last := len(lines) - 1
	for last >= 0 && strings.TrimSpace(lines[last]) == "" {
		last--
	}
	if last < 0 || !strings.HasPrefix(strings.TrimSpace(lines[last]), "#") {
		return lines
	}

	// Walk back over the consecutive comment lines that form the trailing block.
	start := last
	for start-1 >= 0 && strings.HasPrefix(strings.TrimSpace(lines[start-1]), "#") {
		start--
	}

	// The block must sit under the `on:` mapping (i.e. be indented) and be preceded by
	// real content, so a file that is entirely comments is left alone.
	if start == 0 {
		return lines
	}

	// Only dedent comments at the direct `on:` children level (≤2 spaces / 1 tab).
	// Comments nested deeper — e.g. `forks:` inside `pull_request:` at 4-space indent —
	// are part of a nested event section and must keep their indentation so that the
	// visual structure of the `on:` block is preserved.
	firstLineIndent := len(lines[start]) - len(strings.TrimLeft(lines[start], " \t"))
	if firstLineIndent > 2 {
		return lines
	}

	for i := start; i <= last; i++ {
		lines[i] = strings.TrimLeft(lines[i], " \t")
	}
	return lines
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
