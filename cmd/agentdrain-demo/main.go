// cmd/agentdrain-demo is a runnable demonstration of the pkg/agentdrain package.
//
// It shows:
//   - building a coordinator with stage-specific miners
//   - pretraining known templates
//   - ingesting sample agent-session events
//   - printing matched cluster ID, template, extracted params, and anomaly report
//   - saving and reloading a snapshot
//   - running inference on a new event after restore
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/agentdrain"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "demo error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// -----------------------------------------------------------------------
	// 1. Build coordinator with stage-specific miners.
	// -----------------------------------------------------------------------
	cfg := agentdrain.DefaultConfig()
	stages := []string{"plan", "tool_call", "tool_result", "retry", "error", "finish"}
	coord, err := agentdrain.NewCoordinator(cfg, stages)
	if err != nil {
		return fmt.Errorf("create coordinator: %w", err)
	}
	fmt.Println("=== agentdrain demo ===")
	fmt.Printf("Coordinator created with %d stage miners\n\n", len(stages))

	// -----------------------------------------------------------------------
	// 2. Pretrain a few known templates.
	// -----------------------------------------------------------------------
	planMiner, _ := coord.MinerForStage("plan")
	planMiner.PreTrainTemplates([]string{
		"stage=plan action=decompose objective=<*>",
		"stage=plan action=synthesize objective=<*>",
	})

	toolMiner, _ := coord.MinerForStage("tool_call")
	toolMiner.PreTrainTemplate("stage=tool_call tool=search query=<*> latency_ms=<NUM>", 10)

	errorMiner, _ := coord.MinerForStage("error")
	errorMiner.PreTrainTemplate("stage=error type=<*> code=<*> message=<*>", 5)

	fmt.Println("Pre-trained templates:")
	for _, stage := range stages {
		m, _ := coord.MinerForStage(stage)
		if m.ClusterCount() > 0 {
			fmt.Printf("  stage=%-12s  clusters=%d\n", stage, m.ClusterCount())
		}
	}
	fmt.Println()

	// -----------------------------------------------------------------------
	// 3. Ingest sample agent-session events.
	// -----------------------------------------------------------------------
	sampleEvents := []agentdrain.AgentEvent{
		{Stage: "plan", Fields: map[string]string{"action": "decompose", "objective": "summarise paper"}},
		{Stage: "tool_call", Fields: map[string]string{"tool": "search", "query": "drain3 golang", "latency_ms": "123"}},
		{Stage: "tool_result", Fields: map[string]string{"tool": "search", "status": "ok", "hits": "5"}},
		{Stage: "tool_call", Fields: map[string]string{"tool": "search", "query": "log template mining", "latency_ms": "87"}},
		{Stage: "tool_result", Fields: map[string]string{"tool": "search", "status": "ok", "hits": "12"}},
		{Stage: "retry", Fields: map[string]string{"attempt": "2", "reason": "timeout"}},
		{Stage: "error", Fields: map[string]string{"type": "http", "code": "503", "message": "upstream timeout"}},
		{Stage: "finish", Fields: map[string]string{"status": "success", "tokens_in": "1200", "tokens_out": "340"}},
		// A second session with the same patterns.
		{Stage: "plan", Fields: map[string]string{"action": "decompose", "objective": "review PR"}},
		{Stage: "tool_call", Fields: map[string]string{"tool": "search", "query": "golang concurrency", "latency_ms": "200"}},
		{Stage: "finish", Fields: map[string]string{"status": "success", "tokens_in": "800", "tokens_out": "220"}},
		// An unusual event (should trigger anomaly).
		{Stage: "error", Fields: map[string]string{"type": "auth", "code": "403", "message": "forbidden"}},
	}

	fmt.Println("Ingesting events:")
	fmt.Println(dashes(72))
	for _, evt := range sampleEvents {
		result, report, err := coord.AnalyzeEvent(evt)
		if err != nil {
			fmt.Printf("  [SKIP] stage=%-12s  error=%v\n", evt.Stage, err)
			continue
		}
		printEventResult(evt, result, report)
	}
	fmt.Println()

	// -----------------------------------------------------------------------
	// 4. Print cluster summary.
	// -----------------------------------------------------------------------
	fmt.Println("Cluster summary after ingestion:")
	fmt.Println(dashes(72))
	allClusters := coord.AllClusters()
	for _, stage := range stages {
		clusters := allClusters[stage]
		if len(clusters) == 0 {
			continue
		}
		fmt.Printf("  stage=%-12s  clusters=%d\n", stage, len(clusters))
		for _, c := range clusters {
			fmt.Printf("    [id=%d size=%d] %s\n", c.ID, c.Size, printTemplate(c.Template))
		}
	}
	fmt.Println()

	// -----------------------------------------------------------------------
	// 5. Stage sequence helper.
	// -----------------------------------------------------------------------
	seq := agentdrain.StageSequence(sampleEvents)
	fmt.Printf("Stage sequence: %s\n\n", seq)

	// -----------------------------------------------------------------------
	// 6. Save snapshot.
	// -----------------------------------------------------------------------
	snapshots, err := coord.SaveSnapshots()
	if err != nil {
		return fmt.Errorf("save snapshots: %w", err)
	}
	totalBytes := 0
	for _, b := range snapshots {
		totalBytes += len(b)
	}
	fmt.Printf("Saved %d stage snapshots (%d bytes total)\n\n", len(snapshots), totalBytes)

	// -----------------------------------------------------------------------
	// 7. Reload snapshot into a fresh coordinator.
	// -----------------------------------------------------------------------
	coord2, err := agentdrain.NewCoordinator(cfg, stages)
	if err != nil {
		return fmt.Errorf("create coord2: %w", err)
	}
	if err := coord2.LoadSnapshots(snapshots); err != nil {
		return fmt.Errorf("load snapshots: %w", err)
	}
	fmt.Println("Reloaded coordinator from snapshot.")
	allClusters2 := coord2.AllClusters()
	restoredClusters := 0
	for _, cs := range allClusters2 {
		restoredClusters += len(cs)
	}
	fmt.Printf("Restored %d clusters\n\n", restoredClusters)

	// -----------------------------------------------------------------------
	// 8. Inference-only match on a new event after restore.
	// -----------------------------------------------------------------------
	newEvent := agentdrain.AgentEvent{
		Stage:  "error",
		Fields: map[string]string{"type": "http", "code": "500", "message": "internal server error"},
	}
	fmt.Println("Inference on new event after restore:")
	fmt.Println(dashes(72))
	result2, report2, err := coord2.AnalyzeEvent(newEvent)
	if err != nil {
		fmt.Printf("  AnalyzeEvent error: %v\n", err)
	} else {
		printEventResult(newEvent, result2, report2)
	}
	fmt.Println()

	// -----------------------------------------------------------------------
	// 9. Print a snapshot excerpt as JSON for inspection.
	// -----------------------------------------------------------------------
	if data, ok := snapshots["error"]; ok {
		var snap map[string]any
		_ = json.Unmarshal(data, &snap)
		pretty, _ := json.MarshalIndent(snap, "  ", "  ")
		fmt.Println("Error-stage snapshot (excerpt):")
		fmt.Println(dashes(72))
		fmt.Println(" ", string(pretty))
	}

	return nil
}

func printEventResult(evt agentdrain.AgentEvent, result *agentdrain.MatchResult, report *agentdrain.AnomalyReport) {
	flat := agentdrain.FlattenEvent(evt, nil)
	fmt.Printf("  event: %s\n", flat)
	fmt.Printf("         cluster=%d  sim=%.2f  template=%q\n",
		result.ClusterID, result.Similarity, result.Template)
	if len(result.Params) > 0 {
		fmt.Printf("         params=%v\n", result.Params)
	}
	if report != nil && report.AnomalyScore > 0 {
		fmt.Printf("         anomaly: score=%.2f  new=%v  rare=%v  lowSim=%v  reason=%q\n",
			report.AnomalyScore, report.IsNewTemplate, report.RareCluster, report.LowSimilarity, report.Reason)
	}
}

func printTemplate(tokens []string) string {
	return strings.Join(tokens, " ")
}

func dashes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = '-'
	}
	return string(b)
}
