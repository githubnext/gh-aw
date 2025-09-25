// Base interface for all safe output configurations
interface SafeOutputConfig {
  type: string;
  max?: number;
}

// === Specific Safe Output Configuration Interfaces ===

/**
 * Configuration for creating GitHub issues
 */
interface CreateIssueConfig extends SafeOutputConfig {
  "title-prefix"?: string;
  labels?: string[];
  max?: number;
  "github-token"?: string;
}

/**
 * Configuration for creating GitHub discussions
 */
interface CreateDiscussionConfig extends SafeOutputConfig {
  "title-prefix"?: string;
  "category-id"?: string;
  max?: number;
  "github-token"?: string;
}

/**
 * Configuration for adding comments to issues or PRs
 */
interface AddCommentConfig extends SafeOutputConfig {
  max?: number;
  target?: string;
  "github-token"?: string;
}

/**
 * Configuration for creating pull requests
 */
interface CreatePullRequestConfig extends SafeOutputConfig {
  "title-prefix"?: string;
  labels?: string[];
  draft?: boolean;
  max?: number;
  "if-no-changes"?: string;
  "github-token"?: string;
}

/**
 * Configuration for creating pull request review comments
 */
interface CreatePullRequestReviewCommentConfig extends SafeOutputConfig {
  max?: number;
  side?: string;
  "github-token"?: string;
}

/**
 * Configuration for creating code scanning alerts
 */
interface CreateCodeScanningAlertConfig extends SafeOutputConfig {
  max?: number;
  driver?: string;
  "github-token"?: string;
}

/**
 * Configuration for adding labels to issues or PRs
 */
interface AddLabelsConfig extends SafeOutputConfig {
  allowed?: string[];
  max?: number;
  "github-token"?: string;
}

/**
 * Configuration for updating issues
 */
interface UpdateIssueConfig extends SafeOutputConfig {
  status?: boolean;
  target?: string;
  title?: boolean;
  body?: boolean;
  max?: number;
  "github-token"?: string;
}

/**
 * Configuration for pushing to pull request branches
 */
interface PushToPullRequestBranchConfig extends SafeOutputConfig {
  target?: string;
  "if-no-changes"?: string;
  "github-token"?: string;
}

/**
 * Configuration for uploading assets
 */
interface UploadAssetConfig extends SafeOutputConfig {
  branch?: string;
  "max-size"?: number;
  "allowed-exts"?: string[];
  "github-token"?: string;
}

/**
 * Configuration for reporting missing tools
 */
interface MissingToolConfig extends SafeOutputConfig {
  max?: number;
  "github-token"?: string;
}

// Union type of all specific safe output configurations
type SpecificSafeOutputConfig =
  | CreateIssueConfig
  | CreateDiscussionConfig
  | AddCommentConfig
  | CreatePullRequestConfig
  | CreatePullRequestReviewCommentConfig
  | CreateCodeScanningAlertConfig
  | AddLabelsConfig
  | UpdateIssueConfig
  | PushToPullRequestBranchConfig
  | UploadAssetConfig
  | MissingToolConfig;

type SafeOutputConfigs = Record<string, SafeOutputConfig | SpecificSafeOutputConfig>;

export {
  SafeOutputConfig,
  SafeOutputConfigs,
  // Specific configuration types
  CreateIssueConfig,
  CreateDiscussionConfig,
  AddCommentConfig,
  CreatePullRequestConfig,
  CreatePullRequestReviewCommentConfig,
  CreateCodeScanningAlertConfig,
  AddLabelsConfig,
  UpdateIssueConfig,
  PushToPullRequestBranchConfig,
  UploadAssetConfig,
  MissingToolConfig,
  SpecificSafeOutputConfig,
};
