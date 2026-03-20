package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputsAppConcurrencyLog = logger.New("workflow:safe_outputs_app_concurrency")

// hasCancelInProgress reports whether the given WorkflowData has workflow-level
// cancel-in-progress: true set. It checks two sources:
//
//  1. The structured RawFrontmatter map (most reliable; works for user-defined concurrency).
//  2. The rendered Concurrency YAML string (covers compiler-generated concurrency e.g. for PR
//     workflows where cancel-in-progress is emitted by GenerateConcurrencyConfig).
func hasCancelInProgress(workflowData *WorkflowData) bool {
	// Check 1: structured frontmatter
	if raw := workflowData.RawFrontmatter; raw != nil {
		if concurrencyRaw, ok := raw["concurrency"]; ok {
			if concurrencyMap, ok := concurrencyRaw.(map[string]any); ok {
				if cancelRaw, ok := concurrencyMap["cancel-in-progress"]; ok {
					// The YAML library parses `true` as bool; accept the boolean form
					if cancelBool, ok := cancelRaw.(bool); ok && cancelBool {
						return true
					}
					// Defensive: also accept the string "true"
					if cancelStr, ok := cancelRaw.(string); ok && cancelStr == "true" {
						return true
					}
				}
			}
		}
	}

	// Check 2: rendered concurrency YAML string (e.g. compiler-generated)
	return strings.Contains(workflowData.Concurrency, "cancel-in-progress: true")
}

// validateSafeOutputsAppConcurrencyWarning warns when a workflow combines three
// conditions that can cause the original run to self-cancel:
//
//  1. workflow-level cancel-in-progress: true
//  2. comment-based triggers (issue_comment or slash_command)
//  3. safe-outputs.github-app is configured
//
// When all three are present, the GitHub App-authored safe-output comment can
// trigger a passive issue_comment run of the same workflow. Because the new run
// shares the same workflow-level concurrency group (keyed by issue number), the
// cancel-in-progress policy cancels the original run before it finishes.
//
// Note: safe-outputs.concurrency-group does NOT protect against this pattern.
// It only sets concurrency on the safe_outputs job itself, not on the triggered
// passive workflow run that enters the workflow-level concurrency group.
func (c *Compiler) validateSafeOutputsAppConcurrencyWarning(workflowData *WorkflowData) {
	// Check 1: safe-outputs.github-app is configured (directly or via top-level fallback).
	// The fallback is already applied before compilation so SafeOutputs.GitHubApp is set.
	if workflowData.SafeOutputs == nil || workflowData.SafeOutputs.GitHubApp == nil {
		safeOutputsAppConcurrencyLog.Print("No safe-outputs github-app configured, skipping warning")
		return
	}

	// Check 2: workflow has comment-based triggers (issue_comment or slash_command).
	// slash_command expands to issue_comment + workflow_dispatch at compile time,
	// so both cases mean the workflow can be triggered by comments.
	on := workflowData.On
	hasCommentTrigger := isIssueWorkflow(on) || isSlashCommandWorkflow(on)
	if !hasCommentTrigger {
		safeOutputsAppConcurrencyLog.Print("No comment-based triggers found, skipping warning")
		return
	}

	// Check 3: workflow-level concurrency has cancel-in-progress: true.
	if !hasCancelInProgress(workflowData) {
		safeOutputsAppConcurrencyLog.Print("cancel-in-progress: true not found in concurrency config, skipping warning")
		return
	}

	safeOutputsAppConcurrencyLog.Print("Dangerous combination detected: emitting self-cancel warning")

	msg := strings.Join([]string{
		"safe-outputs.github-app combined with comment triggers and cancel-in-progress: true can cause self-cancellation.",
		"",
		"When safe-outputs posts a comment using the GitHub App token, that comment triggers a passive",
		"issue_comment run of the same workflow. If the passive run resolves to the same workflow-level",
		"concurrency group as the original run, cancel-in-progress: true will cancel the original run",
		"before it finishes — even before the passive run reaches pre_activation.",
		"",
		"Note: safe-outputs.concurrency-group does NOT protect against this. It only controls concurrency",
		"on the safe_outputs job itself, not on the passive workflow run that enters the workflow-level",
		"concurrency group.",
		"",
		"To prevent this, give passive comment-triggered runs a unique workflow-level concurrency key.",
		"For example, check github.event.comment.body before resolving to a shared per-issue key:",
		"",
		"  concurrency:",
		"    group: >-",
		"      gh-aw-${{ github.workflow }}-${{",
		"        github.event_name == 'issue_comment' &&",
		"        !startsWith(github.event.comment.body, '/your-command') &&",
		"        format('passive-comment-{0}', github.run_id) ||",
		"        github.event.issue.number ||",
		"        github.event.inputs.issue_number ||",
		"        github.run_id",
		"      }}",
		"    cancel-in-progress: true",
		"",
		"See: https://github.github.com/gh-aw/reference/concurrency/",
	}, "\n")

	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(msg))
	c.IncrementWarningCount()
}
