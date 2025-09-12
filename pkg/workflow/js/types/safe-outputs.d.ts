// TypeScript definitions for GitHub Agentic Workflows Safe Outputs JSONL Types
// This file provides type definitions for JSONL output items produced by agents

// === JSONL Output Item Types ===

/**
 * Base interface for all safe output items
 */
interface BaseSafeOutputItem {
  /** The type of safe output action */
  type: string;
}

/**
 * JSONL item for creating a GitHub issue
 */
interface CreateIssueItem extends BaseSafeOutputItem {
  type: "create-issue";
  /** Issue title */
  title: string;
  /** Issue body content */
  body: string;
  /** Optional labels to add to the issue */
  labels?: string[];
}

/**
 * JSONL item for creating a GitHub discussion
 */
interface CreateDiscussionItem extends BaseSafeOutputItem {
  type: "create-discussion";
  /** Discussion title */
  title: string;
  /** Discussion body content */
  body: string;
}

/**
 * JSONL item for adding a comment to an issue or PR
 */
interface AddIssueCommentItem extends BaseSafeOutputItem {
  type: "add-issue-comment";
  /** Comment body content */
  body: string;
}

/**
 * JSONL item for creating a pull request
 */
interface CreatePullRequestItem extends BaseSafeOutputItem {
  type: "create-pull-request";
  /** Pull request title */
  title: string;
  /** Pull request body content */
  body: string;
  /** Optional branch name (will be auto-generated if not provided) */
  branch?: string;
  /** Optional labels to add to the PR */
  labels?: string[];
}

/**
 * JSONL item for creating a pull request review comment
 */
interface CreatePullRequestReviewCommentItem extends BaseSafeOutputItem {
  type: "create-pull-request-review-comment";
  /** File path for the review comment */
  path: string;
  /** Line number for the comment */
  line: number | string;
  /** Comment body content */
  body: string;
  /** Optional start line for multi-line comments */
  start_line?: number | string;
  /** Optional side of the diff: "LEFT" or "RIGHT" */
  side?: "LEFT" | "RIGHT";
}

/**
 * JSONL item for creating a code scanning alert
 */
interface CreateCodeScanningAlertItem extends BaseSafeOutputItem {
  type: "create-code-scanning-alert";
  /** File path where the issue was found */
  file: string;
  /** Line number where the issue was found */
  line: number | string;
  /** Severity level: "error", "warning", "info", or "note" */
  severity: "error" | "warning" | "info" | "note";
  /** Alert message describing the issue */
  message: string;
  /** Optional column number */
  column?: number | string;
  /** Optional rule ID suffix for uniqueness */
  ruleIdSuffix?: string;
}

/**
 * JSONL item for adding labels to an issue or PR
 */
interface AddIssueLabelItem extends BaseSafeOutputItem {
  type: "add-issue-label";
  /** Array of label names to add */
  labels: string[];
}

/**
 * JSONL item for updating an issue
 */
interface UpdateIssueItem extends BaseSafeOutputItem {
  type: "update-issue";
  /** Optional new issue status */
  status?: "open" | "closed";
  /** Optional new issue title */
  title?: string;
  /** Optional new issue body */
  body?: string;
  /** Optional issue number for target "*" */
  issue_number?: number | string;
}

/**
 * JSONL item for pushing to a PR branch
 */
interface PushToPrBranchItem extends BaseSafeOutputItem {
  type: "push-to-pr-branch";
  /** Optional commit message */
  message?: string;
  /** Optional pull request number for target "*" */
  pull_request_number?: number | string;
}

/**
 * JSONL item for reporting missing tools
 */
interface MissingToolItem extends BaseSafeOutputItem {
  type: "missing-tool";
  /** Name of the missing tool */
  tool: string;
  /** Reason why the tool is needed */
  reason: string;
  /** Optional alternatives or workarounds */
  alternatives?: string;
}

/**
 * Union type of all possible safe output items
 */
type SafeOutputItem =
  | CreateIssueItem
  | CreateDiscussionItem
  | AddIssueCommentItem
  | CreatePullRequestItem
  | CreatePullRequestReviewCommentItem
  | CreateCodeScanningAlertItem
  | AddIssueLabelItem
  | UpdateIssueItem
  | PushToPrBranchItem
  | MissingToolItem;



// === Export JSONL types ===
export {
  // JSONL item types
  BaseSafeOutputItem,
  CreateIssueItem,
  CreateDiscussionItem,
  AddIssueCommentItem,
  CreatePullRequestItem,
  CreatePullRequestReviewCommentItem,
  CreateCodeScanningAlertItem,
  AddIssueLabelItem,
  UpdateIssueItem,
  PushToPrBranchItem,
  MissingToolItem,
  SafeOutputItem,
};
