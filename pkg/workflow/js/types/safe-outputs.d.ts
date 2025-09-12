// TypeScript definitions for GitHub Agentic Workflows Safe Outputs
// This file provides type definitions for the safe output system
// including configuration types, JSONL output types, and processing functions

/**
 * Common configuration fields shared across output types
 */
interface BaseOutputConfig {
  /** Maximum number of items to create/process (default varies by type) */
  max?: number;
}

/**
 * Configuration for creating GitHub issues from agent output
 */
interface CreateIssuesConfig extends BaseOutputConfig {
  /** Prefix to add to issue titles */
  "title-prefix"?: string;
  /** Labels to automatically add to created issues */
  labels?: string[];
  /** Maximum number of issues to create (default: 1) */
  max?: number;
}

/**
 * Configuration for creating GitHub discussions from agent output
 */
interface CreateDiscussionsConfig extends BaseOutputConfig {
  /** Prefix to add to discussion titles */
  "title-prefix"?: string;
  /** Discussion category ID */
  "category-id"?: string;
  /** Maximum number of discussions to create (default: 1) */
  max?: number;
}

/**
 * Configuration for adding comments to issues/PRs from agent output
 */
interface AddIssueCommentsConfig extends BaseOutputConfig {
  /** Maximum number of comments to create (default: 1) */
  max?: number;
  /** Target for comments: "triggering" (default), "*" (any issue), or explicit issue number */
  target?: string;
}

/**
 * Configuration for creating pull requests from agent output
 */
interface CreatePullRequestsConfig extends BaseOutputConfig {
  /** Prefix to add to PR titles */
  "title-prefix"?: string;
  /** Labels to automatically add to created PRs */
  labels?: string[];
  /** Whether to create as draft PR (default: true) */
  draft?: boolean;
  /** Maximum number of pull requests to create (default: 1) */
  max?: number;
  /** Behavior when no changes to commit: "warn" (default), "error", or "ignore" */
  "if-no-changes"?: "warn" | "error" | "ignore";
}

/**
 * Configuration for creating pull request review comments from agent output
 */
interface CreatePullRequestReviewCommentsConfig extends BaseOutputConfig {
  /** Maximum number of review comments to create (default: 1) */
  max?: number;
  /** Side of the diff: "LEFT" or "RIGHT" (default: "RIGHT") */
  side?: "LEFT" | "RIGHT";
}

/**
 * Configuration for creating code scanning alerts (SARIF format) from agent output
 */
interface CreateCodeScanningAlertsConfig extends BaseOutputConfig {
  /** Maximum number of security findings to include (default: unlimited) */
  max?: number;
  /** Driver name for SARIF tool.driver.name field */
  driver?: string;
}

/**
 * Configuration for adding labels to issues/PRs from agent output
 */
interface AddIssueLabelsConfig extends BaseOutputConfig {
  /** Optional list of allowed labels. If omitted, any labels are allowed */
  allowed?: string[];
  /** Maximum number of labels to add (default: 3) */
  max?: number;
}

/**
 * Configuration for updating GitHub issues from agent output
 */
interface UpdateIssuesConfig extends BaseOutputConfig {
  /** Allow updating issue status (open/closed) */
  status?: boolean;
  /** Target for updates: "triggering" (default), "*" (any issue), or explicit issue number */
  target?: string;
  /** Allow updating issue title */
  title?: boolean;
  /** Allow updating issue body */
  body?: boolean;
  /** Maximum number of issues to update (default: 1) */
  max?: number;
}

/**
 * Configuration for pushing changes to a specific branch from agent output
 */
interface PushToPullRequestBranchConfig extends BaseOutputConfig {
  /** Target for push: like add-issue-comment but for pull requests */
  target?: string;
  /** Behavior when no changes to push: "warn", "error", or "ignore" (default: "warn") */
  "if-no-changes"?: "warn" | "error" | "ignore";
}

/**
 * Configuration for reporting missing tools or functionality
 */
interface MissingToolConfig extends BaseOutputConfig {
  /** Maximum number of missing tool reports (default: unlimited) */
  max?: number;
}

/**
 * Complete safe outputs configuration structure
 */
interface SafeOutputsConfig {
  /** Configuration for creating GitHub issues */
  "create-issue"?: CreateIssuesConfig;
  /** Configuration for creating GitHub discussions */
  "create-discussion"?: CreateDiscussionsConfig;
  /** Configuration for adding comments to issues/PRs */
  "add-issue-comment"?: AddIssueCommentsConfig;
  /** Configuration for creating pull requests */
  "create-pull-request"?: CreatePullRequestsConfig;
  /** Configuration for creating pull request review comments */
  "create-pull-request-review-comment"?: CreatePullRequestReviewCommentsConfig;
  /** Configuration for creating code scanning alerts */
  "create-code-scanning-alert"?: CreateCodeScanningAlertsConfig;
  /** Configuration for adding labels to issues/PRs */
  "add-issue-label"?: AddIssueLabelsConfig;
  /** Configuration for updating issues */
  "update-issue"?: UpdateIssuesConfig;
  /** Configuration for pushing to branch */
  "push-to-pr-branch"?: PushToPullRequestBranchConfig;
  /** Configuration for missing tool reporting */
  "missing-tool"?: MissingToolConfig;
  /** List of allowed domains for URL sanitization */
  "allowed-domains"?: string[];
  /** If true, emit step summary messages instead of making GitHub API calls */
  staged?: boolean;
}

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

// === Processing and Validation Types ===

/**
 * Validated output structure returned by collect_ndjson_output.cjs
 */
interface ValidatedOutput {
  /** Array of validated safe output items */
  items: SafeOutputItem[];
  /** Array of validation error messages */
  errors: string[];
}

/**
 * Content sanitization function signature
 */
type SanitizeContentFunction = (content: string) => string;

/**
 * JSON repair function signature for fixing malformed JSON
 */
type RepairJsonFunction = (jsonStr: string) => string;

/**
 * JSON parsing function with repair fallback
 */
type ParseJsonWithRepairFunction = (jsonStr: string) => any | undefined;

/**
 * Function to get maximum allowed count for an output type
 */
type GetMaxAllowedForTypeFunction = (
  itemType: string,
  config: SafeOutputsConfig
) => number;

// === Environment Variable Types ===

/**
 * Environment variables used by safe output processing
 */
interface SafeOutputEnvironment {
  /** Path to the JSONL output file */
  GITHUB_AW_SAFE_OUTPUTS?: string;
  /** JSON string of safe outputs configuration */
  GITHUB_AW_SAFE_OUTPUTS_CONFIG?: string;
  /** Path to the validated agent output JSON file */
  GITHUB_AW_AGENT_OUTPUT?: string;
  /** Comma-separated list of allowed domains */
  GITHUB_AW_ALLOWED_DOMAINS?: string;
  /** Whether safe outputs are in staged mode */
  GITHUB_AW_SAFE_OUTPUTS_STAGED?: string;
  /** Workflow ID for branching */
  GITHUB_AW_WORKFLOW_ID?: string;
  /** Base branch for PRs */
  GITHUB_AW_BASE_BRANCH?: string;
  /** PR title prefix */
  GITHUB_AW_PR_TITLE_PREFIX?: string;
  /** Comma-separated PR labels */
  GITHUB_AW_PR_LABELS?: string;
  /** PR draft setting */
  GITHUB_AW_PR_DRAFT?: string;
  /** PR if-no-changes behavior */
  GITHUB_AW_PR_IF_NO_CHANGES?: string;
  /** Issue title prefix */
  GITHUB_AW_ISSUE_TITLE_PREFIX?: string;
  /** Comma-separated issue labels */
  GITHUB_AW_ISSUE_LABELS?: string;
  /** Maximum missing tool reports */
  GITHUB_AW_MISSING_TOOL_MAX?: string;
  /** Maximum security reports */
  GITHUB_AW_SECURITY_REPORT_MAX?: string;
  /** Security report driver name */
  GITHUB_AW_SECURITY_REPORT_DRIVER?: string;
  /** Workflow filename for rule ID prefix */
  GITHUB_AW_WORKFLOW_FILENAME?: string;
  /** Push target configuration */
  GITHUB_AW_PUSH_TARGET?: string;
  /** Push if-no-changes behavior */
  GITHUB_AW_PUSH_IF_NO_CHANGES?: string;
}

// === Export all types ===
export {
  // Configuration types
  BaseOutputConfig,
  CreateIssuesConfig,
  CreateDiscussionsConfig,
  AddIssueCommentsConfig,
  CreatePullRequestsConfig,
  CreatePullRequestReviewCommentsConfig,
  CreateCodeScanningAlertsConfig,
  AddIssueLabelsConfig,
  UpdateIssuesConfig,
  PushToPullRequestBranchConfig,
  MissingToolConfig,
  SafeOutputsConfig,

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

  // Processing types
  ValidatedOutput,
  SanitizeContentFunction,
  RepairJsonFunction,
  ParseJsonWithRepairFunction,
  GetMaxAllowedForTypeFunction,

  // Environment types
  SafeOutputEnvironment,
};
