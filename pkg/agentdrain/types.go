package agentdrain

// Config holds tuning parameters for the Drain log template miner.
type Config struct {
	// Depth controls how many levels of the parse tree are used.
	Depth int
	// SimThreshold is the minimum similarity score (0–1) required to match an existing cluster.
	SimThreshold float64
	// MaxChildren limits the number of children per internal tree node.
	MaxChildren int
	// ParamToken is the wildcard string inserted where tokens differ across log lines.
	ParamToken string
	// RareClusterThreshold marks clusters with size ≤ this value as rare.
	RareClusterThreshold int
	// MaskRules are applied before tokenization to normalize variable parts of log lines.
	MaskRules []MaskRule
	// ExcludeFields lists AgentEvent field keys that are omitted when flattening events.
	ExcludeFields []string
}

// MaskRule describes a regex substitution applied to log lines before processing.
type MaskRule struct {
	// Name is a human-readable identifier for the rule.
	Name string
	// Pattern is the regular expression to match.
	Pattern string
	// Replacement is the string substituted for each match.
	Replacement string
}

// Cluster represents a group of log lines that share the same template.
type Cluster struct {
	// ID is the unique cluster identifier.
	ID int
	// Template is the tokenized log template with wildcards at variable positions.
	Template []string
	// Size is the number of log lines that have been assigned to this cluster.
	Size int
	// Stage identifies which agent stage generated this cluster.
	Stage string
}

// MatchResult is returned after processing a log line through the miner.
type MatchResult struct {
	// ClusterID is the ID of the matched or newly created cluster.
	ClusterID int
	// Template is the space-joined template string.
	Template string
	// Params holds the actual token values at wildcard positions.
	Params []string
	// Similarity is the fraction of non-wildcard positions that matched exactly.
	Similarity float64
	// Stage is the agent stage associated with the matched cluster.
	Stage string
}

// AnomalyReport describes anomalies detected for a log line.
type AnomalyReport struct {
	// IsNewTemplate is true when the log line created a new cluster.
	IsNewTemplate bool
	// LowSimilarity is true when the best match score was below the configured threshold.
	LowSimilarity bool
	// RareCluster is true when the matched cluster has been seen fewer times than the rare threshold.
	RareCluster bool
	// NewClusterCreated is true when this event produced a brand-new cluster.
	NewClusterCreated bool
	// AnomalyScore is a weighted composite score in the range [0, 1].
	AnomalyScore float64
	// Reason is a human-readable description of all anomalies that were detected.
	Reason string
}

// AgentEvent is a structured log event emitted by an agent pipeline stage.
type AgentEvent struct {
	// Stage identifies the pipeline stage (e.g., "plan", "tool_call", "finish").
	Stage string
	// Fields contains the key-value pairs parsed from the log line.
	Fields map[string]string
}
