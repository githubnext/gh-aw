package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

type episodeExplanationView struct {
	EpisodeID string          `json:"episode_id"`
	Coverage  EpisodeCoverage `json:"coverage"`
	*EpisodeExplanation
}

func newLogsExplainCommand() *cobra.Command {
	var episodeID string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "explain <logs.json>",
		Short: "Explain recorded episode hops without retrieving more evidence",
		Long:  "Show recorded hop identities, same-run reusable-workflow links, run edges, and missing evidence from logs JSON. Costs remain attached once to each run, never allocated or duplicated across hops.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return runLogsExplain(args[0], episodeID, jsonOutput, cmd.OutOrStdout())
			}
			return errors.New("expected a logs JSON file")
		},
	}
	cmd.Flags().StringVar(&episodeID, "episode", "", "Recorded episode ID to explain (defaults to all selected episodes)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured hop explanations and separate run costs")
	return cmd
}

func runLogsExplain(path, episodeID string, jsonOutput bool, out io.Writer) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read episode logs: %w", err)
	}
	var logs LogsData
	if err := json.Unmarshal(content, &logs); err != nil {
		return fmt.Errorf("decode episode logs: %w", err)
	}
	views := make([]episodeExplanationView, 0, len(logs.Episodes))
	for _, episode := range logs.Episodes {
		if episodeID != "" && episode.EpisodeID != episodeID {
			continue
		}
		explanation := episode.Explanation
		if explanation == nil {
			explanation = explainEpisode(episode, logs.Runs, logs.Edges)
		}
		views = append(views, episodeExplanationView{EpisodeID: episode.EpisodeID, Coverage: episode.Coverage, EpisodeExplanation: explanation})
	}
	if len(views) == 0 {
		return fmt.Errorf("no recorded episode matches %q in the logs report", episodeID)
	}
	if jsonOutput {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(views)
	}
	return renderEpisodeExplanations(out, views)
}

func renderEpisodeExplanations(out io.Writer, views []episodeExplanationView) error {
	var text []byte
	for _, view := range views {
		text = fmt.Appendf(text, "Episode %s\nCoverage: %s (%s); complete membership is not established.\n", view.EpisodeID, view.Coverage.Status, view.Coverage.Scope)
		for _, hop := range view.Hops {
			label := hop.HopID
			if label == "" {
				label = "(hop ID not recorded)"
			}
			text = fmt.Appendf(text, "  Hop %s: run %d, %s, workflow %s", label, hop.RunID, hop.Repository, hop.Workflow)
			if hop.SharedRun {
				text = append(text, " [shared run]"...)
			}
			text = append(text, '\n')
			for _, gap := range hop.MissingEvidence {
				text = fmt.Appendf(text, "    Missing evidence: %s\n", gap)
			}
		}
		for _, edge := range view.Edges {
			text = fmt.Appendf(text, "  Hop edge %s -> %s: %s", edge.SourceHopID, edge.TargetHopID, edge.Status)
			if edge.Reason != "" {
				text = fmt.Appendf(text, " (%s)", edge.Reason)
			}
			text = append(text, '\n')
		}
		for _, edge := range view.RunEdges {
			text = fmt.Appendf(text, "  Recorded run edge %d -> %d: %s\n", edge.SourceRunID, edge.TargetRunID, edge.EdgeType)
		}
		for _, gap := range append(append([]string{}, view.Coverage.Reasons...), view.MissingEvidence...) {
			text = fmt.Appendf(text, "  Missing evidence: %s\n", gap)
		}
		text = append(text, "Run-level accounting (observed once per run; not allocated to hops):\n"...)
		for _, cost := range view.RunCosts {
			text = fmt.Appendf(text, "  Run %d: %d tokens, %.6g AI credits\n", cost.RunID, cost.Tokens, cost.AIC)
		}
		text = append(text, '\n')
	}
	if _, err := io.Copy(out, bytes.NewReader(text)); err != nil {
		return fmt.Errorf("write episode explanation: %w", err)
	}
	return nil
}
