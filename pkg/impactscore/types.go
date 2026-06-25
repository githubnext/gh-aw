package impactscore

// DimensionStateReason is the standard categorical dimension for normalized
// issue close reasons and pull request lifecycle outcomes.
const DimensionStateReason = "state_reason"

// WorkItem is the normalized issue or pull request record used by the ranker.
// Fetching and API caching are deliberately outside this package.
type WorkItem struct {
	Repo   string
	Number int
	Type   string
	Title  string
	State  string

	// StateReason carries lifecycle detail beyond the coarse GitHub state, such
	// as not_planned, completed, draft, merged, or closed_unmerged.
	StateReason string

	// Dimensions and Measures are repo-defined extension points. Use Dimensions
	// for categorical values and Measures for numeric signals consumed by impact
	// policy evaluation.
	Dimensions map[string][]string
	Measures   map[string]float64

	// GraphNodes and GraphEdges let repo adapters add custom graph structure that
	// does not fit the built-in fields. A GraphEdge with an empty Source is treated
	// as an edge from this work item.
	GraphNodes []Node
	GraphEdges []Edge

	Labels               []string
	Components           []string
	Areas                []string
	Signals              []string
	ContextSignals       []string
	ContextEvidence      []string
	SourceWorkflows      []string
	SourceWorkflowPaths  []string
	SourceWorkflowRuns   []string
	ArchitectureSpecs    []string
	ArchitectureConcepts []string

	ReleaseCategory       string
	ChangeType            string
	ReleaseNoteImportance float64
	Released              bool

	ChangedFiles            int
	SensitivePathCount      int
	ComponentCount          int
	CentralityWeightedTouch float64
	HotspotWeightedTouch    float64
}

// WorkflowDefinition describes an agentic workflow known to the repository.
type WorkflowDefinition struct {
	Name        string
	Aliases     []string
	Path        string
	SourcePath  string
	TitlePrefix string
	Triggers    []string
	Tools       []string
	Outputs     []string
}

// Node is a typed knowledge graph node.
type Node struct {
	ID      string
	Type    string
	Name    string
	Title   string
	Number  int
	State   string
	Item    string
	Metrics map[string]float64
}

// Edge is a typed knowledge graph edge with optional evidence text.
type Edge struct {
	Source   string
	Target   string
	Type     string
	Weight   float64
	Evidence string
}

// Graph is an in-memory typed knowledge graph optimized for ranking passes.
type Graph struct {
	Nodes map[string]Node
	Edges []Edge
	Out   map[string][]Edge
	In    map[string][]Edge
	Items map[string]WorkItem
}

// ItemFeatures is the explorable measurement surface for one work item.
// Dimensions are categorical values and Measures are numeric signals available
// to the configured scorer.
type ItemFeatures struct {
	Item       WorkItem
	ItemID     string
	Dimensions map[string][]string
	Measures   map[string]float64
}

// ItemScore is the result produced by impact scoring.
type ItemScore struct {
	Score       float64
	Prior       float64
	Confidence  float64
	Support     int
	Source      string
	Explanation ScoreExplanation
}

// ScoreExplanation records why a deterministic score was produced. Policy
// fields are populated when a repository policy contributes the score.
type ScoreExplanation struct {
	PolicyPath    string   `json:"policy_path,omitempty"`
	PolicyVersion int      `json:"policy_version,omitempty"`
	PolicySHA256  string   `json:"policy_sha256,omitempty"`
	MatchedRules  []string `json:"matched_rules,omitempty"`
}

// ScoreFunc computes the final item score from extracted graph features.
type ScoreFunc func(features ItemFeatures) ItemScore

// RankOptions configures how a repository turns graph features into impact. The
// zero value uses DefaultRankOptions.
type RankOptions struct {
	DisabledMeasures []string
	Score            ScoreFunc
}

// ItemRank is a work item scored by a repository impact policy.
type ItemRank struct {
	Number           int
	ItemType         string
	State            string
	StateReason      string
	Title            string
	Released         bool
	ScoreSource      string
	ScoreExplanation ScoreExplanation `json:"score_explanation,omitzero"`
	ImpactScore      float64
	SourceWorkflows  []string
}

// WorkflowCostRun is a normalized workflow-run cost observation.
type WorkflowCostRun struct {
	Workflow      string
	RunID         string
	RunURL        string
	AICCost       float64
	TokenUsage    float64
	Turns         float64
	ActionMinutes float64
	Errors        float64
	Source        string
}

// WorkflowRank is the impact-score/cost ranking for a workflow.
type WorkflowRank struct {
	Workflow              string
	AttributedImpactScore float64
	LinkedItems           int
	OpenItems             int
	ReleasedItems         int
	RunCount              int
	CostedRunCount        int
	TotalAICCost          float64
	AverageAICCostPerRun  float64
	TotalTokens           float64
	TotalTurns            float64
	ActionMinutes         float64
	Errors                float64
	ImpactPerAIC          float64
	ImpactPerThousandAIC  float64
	AICPerImpact          float64
	ActionZone            string
	CostSources           []string
}
