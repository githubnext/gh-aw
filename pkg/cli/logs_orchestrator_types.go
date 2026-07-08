// This file provides command-line interface functionality for gh-aw.
// This file (logs_orchestrator_types.go) contains type definitions used by the
// logs orchestrator: option structs for public API functions and internal helper types.

package cli

// LogsDownloadOptions holds parameters for DownloadWorkflowLogs.
type LogsDownloadOptions struct {
	WorkflowName      string
	Count             int
	StartDate         string
	EndDate           string
	OutputDir         string
	Engine            string
	Ref               string
	BeforeRunID       int64
	AfterRunID        int64
	RepoOverride      string
	Verbose           bool
	ToolGraph         bool
	NoStaged          bool
	FirewallOnly      bool
	NoFirewall        bool
	Parse             bool
	JSONOutput        bool
	TimeoutMinutes    int
	SummaryFile       string
	SafeOutputType    string
	FilteredIntegrity bool
	Train             bool
	Format            string
	ArtifactSets      []string
	After             string
	ReportFile        string
}

// StdinLogsOptions holds parameters for DownloadWorkflowLogsFromStdin.
type StdinLogsOptions struct {
	RunURLs           []string
	OutputDir         string
	Engine            string
	RepoOverride      string
	Verbose           bool
	ToolGraph         bool
	NoStaged          bool
	FirewallOnly      bool
	NoFirewall        bool
	Parse             bool
	JSONOutput        bool
	Timeout           int
	SummaryFile       string
	SafeOutputType    string
	FilteredIntegrity bool
	Train             bool
	Format            string
	ReportFile        string
	// ArtifactSets defaults to nil (download all artifacts) when this API is used
	// programmatically. The CLI passes ["usage"] to match the logs command default.
	ArtifactSets []string
}

// continuationOptions carries the parameters needed by buildContinuationIfNeeded
// to produce a ContinuationData cursor for paginated log fetches.
type continuationOptions struct {
	workflowName   string
	startDate      string
	endDate        string
	engine         string
	branch         string
	afterRunID     int64
	count          int
	timeoutMinutes int
}

// renderLogsOutputOptions holds configuration for renderLogsOutput.
type renderLogsOutputOptions struct {
	outputDir      string
	summaryFile    string
	format         string
	reportFile     string
	jsonOutput     bool
	toolGraph      bool
	train          bool
	continuation   *ContinuationData
	verbose        bool
	artifactFilter []string
}
