package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/agentdrain"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var drain3TrainLog = logger.New("cli:drain3_train")

// drain3WeightsFilename is the output filename for the trained weights.
const drain3WeightsFilename = "drain3_weights.json"

// TrainDrain3Weights trains a Drain3 coordinator across all processed runs,
// serialises the resulting weights to drain3_weights.json in outputDir, and
// prints instructions on how to embed the file as default weights.
//
// This function is invoked when the user passes --train to the logs command.
func TrainDrain3Weights(processedRuns []ProcessedRun, outputDir string, verbose bool) error {
	if len(processedRuns) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No processed runs available for log pattern training"))
		return nil
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Training log pattern weights from %d run(s)...", len(processedRuns))))

	cfg := agentdrain.DefaultConfig()
	coordinator, err := agentdrain.NewCoordinator(cfg, defaultAgentDrainStages)
	if err != nil {
		return fmt.Errorf("log pattern training: create coordinator: %w", err)
	}

	totalEvents := 0
	for _, pr := range processedRuns {
		events := buildAgentEventsFromProcessedRun(pr, MetricsData{
			Turns:         pr.Run.Turns,
			TokenUsage:    pr.Run.TokenUsage,
			EstimatedCost: pr.Run.EstimatedCost,
			ErrorCount:    pr.Run.ErrorCount,
			WarningCount:  pr.Run.WarningCount,
		}, nil)
		totalEvents += len(events)
		for _, evt := range events {
			if _, err := coordinator.TrainEvent(evt); err != nil {
				drain3TrainLog.Printf("TrainEvent skipped: stage=%s err=%v", evt.Stage, err)
			}
		}
	}

	if verbose {
		allClusters := coordinator.AllClusters()
		total := 0
		for _, cs := range allClusters {
			total += len(cs)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf(
			"Trained %d events → %d clusters across %d stages",
			totalEvents, total, len(allClusters),
		)))
	}

	weightsData, err := coordinator.SaveWeightsJSON()
	if err != nil {
		return fmt.Errorf("log pattern training: serialize weights: %w", err)
	}

	// Pretty-print the weights for readability.
	var raw map[string]any
	if unmarshalErr := json.Unmarshal(weightsData, &raw); unmarshalErr != nil {
		drain3TrainLog.Printf("Could not unmarshal weights for pretty-printing: %v", unmarshalErr)
	} else if pretty, marshalErr := json.MarshalIndent(raw, "", "  "); marshalErr != nil {
		drain3TrainLog.Printf("Could not indent weights JSON: %v", marshalErr)
	} else {
		weightsData = pretty
	}

	outputPath := filepath.Join(outputDir, drain3WeightsFilename)
	if err := os.WriteFile(outputPath, weightsData, 0o644); err != nil {
		return fmt.Errorf("log pattern training: write weights file: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Log pattern weights written to: "+outputPath))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
		"To embed these weights as default, copy the file and rebuild:\n"+
			"  cp "+outputPath+" pkg/agentdrain/data/default_weights.json\n"+
			"  make build",
	))

	return nil
}
