// Base interface for all safe output configurations
interface SafeOutputConfig {
  type: string;
  max?: number;
  min?: number;
  "github-token"?: string;
}

// === Specific Safe Output Configuration Interfaces ===

/**
 * Configuration for creating GitHub issues
 */
interface CreateIssueConfig extends SafeOutputConfig {
  "title-prefix"?: string;
  labels?: string[];
}

/**
 * Configuration for creating GitHub discussions
 */
interface CreateDiscussionConfig extends SafeOutputConfig {
  "title-prefix"?: string;
  "category-id"?: string;
}

/**
 * Configuration for adding comments to issues or PRs
 */
interface AddCommentConfig extends SafeOutputConfig {
  target?: string;
}

/**
 * Configuration for creating pull requests
 */
interface CreatePullRequestConfig extends SafeOutputConfig {
  "title-prefix"?: string;
  labels?: string[];
  draft?: boolean;
  "if-no-changes"?: string;
}

/**
 * Configuration for creating pull request review comments
 */
interface CreatePullRequestReviewCommentConfig extends SafeOutputConfig {
  side?: string;
  target?: string;
}

/**
 * Configuration for creating code scanning alerts
 */
interface CreateCodeScanningAlertConfig extends SafeOutputConfig {
  driver?: string;
}

/**
 * Configuration for adding labels to issues or PRs
 */
interface AddLabelsConfig extends SafeOutputConfig {
  allowed?: string[];
}

/**
 * Configuration for adding issues to milestones
 */
interface AssignMilestoneConfig extends SafeOutputConfig {
  allowed?: string | string[];
  target?: string;
}

/**
 * Configuration for updating issues
 */
interface UpdateIssueConfig extends SafeOutputConfig {
  status?: boolean;
  target?: string;
  title?: boolean;
  body?: boolean;
}

/**
 * Configuration for pushing to pull request branches
 */
interface PushToPullRequestBranchConfig extends SafeOutputConfig {
  target?: string;
  "title-prefix"?: string;
  labels?: string[];
  "if-no-changes"?: string;
}

/**
 * Configuration for uploading assets
 */
interface UploadAssetConfig extends SafeOutputConfig {
  branch?: string;
  "max-size"?: number;
  "allowed-exts"?: string[];
}

/**
 * Configuration for reporting missing tools
 */
interface MissingToolConfig extends SafeOutputConfig {}

/**
 * Configuration for threat detection
 */
interface ThreatDetectionConfig extends SafeOutputConfig {
  enabled?: boolean;
  steps?: any[];
}

// === Safe Job Configuration Interfaces ===

/**
 * Safe job input parameter configuration
 */
interface SafeJobInput {
  description?: string;
  required?: boolean;
  default?: string;
  type?: string;
  options?: string[];
}

/**
 * Safe job configuration item
 */
interface SafeJobConfig {
  name?: string;
  "runs-on"?: any;
  if?: string;
  needs?: string[];
  steps?: any[];
  env?: Record<string, string>;
  permissions?: Record<string, string>;
  inputs?: Record<string, SafeJobInput>;
  "github-token"?: string;
  output?: string;
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
  | AssignMilestoneConfig
  | UpdateIssueConfig
  | PushToPullRequestBranchConfig
  | UploadAssetConfig
  | MissingToolConfig
  | ThreatDetectionConfig;

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
  AssignMilestoneConfig,
  UpdateIssueConfig,
  PushToPullRequestBranchConfig,
  UploadAssetConfig,
  MissingToolConfig,
  ThreatDetectionConfig,
  SpecificSafeOutputConfig,
  // Safe job configuration types
  SafeJobInput,
  SafeJobConfig,
};
