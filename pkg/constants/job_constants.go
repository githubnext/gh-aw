package constants

// JobName represents a GitHub Actions job identifier.
// This semantic type distinguishes job names from arbitrary strings,
// preventing mixing of job identifiers with other string types.
//
// Example usage:
//
//	const AgentJobName JobName = "agent"
//	func GetJob(name JobName) (*Job, error) { ... }
type JobName string

// String returns the string representation of the job name
func (j JobName) String() string {
	return string(j)
}

// IsValid returns true if the job name is non-empty
func (j JobName) IsValid() bool {
	return len(j) > 0
}

// StepID represents a GitHub Actions step identifier.
// This semantic type distinguishes step IDs from arbitrary strings,
// preventing mixing of step identifiers with job names or other strings.
//
// Example usage:
//
//	const CheckMembershipStepID StepID = "check_membership"
//	func GetStep(id StepID) (*Step, error) { ... }
type StepID string

// String returns the string representation of the step ID
func (s StepID) String() string {
	return string(s)
}

// IsValid returns true if the step ID is non-empty
func (s StepID) IsValid() bool {
	return len(s) > 0
}

// MCPServerID represents a built-in MCP server identifier.
// This semantic type distinguishes MCP server IDs from arbitrary strings,
// preventing accidental mixing of server identifiers with other string types.
//
// Example usage:
//
//	const SafeOutputsMCPServerID MCPServerID = "safeoutputs"
//	func GetServer(id MCPServerID) (*Server, error) { ... }
type MCPServerID string

// String returns the string representation of the MCP server ID
func (m MCPServerID) String() string {
	return string(m)
}

// Job name constants for GitHub Actions workflow jobs
const AgentJobName JobName = "agent"
const ActivationJobName JobName = "activation"
const PreActivationJobName JobName = "pre_activation"
const DetectionJobName JobName = "detection"
const SafeOutputsJobName JobName = "safe_outputs"
const UploadAssetsJobName JobName = "upload_assets"
const UploadCodeScanningJobName JobName = "upload_code_scanning_sarif"
const ConclusionJobName JobName = "conclusion"
const UnlockJobName JobName = "unlock"

// Artifact name constants
const SafeOutputArtifactName = "safe-output"
const AgentOutputArtifactName = "agent-output"

// AgentArtifactName is the name of the unified agent artifact that contains all agent job outputs,
// including safe outputs, agent output, engine logs, and other agent-related files.
const AgentArtifactName = "agent"

// DetectionArtifactName is the artifact name for the threat detection log.
const DetectionArtifactName = "detection"

// LegacyDetectionArtifactName is the old artifact name used before the rename.
// Kept for backward compatibility when downloading artifacts from older workflow runs.
const LegacyDetectionArtifactName = "threat-detection.log"

// AgentOutputFilename is the filename of the agent output JSON file
const AgentOutputFilename = "agent_output.json"

// SafeOutputsFilename is the filename of the raw safe outputs NDJSON file copied to /tmp/gh-aw/
const SafeOutputsFilename = "safeoutputs.jsonl"

// TokenUsageFilename is the filename of the aggregated token usage JSON file written to /tmp/gh-aw/
// by parse_token_usage.cjs. It is included in the agent artifact so third-party tools can
// consume structured token data without parsing the step summary or GITHUB_OUTPUT.
const TokenUsageFilename = "agent_usage.json"

// GithubRateLimitsFilename is the filename of the GitHub API rate-limit log written to /tmp/gh-aw/.
// Each line is a JSON object recording the x-ratelimit-* headers (or rate-limit API snapshot)
// captured during github.rest API calls, enabling post-run analysis of rate-limit consumption.
const GithubRateLimitsFilename = "github_rate_limits.jsonl"

// OtelJsonlFilename is the filename of the OTLP span mirror written to /tmp/gh-aw/
// by send_otlp_span.cjs. Each line is a full OTLP/HTTP JSON traces payload.
// Included in the agent artifact so spans are available without a live collector.
const OtelJsonlFilename = "otel.jsonl"

// ArtifactPrefixOutputName is the job output name that exposes the artifact name prefix.
// In workflow_call context, the prefix is a stable hash derived from the workflow inputs,
// ensuring artifact names are unique when the same workflow is called multiple times in
// the same workflow run (e.g. multiple jobs each calling the same reusable workflow).
// Empty string in non-workflow_call context.
const ArtifactPrefixOutputName = "artifact_prefix"

// ActivationArtifactName is the artifact name for the activation job output
// (aw_info.json and prompt.txt).
const ActivationArtifactName = "activation"

// SafeOutputItemsArtifactName is the artifact name for the safe output items manifest.
// This artifact contains the JSONL manifest of all items created by safe output handlers
// and is uploaded by the safe_outputs job to avoid conflicting with the "agent" artifact
// that is already uploaded by the agent job.
const SafeOutputItemsArtifactName = "safe-outputs-items"

// SarifArtifactName is the artifact name used to transfer the SARIF file generated by
// the create_code_scanning_alert handler from the safe_outputs job to the
// upload_code_scanning_sarif job.  The safe_outputs job uploads the file under this name;
// the upload job downloads it and passes it to github/codeql-action/upload-sarif.
const SarifArtifactName = "code-scanning-sarif"

// SarifArtifactDownloadPath is the local path where the upload_code_scanning_sarif job
// downloads the SARIF artifact.  The file will be available at this path + the SARIF
// filename ("code-scanning-alert.sarif") after actions/download-artifact completes.
const SarifArtifactDownloadPath = "/tmp/gh-aw/sarif/"

// SarifFileName is the name of the SARIF file generated by create_code_scanning_alert.cjs
// and uploaded / downloaded as part of the code-scanning-sarif artifact.
const SarifFileName = "code-scanning-alert.sarif"

// MCP server ID constants
const SafeOutputsMCPServerID MCPServerID = "safeoutputs"

// MCPScriptsMCPServerID is the identifier for the mcp-scripts MCP server
const MCPScriptsMCPServerID MCPServerID = "mcpscripts"

// MCPScriptsMCPVersion is the version of the mcp-scripts MCP server
const MCPScriptsMCPVersion = "1.0.0"

// AgenticWorkflowsMCPServerID is the identifier for the agentic-workflows MCP server
const AgenticWorkflowsMCPServerID MCPServerID = "agenticworkflows"

// Step IDs for pre-activation job
const CheckMembershipStepID StepID = "check_membership"
const CheckStopTimeStepID StepID = "check_stop_time"
const CheckSkipIfMatchStepID StepID = "check_skip_if_match"
const CheckSkipIfNoMatchStepID StepID = "check_skip_if_no_match"
const CheckCommandPositionStepID StepID = "check_command_position"
const RemoveTriggerLabelStepID StepID = "remove_trigger_label"
const GetTriggerLabelStepID StepID = "get_trigger_label"
const CheckRateLimitStepID StepID = "check_rate_limit"
const CheckSkipRolesStepID StepID = "check_skip_roles"
const CheckSkipBotsStepID StepID = "check_skip_bots"
const CheckSkipIfCheckFailingStepID StepID = "check_skip_if_check_failing"

// PreActivationAppTokenStepID is the step ID for the unified GitHub App token mint step
// emitted in the pre-activation job when on.github-app is configured alongside skip-if checks.
const PreActivationAppTokenStepID StepID = "pre-activation-app-token"

// ParseMCPGatewayStepID is the step ID for the MCP gateway log parsing step in the agent job.
// Its effective_tokens output is exposed as an agent job output so that the safe_outputs job
// can pass the value as GH_AW_EFFECTIVE_TOKENS to the footer template renderer.
const ParseMCPGatewayStepID StepID = "parse-mcp-gateway"

// Output names for pre-activation job steps
const IsTeamMemberOutput = "is_team_member"
const StopTimeOkOutput = "stop_time_ok"
const SkipCheckOkOutput = "skip_check_ok"
const SkipNoMatchCheckOkOutput = "skip_no_match_check_ok"
const CommandPositionOkOutput = "command_position_ok"
const MatchedCommandOutput = "matched_command"
const RateLimitOkOutput = "rate_limit_ok"
const SkipRolesOkOutput = "skip_roles_ok"
const SkipBotsOkOutput = "skip_bots_ok"
const SkipIfCheckFailingOkOutput = "skip_if_check_failing_ok"
const ActivatedOutput = "activated"

// Rate limit defaults
const DefaultRateLimitMax = 5     // Default maximum runs per time window
const DefaultRateLimitWindow = 60 // Default time window in minutes (1 hour)
