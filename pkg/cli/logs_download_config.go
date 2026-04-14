// This file provides command-line interface functionality for gh-aw.
// This file (logs_download_config.go) defines the DownloadConfig struct
// used to pass parameters to DownloadWorkflowLogs.

package cli

// DownloadConfig holds all parameters for DownloadWorkflowLogs,
// replacing the previous 25-parameter function signature.
type DownloadConfig struct {
	// WorkflowName filters to a specific workflow by name (or file path).
	// Leave empty to scan all agentic workflows.
	WorkflowName string

	// Count is the maximum number of workflow runs to return.
	Count int

	// StartDate is the earliest created_at timestamp (RFC 3339 / date string).
	StartDate string

	// EndDate is the latest created_at timestamp (RFC 3339 / date string).
	EndDate string

	// OutputDir is the local directory where artifacts are downloaded.
	OutputDir string

	// Engine filters runs by AI engine name (e.g. "copilot", "claude").
	Engine string

	// Ref filters runs by Git ref (branch or tag).
	Ref string

	// BeforeRunID restricts results to runs with ID less than this value.
	BeforeRunID int64

	// AfterRunID restricts results to runs with ID greater than this value.
	AfterRunID int64

	// RepoOverride overrides the target repository (owner/repo).
	RepoOverride string

	// Verbose enables verbose diagnostic output to stderr.
	Verbose bool

	// ToolGraph enables tool dependency graph rendering.
	ToolGraph bool

	// NoStaged skips staged runs (runs created by the staging system).
	NoStaged bool

	// FirewallOnly restricts output to runs that contain firewall data.
	FirewallOnly bool

	// NoFirewall excludes runs that contain firewall data.
	NoFirewall bool

	// Parse enables structured parsing of log content.
	Parse bool

	// JSONOutput emits output as JSON to stdout instead of console tables.
	JSONOutput bool

	// Timeout is the maximum run time in minutes (0 = unlimited).
	Timeout int

	// SummaryFile is the path to write a JSON summary file.
	SummaryFile string

	// SafeOutputType filters runs to those containing a specific safe-output type.
	SafeOutputType string

	// FilteredIntegrity restricts output to runs with DIFC-filtered events.
	FilteredIntegrity bool

	// Train enables training-data export mode.
	Train bool

	// Format selects the output format for console rendering.
	Format string

	// ArtifactSets is an optional list of artifact set names to download.
	ArtifactSets []string
}
