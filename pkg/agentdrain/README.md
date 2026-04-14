# agentdrain Package

The `agentdrain` package implements the [Drain](https://jiemingzhu.github.io/pub/pjhe_icws2017.pdf) log template mining algorithm adapted for analyzing structured agent pipeline events. It is used for anomaly detection in agentic workflow runs.

## Overview

Drain is an online log parsing algorithm that groups log lines into clusters based on token similarity. Each cluster has a *template* — a tokenized log pattern where variable tokens are replaced with a wildcard (`<*>`). When a new log line arrives, Drain finds the most similar existing cluster or creates a new one.

In GitHub Agentic Workflows, `agentdrain` processes `AgentEvent` records emitted by pipeline stages (e.g. `"plan"`, `"tool_call"`, `"finish"`) to:
1. Build a model of normal behavior by training on events from successful runs.
2. Detect anomalies in new runs by comparing events against the learned model.

## Types

### `Config`

Tuning parameters for the Drain miner.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Depth` | `int` | `4` | Parse-tree depth |
| `SimThreshold` | `float64` | `0.4` | Minimum similarity score to match a cluster |
| `MaxChildren` | `int` | `100` | Maximum children per tree node |
| `ParamToken` | `string` | `"<*>"` | Wildcard inserted at variable positions |
| `RareClusterThreshold` | `int` | `2` | Clusters with `Size ≤` this value are flagged as rare |
| `MaskRules` | `[]MaskRule` | (see below) | Regex substitutions applied before tokenization |
| `ExcludeFields` | `[]string` | `["session_id", "trace_id", "span_id", "timestamp"]` | Event fields excluded from flattening |

Use `DefaultConfig()` for production-ready defaults.

### `MaskRule`

A regex-based substitution applied to log lines before tokenization to normalize variable content.

```go
type MaskRule struct {
    Name        string // Human-readable identifier
    Pattern     string // Regular expression
    Replacement string // Substitution string
}
```

Default mask rules normalize UUIDs, session IDs, numeric values, URLs, quoted strings, and timestamps.

### `AgentEvent`

A structured event from an agent pipeline stage.

```go
type AgentEvent struct {
    Stage  string            // e.g. "plan", "tool_call", "finish"
    Fields map[string]string // Key-value pairs from the log line
}
```

### `Cluster`

A group of log lines that share the same template.

```go
type Cluster struct {
    ID       int      // Unique identifier
    Template []string // Tokenized template with wildcards
    Size     int      // Number of lines assigned to this cluster
    Stage    string   // Pipeline stage that generated this cluster
}
```

### `MatchResult`

Returned after processing a log line.

```go
type MatchResult struct {
    ClusterID  int      // Matched or newly created cluster ID
    Template   string   // Space-joined template string
    Params     []string // Actual token values at wildcard positions
    Similarity float64  // Fraction of non-wildcard tokens that matched exactly
    Stage      string   // Pipeline stage of the matched cluster
}
```

### `AnomalyReport`

Describes anomalies detected for a log line.

```go
type AnomalyReport struct {
    IsNewTemplate     bool    // Line created a new cluster
    LowSimilarity     bool    // Best match score was below SimThreshold
    RareCluster       bool    // Matched cluster has been seen ≤ RareClusterThreshold times
    NewClusterCreated bool    // This event produced a brand-new cluster
    AnomalyScore      float64 // Weighted composite score in [0, 1]
    Reason            string  // Human-readable anomaly description
}
```

## Core Components

### `Miner`

The single-stage Drain miner. Processes one pipeline stage at a time.

```go
cfg := agentdrain.DefaultConfig()
miner, err := agentdrain.NewMiner(cfg)

// Training phase — call for known-good events
result, err := miner.TrainEvent(evt)

// Analysis phase — call for events to check
result, report, err := miner.AnalyzeEvent(evt)

// Inspect clusters
clusters := miner.Clusters()
count := miner.ClusterCount()
```

#### Persistence

```go
// Save miner state to JSON
data, err := miner.SaveJSON()

// Restore miner state from JSON
err = miner.LoadJSON(data)
```

### `Coordinator`

Manages a separate `Miner` per pipeline stage, routing events to the correct miner.

```go
stages := []string{"plan", "tool_call", "finish"}
coord, err := agentdrain.NewCoordinator(cfg, stages)

// Load default trained weights
err = coord.LoadDefaultWeights()

// Analyze an event
result, report, err := coord.AnalyzeEvent(evt)

// Access all clusters across all stages
allClusters := coord.AllClusters()

// Save/restore snapshots
snapshots, err := coord.SaveSnapshots()
err = coord.LoadSnapshots(snapshots)

// Save/restore coordinator weights as JSON
data, err := coord.SaveWeightsJSON()
err = coord.LoadWeightsJSON(data)
```

### `AnomalyDetector`

Post-processes `MatchResult` values to produce an `AnomalyReport`.

```go
detector := agentdrain.NewAnomalyDetector(cfg.SimThreshold, cfg.RareClusterThreshold)
report := detector.Analyze(result, isNew, cluster)
```

### `Masker`

Applies `MaskRule` substitutions to log lines before tokenization.

```go
masker, err := agentdrain.NewMasker(cfg.MaskRules)
masked := masker.Mask(rawLine)
```

### Utility Functions

#### `FlattenEvent(evt AgentEvent, excludeFields []string) string`

Converts an `AgentEvent` to a single string for tokenization, omitting fields listed in `excludeFields`. Fields are sorted for deterministic output.

#### `Tokenize(line string) []string`

Splits a log line into tokens on whitespace boundaries.

#### `StageSequence(events []AgentEvent) string`

Returns a comma-separated string of the stages from a slice of events. Useful for summarizing pipeline execution paths.

## Default Weights

The package embeds a set of default trained weights (in `data/`) via `//go:embed`. Call `coord.LoadDefaultWeights()` to initialize the coordinator with pre-trained cluster weights rather than starting cold.

## Design Notes

- The Drain algorithm is O(n·d) per event, where `n` is the number of tokens and `d` is `Depth`.
- `SimThreshold` of `0.4` means at least 40% of tokens must match exactly (excluding wildcards) for a line to join an existing cluster.
- The `Coordinator` routes each `AgentEvent` to its stage-specific `Miner` so that templates from different stages do not interfere.
- `SaveJSON`/`LoadJSON` serialize the parse tree and cluster list to enable persistence across workflow runs.
