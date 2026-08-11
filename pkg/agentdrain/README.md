# agentdrain Package

> Drain-style log template mining and anomaly scoring for structured agent pipeline events.

## Overview

The `agentdrain` package implements an online log-template miner inspired by the Drain algorithm and adapts it to `AgentEvent` records emitted by agentic workflow stages. It converts structured events into deterministic token streams, normalizes variable values with regex-based masking, groups similar events into clusters, and returns a `MatchResult` that captures the matched template, extracted parameters, and similarity score.

The package is designed for two related tasks: training on known-good runs and anomaly analysis of new runs. `Miner` handles a single stream of events, while `Coordinator` manages one `Miner` per stage so templates from `plan`, `tool_call`, `finish`, and other stages do not interfere with each other. Persisted snapshots and embedded default weights allow models to be reused across runs instead of starting from an empty state every time.

## Public API

### Types

| Type | Kind | Description |
|------|------|-------------|
| `AgentEvent` | struct | Structured event with a stage name and key/value fields to flatten, mask, and mine. |
| `AnomalyDetector` | struct | Scores `MatchResult` values against similarity and rarity thresholds. |
| `AnomalyReport` | struct | Summarizes anomaly flags, normalized score, and human-readable reason text. |
| `Cluster` | struct | Template cluster with ID, tokenized template, observation count, and optional stage. |
| `Config` | struct | Tuning parameters for masking, parse-tree depth, similarity threshold, and excluded fields. |
| `Coordinator` | struct | Routes events to one `Miner` per stage and persists combined weights. |
| `MaskRule` | struct | Regex substitution rule applied before tokenization. |
| `Masker` | struct | Compiled sequence of `MaskRule` values applied in order. |
| `MatchResult` | struct | Result of matching or creating a cluster, including template, params, and similarity. |
| `Miner` | struct | Concurrent single-stream Drain-style miner with training and analysis methods. |
| `Snapshot` | struct | Serializable miner state used by `SaveJSON` and `LoadJSON`. |
| `SnapshotCluster` | struct | Serializable form of a single `Cluster` within a `Snapshot`. |

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `(*Coordinator).AllClusters` | `func (c *Coordinator) AllClusters() map[string][]Cluster` | Returns a stage-to-cluster snapshot for every managed miner. |
| `(*Coordinator).AnalyzeEvent` | `func (c *Coordinator) AnalyzeEvent(evt AgentEvent) (*MatchResult, *AnomalyReport, error)` | Routes an event to its stage's miner, analyzes it, then trains on it, returning both the match and the anomaly report. |
| `(*Coordinator).LoadDefaultWeights` | `func (c *Coordinator) LoadDefaultWeights() error` | Loads the embedded default trained weights into all stage miners. |
| `(*Coordinator).LoadSnapshots` | `func (c *Coordinator) LoadSnapshots(data map[string][]byte) error` | Restores stage miners from per-stage JSON snapshots, creating new miners for stages not in the original constructor input. |
| `(*Coordinator).LoadWeightsJSON` | `func (c *Coordinator) LoadWeightsJSON(data []byte) error` | Restores all stage miners from a combined JSON blob produced by `SaveWeightsJSON`. |
| `(*Coordinator).SaveSnapshots` | `func (c *Coordinator) SaveSnapshots() (map[string][]byte, error)` | Serializes each stage miner to per-stage JSON snapshots. |
| `(*Coordinator).SaveWeightsJSON` | `func (c *Coordinator) SaveWeightsJSON() ([]byte, error)` | Serializes all stage snapshots into one combined JSON document. |
| `(*Coordinator).TrainEvent` | `func (c *Coordinator) TrainEvent(evt AgentEvent) (*MatchResult, error)` | Routes an event to its stage's miner and trains on it, creating the stage miner on demand if needed. |
| `(*AnomalyDetector).Analyze` | `func (d *AnomalyDetector) Analyze(result *MatchResult, isNew bool, cluster *Cluster) *AnomalyReport` | Produces an anomaly report for a match result and cluster context. |
| `(*Masker).Mask` | `func (m *Masker) Mask(line string) string` | Applies all configured mask rules and returns the normalized line. |
| `(*Miner).AnalyzeEvent` | `func (m *Miner) AnalyzeEvent(evt AgentEvent) (*MatchResult, *AnomalyReport, error)` | Flattens and analyzes an event without training, returning the would-be match and an anomaly report. |
| `(*Miner).Clusters` | `func (m *Miner) Clusters() []Cluster` | Returns a safe snapshot of all known clusters in the miner. |
| `(*Miner).LoadJSON` | `func (m *Miner) LoadJSON(data []byte) error` | Restores miner state (clusters and parse tree) from a JSON snapshot produced by `SaveJSON`. |
| `(*Miner).SaveJSON` | `func (m *Miner) SaveJSON() ([]byte, error)` | Serializes the miner's clusters into a JSON `Snapshot`. |
| `(*Miner).Train` | `func (m *Miner) Train(line string) (*MatchResult, error)` | Trains the miner on a raw line and returns the resulting match. |
| `(*Miner).TrainEvent` | `func (m *Miner) TrainEvent(evt AgentEvent) (*MatchResult, error)` | Flattens an `AgentEvent` and trains the miner on the resulting line. |
| `DefaultConfig` | `func DefaultConfig() Config` | Returns the production default miner configuration and default masking rules. |
| `FlattenEvent` | `func FlattenEvent(evt AgentEvent, excludeFields []string) string` | Converts an event into deterministic `key=value` tokens with stage first and excluded fields omitted. |
| `NewAnomalyDetector` | `func NewAnomalyDetector(simThreshold float64, rareClusterThreshold int) (*AnomalyDetector, error)` | Validates thresholds and constructs an anomaly detector. |
| `NewCoordinator` | `func NewCoordinator(cfg Config, stages []string) (*Coordinator, error)` | Creates one stage-scoped miner for each supplied stage. |
| `NewMasker` | `func NewMasker(rules []MaskRule) (*Masker, error)` | Compiles masking regexes into a reusable masker. |
| `NewMiner` | `func NewMiner(cfg Config) (*Miner, error)` | Creates a miner with compiled mask rules, empty clusters, and a fresh parse tree. |
| `StageSequence` | `func StageSequence(events []AgentEvent) string` | Returns a space-separated sequence of event stages. |
| `Tokenize` | `func Tokenize(line string) []string` | Splits a masked line on whitespace boundaries. |

### Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `AnomalyMaxScore` | untyped `float64` | `2.0` | Maximum raw anomaly score before normalization to `[0,1]`. |
| `AnomalyWeightLow` | untyped `float64` | `0.7` | Weight applied when a known template matches below the configured similarity threshold. |
| `AnomalyWeightNew` | untyped `float64` | `1.0` | Weight applied when analysis creates a brand-new cluster. |
| `AnomalyWeightRare` | untyped `float64` | `0.3` | Weight applied when the matched cluster size is at or below the rare-cluster threshold. |

## Usage Examples

```go
cfg := agentdrain.DefaultConfig()
miner, err := agentdrain.NewMiner(cfg)
if err != nil {
	panic(err)
}

evt := agentdrain.AgentEvent{
	Stage:  "plan",
	Fields: map[string]string{"action": "start", "step": "1"},
}
result, err := miner.TrainEvent(evt)
if err != nil {
	panic(err)
}
fmt.Println(result.ClusterID)
```

```go
cfg := agentdrain.DefaultConfig()
stages := []string{"plan", "tool_call", "finish"}
coord, err := agentdrain.NewCoordinator(cfg, stages)
if err != nil {
	panic(err)
}
if err := coord.LoadDefaultWeights(); err != nil {
	panic(err)
}

evt := agentdrain.AgentEvent{
	Stage:  "tool_call",
	Fields: map[string]string{"tool": "bash", "status": "ok"},
}
result, report, err := coord.AnalyzeEvent(evt)
if err != nil {
	panic(err)
}
fmt.Println(result.Template, report.AnomalyScore)
```

```go
flat := agentdrain.FlattenEvent(
	agentdrain.AgentEvent{
		Stage: "tool_call",
		Fields: map[string]string{
			"tool":       "search",
			"query":      "foo",
			"session_id": "abc123",
			"latency_ms": "42",
		},
	},
	[]string{"session_id"},
)
// flat == "stage=tool_call latency_ms=42 query=foo tool=search"
```

## Design Decisions

`FlattenEvent` MUST emit deterministic output: the `stage=` token is first when present, remaining keys are sorted alphabetically, and excluded fields are omitted. This keeps clustering stable across map iteration order and allows persisted weights to be reused reliably.

`AnalyzeEvent` performs inference before updating training state, then trains on the same event and scores the result against the matched or created cluster. New-template anomalies and low-similarity anomalies are intentionally mutually exclusive: a brand-new cluster is already anomalous without also being labeled low similarity.

`Coordinator` SHOULD be used when events belong to semantically different stages. Each stage receives its own miner so templates from unrelated phases do not merge into the same cluster space. `LoadSnapshots` MAY create new stage miners when snapshots contain stages that were not part of the original constructor input.

The package embeds default trained weights in `data/default_weights.json`. Callers MAY use `LoadDefaultWeights` to start from a pre-trained baseline instead of training from scratch.

## Dependencies

Internal dependencies include `pkg/logger` for debug logging, `pkg/setutil` for exclusion-set membership, and `pkg/sliceutil` for slice and map helpers. External dependencies are limited to the Go standard library.

## Thread Safety

`Miner` and `Coordinator` are safe for concurrent use. `Miner` protects mutable state with an internal `sync.RWMutex`; training, analysis, and load paths acquire write locks, while cluster snapshots and persistence reads acquire read locks. `Coordinator` protects its stage-to-miner map with its own `sync.RWMutex` and delegates per-stage concurrency to each `Miner`.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
