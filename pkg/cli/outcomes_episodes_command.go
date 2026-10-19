package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

func NewOutcomesEpisodesSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "episodes",
		Short: "Join episode membership with recorded outcome observations",
		Long: `Summarize the latest recorded outcomes for each episode in a logs JSON report.

This is an offline join, not a live evaluator. Membership is limited to the
supplied logs selection. Usage is read from downloaded run-* directories,
including detection usage; missing evidence and cost are explicitly qualified.
Generate membership with gh aw logs --verbose --json and record observations
with gh aw outcomes <run-id> --outcomes-dir <directory>.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` outcomes episodes --logs logs.json --outcomes-dir ./outcomes
  ` + string(constants.CLIExtensionPrefix) + ` outcomes episodes --logs logs.json --outcomes-dir ./outcomes --output ./logs --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logsPath, _ := cmd.Flags().GetString("logs")
			outcomesDir, _ := cmd.Flags().GetString("outcomes-dir")
			outputDir, _ := cmd.Flags().GetString("output")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			return runOutcomesEpisodes(logsPath, outcomesDir, outputDir, jsonOutput, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().String("logs", "", "Full gh aw logs --verbose --json report containing episodes and runs")
	cmd.Flags().String("outcomes-dir", "", "Directory containing outcome JSONL observations")
	cmd.Flags().StringP("output", "o", defaultLogsOutputDir, "Directory containing downloaded run-* usage artifacts")
	addJSONFlag(cmd)
	_ = cmd.MarkFlagRequired("logs")
	_ = cmd.MarkFlagRequired("outcomes-dir")
	return cmd
}

func runOutcomesEpisodes(logsPath, outcomesDir, outputDir string, jsonOutput bool, stdout, stderr io.Writer) error {
	content, err := os.ReadFile(logsPath)
	if err != nil {
		return fmt.Errorf("failed to read episode logs: %w", err)
	}
	var logs episodeOutcomesLogs
	if err := json.Unmarshal(content, &logs); err != nil {
		return fmt.Errorf("failed to parse episode logs: %w", err)
	}
	if len(logs.Episodes) == 0 {
		return errors.New("logs report contains no episodes; regenerate it with gh aw logs --verbose --json")
	}
	observations, err := readEpisodeOutcomeObservations(outcomesDir)
	if err != nil {
		return err
	}
	costs := make(map[int64]episodeOutcomeCost)
	for _, episode := range logs.Episodes {
		for _, id := range episode.RunIDs {
			if _, exists := costs[id]; exists {
				continue
			}
			runDir, err := episodeOutcomeRunDirectory(logs.Runs, outputDir, id)
			if err != nil {
				return err
			}
			usage, err := analyzeTokenUsageAICOnly(runDir, false)
			cost := episodeOutcomeCost{}
			if err != nil {
				warning := fmt.Sprintf("Failed to read usage; cost is unavailable: %v", err)
				cost.warnings = []string{warning}
			} else if usage != nil {
				if usage.AICFound {
					cost.aic = &usage.TotalAIC
				}
				cost.warnings = usage.Warnings
			}
			for _, warning := range cost.warnings {
				if _, err := fmt.Fprintln(stderr, console.FormatWarningMessage(fmt.Sprintf("Run %d: %s", id, warning))); err != nil {
					return fmt.Errorf("failed to write episode usage warning: %w", err)
				}
			}
			costs[id] = cost
		}
	}
	data, err := buildEpisodeOutcomes(logs, observations, costs)
	if err != nil {
		return err
	}
	if jsonOutput {
		output, err := marshalIndentJSONOrWrap(data, "episode outcomes")
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(stdout, string(output)); err != nil {
			return fmt.Errorf("failed to write episode outcomes: %w", err)
		}
		return nil
	}
	return renderEpisodeOutcomes(stdout, data)
}

func episodeOutcomeRunDirectory(runs []RunData, outputDir string, id int64) (string, error) {
	root, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve episode usage output directory: %w", err)
	}
	fallback := filepath.Join(root, fmt.Sprintf("run-%d", id))
	dir := fallback
	var recorded string
	for _, run := range runs {
		if run.RunID != id || run.LogsPath == "" {
			continue
		}
		path, err := filepath.Abs(run.LogsPath)
		if err != nil {
			return "", fmt.Errorf("resolve recorded logs path for run %d: %w", id, err)
		}
		recordedPath := path
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			if !episodeFallbackIsConfinedDirectory(root, fallback) {
				return "", fmt.Errorf("recorded logs path for run %d is outside --output %s", id, root)
			}
			path = fallback
		}
		if recorded != "" && recorded != recordedPath {
			return "", fmt.Errorf("conflicting recorded logs paths for run %d", id)
		}
		recorded, dir = recordedPath, path
	}
	return dir, nil
}

func episodeFallbackIsConfinedDirectory(root, fallback string) bool {
	info, err := os.Stat(fallback)
	if err != nil || !info.IsDir() {
		return false
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	resolvedFallback, err := filepath.EvalSymlinks(fallback)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedFallback)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func readEpisodeOutcomeObservations(dir string) ([]EpisodeOutcomeObservation, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read outcomes directory: %w", err)
	}
	observations := []EpisodeOutcomeObservation{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		reports, err := readEpisodeOutcomeFile(path)
		if err != nil {
			return nil, err
		}
		observations = append(observations, reports...)
	}
	if len(observations) == 0 {
		return nil, fmt.Errorf("no outcome observations found in %s; record them with gh aw outcomes <run-id> --outcomes-dir <directory>", dir)
	}
	return observations, nil
}

func readEpisodeOutcomeFile(path string) (reports []EpisodeOutcomeObservation, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open outcome observations %s: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close outcome observations %s: %w", path, closeErr)
		}
	}()
	decoder := json.NewDecoder(file)
	reports = []EpisodeOutcomeObservation{}
	for {
		var report EpisodeOutcomeObservation
		if err := decoder.Decode(&report); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to parse outcome observation %d in %s: %w", len(reports)+1, path, err)
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func renderEpisodeOutcomes(output io.Writer, data EpisodeOutcomesData) error {
	var lines []string
	for _, episode := range data.Episodes {
		lines = append(lines, fmt.Sprintf("\nEpisode %s (selected runs: %v)", episode.EpisodeID, episode.RunIDs))
		lines = append(lines, fmt.Sprintf("  Observed outcomes: %d accepted, %d rejected, %d ignored, %d lifecycle, %d pending, %d unknown, %d errors (%d total)",
			episode.Summary.Accepted, episode.Summary.Rejected, episode.Summary.Ignored, episode.Summary.Lifecycle, episode.Summary.Pending,
			episode.Summary.Unknown, episode.Summary.Errors, episode.Summary.Total))
		lines = append(lines, fmt.Sprintf("  Outcome evidence: %s (%d/%d selected runs)", episode.OutcomeCoverage, len(episode.OutcomeRunIDs), len(episode.RunIDs)))
		if episode.ObservedAIC == nil {
			lines = append(lines, "  Observed AIC: unavailable")
		} else {
			lines = append(lines, fmt.Sprintf("  Observed AIC: %.3f (%s cost coverage, %d/%d selected runs)",
				*episode.ObservedAIC, episode.CostCoverage, len(episode.CostRunIDs), len(episode.RunIDs)))
		}
		if episode.CostPerAcceptedOutcome == nil {
			lines = append(lines, "  AIC per accepted outcome: unavailable")
		} else {
			lines = append(lines, fmt.Sprintf("  AIC per accepted outcome (selected runs): %.3f", *episode.CostPerAcceptedOutcome))
		}
		for _, qualification := range episode.Qualifications {
			lines = append(lines, "  Note: "+qualification)
		}
	}
	if len(data.UnmatchedOutcomeRunIDs) > 0 {
		lines = append(lines, fmt.Sprintf("\nOutcome source runs outside selected membership (not counted): %v", data.UnmatchedOutcomeRunIDs))
	}
	if _, err := fmt.Fprintln(output, strings.Join(lines, "\n")); err != nil {
		return fmt.Errorf("failed to write episode outcomes: %w", err)
	}
	return nil
}
