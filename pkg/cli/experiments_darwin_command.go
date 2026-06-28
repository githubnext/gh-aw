package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

const defaultDarwinArchiveDir = ".github/experiments/archive"

// ExperimentsDarwinConfig holds configuration for the experiments darwin subcommand.
type ExperimentsDarwinConfig struct {
	WorkflowID   string
	Experiment   string
	ArchiveDir   string
	Apply        bool
	JSONOutput   bool
	Winner       string
	NextVariants []string
}

// DarwinVariantScore captures the ranking of a single variant in a Darwin run.
type DarwinVariantScore struct {
	Name              string `json:"name"`
	Count             int    `json:"count"`
	CurrentControl    bool   `json:"current_control"`
	PromotedToControl bool   `json:"promoted_to_control"`
}

// DarwinPlan summarizes the result of a Darwin run for one named experiment.
type DarwinPlan struct {
	WorkflowID      string               `json:"workflow_id"`
	WorkflowPath    string               `json:"workflow_path,omitempty"`
	ExperimentName  string               `json:"experiment_name"`
	Branch          string               `json:"branch"`
	ArchivePath     string               `json:"archive_path,omitempty"`
	ArchivedAt      string               `json:"archived_at,omitempty"`
	Apply           bool                 `json:"apply"`
	Winner          string               `json:"winner"`
	CurrentVariants []string             `json:"current_variants"`
	NextVariants    []string             `json:"next_variants"`
	Ranking         []DarwinVariantScore `json:"ranking"`
	Analysis        ExperimentAnalysis   `json:"analysis"`
}

// DarwinArchive stores the archived state and promotion decision for one Darwin run.
type DarwinArchive struct {
	WorkflowID      string               `json:"workflow_id"`
	WorkflowPath    string               `json:"workflow_path"`
	ExperimentName  string               `json:"experiment_name"`
	Branch          string               `json:"branch"`
	ArchivedAt      string               `json:"archived_at"`
	Winner          string               `json:"winner"`
	CurrentVariants []string             `json:"current_variants"`
	NextVariants    []string             `json:"next_variants"`
	Ranking         []DarwinVariantScore `json:"ranking"`
	Analysis        ExperimentAnalysis   `json:"analysis"`
	State           *ExperimentState     `json:"state"`
}

// NewExperimentsDarwinSubcommand creates the experiments darwin subcommand.
func NewExperimentsDarwinSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "darwin <workflow> <experiment>",
		Short: "Evaluate and promote the next generation of an experiment",
		Long: `Evaluate and promote the next generation of an existing workflow experiment.

Darwin mode is an experimental extension of gh-aw experiments. It reuses the
existing experiments state.json history to rank the currently declared variants,
archives the current generation, and promotes a selected winner to control by
moving it to the front of the variants list.

When --variant is omitted, Darwin keeps the current population and only
reorders it so the promoted winner becomes the first variant. When --variant is
provided, Darwin generates the next generation from the promoted winner plus the
explicitly provided variants. Darwin only updates experiment frontmatter; any
conditional prompt branches for new variant names must already exist in the
workflow source.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` experiments darwin myworkflow style
  ` + string(constants.CLIExtensionPrefix) + ` experiments darwin myworkflow style --winner concise
  ` + string(constants.CLIExtensionPrefix) + ` experiments darwin myworkflow style --variant concise --variant detailed --apply
  ` + string(constants.CLIExtensionPrefix) + ` experiments darwin myworkflow style --archive-dir .github/experiments/archive --json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			apply, _ := cmd.Flags().GetBool("apply")
			winner, _ := cmd.Flags().GetString("winner")
			archiveDir, _ := cmd.Flags().GetString("archive-dir")
			nextVariants, _ := cmd.Flags().GetStringSlice("variant")
			return RunExperimentsDarwin(ExperimentsDarwinConfig{
				WorkflowID:   args[0],
				Experiment:   args[1],
				ArchiveDir:   archiveDir,
				Apply:        apply,
				JSONOutput:   jsonOutput,
				Winner:       winner,
				NextVariants: nextVariants,
			})
		},
	}

	addJSONFlag(cmd)
	cmd.Flags().Bool("apply", false, "Archive the current generation and update the workflow file in place")
	cmd.Flags().String("winner", "", "Override the promoted winner variant")
	cmd.Flags().String("archive-dir", defaultDarwinArchiveDir, "Directory where Darwin archives are written when --apply is used")
	cmd.Flags().StringSlice("variant", nil, "Variant names for the next generation (repeatable)")

	return cmd
}

// RunExperimentsDarwin evaluates, archives, and optionally promotes a next generation.
func RunExperimentsDarwin(config ExperimentsDarwinConfig) error {
	plan, archive, err := buildDarwinPlan(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		return nil
	}

	if config.Apply {
		if err := writeDarwinArchive(plan.ArchivePath, archive); err != nil {
			return fmt.Errorf("failed to write Darwin archive: %w", err)
		}
		if err := applyDarwinPromotion(plan.WorkflowPath, plan.ExperimentName, plan.NextVariants); err != nil {
			return fmt.Errorf("failed to update workflow file: %w", err)
		}
	}

	if config.JSONOutput {
		jsonBytes, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(jsonBytes))
		return nil
	}

	printDarwinPlan(plan)
	return nil
}

func buildDarwinPlan(config ExperimentsDarwinConfig) (*DarwinPlan, *DarwinArchive, error) {
	if strings.TrimSpace(config.WorkflowID) == "" {
		return nil, nil, errors.New("workflow is required")
	}
	if strings.TrimSpace(config.Experiment) == "" {
		return nil, nil, errors.New("experiment name is required")
	}
	if strings.TrimSpace(config.ArchiveDir) == "" {
		config.ArchiveDir = defaultDarwinArchiveDir
	}

	workflowID := workflow.SanitizeWorkflowIDForCacheKey(config.WorkflowID)
	if workflowID == "" {
		return nil, nil, errors.New("workflow is required")
	}

	workflowPath := findWorkflowFileForExperiment(workflowID)
	if workflowPath == "" {
		return nil, nil, fmt.Errorf("workflow %q not found in .github/workflows/", config.WorkflowID)
	}

	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read workflow file: %w", err)
	}
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse workflow frontmatter: %w", err)
	}
	cfg, err := workflow.ParseFrontmatterConfig(result.Frontmatter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse workflow config: %w", err)
	}

	expCfg := cfg.ExperimentConfigs[config.Experiment]
	if expCfg == nil {
		return nil, nil, fmt.Errorf("experiment %q not found in workflow %q", config.Experiment, config.WorkflowID)
	}
	if len(expCfg.Variants) == 0 {
		return nil, nil, fmt.Errorf("experiment %q has no variants", config.Experiment)
	}

	branchName := experimentsBranchPrefix + workflowID
	details, err := fetchLocalExperimentDetails(branchName, workflowID)
	if err != nil {
		state := emptyExperimentState()
		details = experimentDetailsFromState(workflowID, branchName, state)
		if !strings.Contains(err.Error(), "not found locally") {
			return nil, nil, err
		}
	}
	stateRef := "origin/" + branchName
	if !gitRefExists(stateRef) {
		stateRef = branchName
	}
	state := emptyExperimentState()
	if gitRefExists(stateRef) {
		state = readLocalExperimentState(stateRef)
	}

	stats := findOrBuildExperimentStats(details.Experiments, config.Experiment, expCfg.Variants)
	analysis := computeExperimentAnalysis(stats, expCfg)

	ranking := rankDarwinVariants(expCfg.Variants, stats.Variants)
	winner, err := selectDarwinWinner(config.Winner, ranking)
	if err != nil {
		return nil, nil, err
	}
	nextVariants := buildDarwinNextVariants(winner, expCfg.Variants, ranking, config.NextVariants)
	if len(nextVariants) < 2 {
		return nil, nil, fmt.Errorf("darwin generation for %q must contain at least 2 variants", config.Experiment)
	}
	archivePath, archivedAt := darwinArchiveLocation(config.ArchiveDir, workflowID, config.Experiment)
	rankedRows := darwinRankingRows(ranking, expCfg.Variants[0], winner)

	plan := &DarwinPlan{
		WorkflowID:      workflowID,
		WorkflowPath:    workflowPath,
		ExperimentName:  config.Experiment,
		Branch:          branchName,
		ArchivePath:     archivePath,
		ArchivedAt:      archivedAt,
		Apply:           config.Apply,
		Winner:          winner,
		CurrentVariants: slices.Clone(expCfg.Variants),
		NextVariants:    nextVariants,
		Ranking:         rankedRows,
		Analysis:        analysis,
	}
	archive := &DarwinArchive{
		WorkflowID:      workflowID,
		WorkflowPath:    workflowPath,
		ExperimentName:  config.Experiment,
		Branch:          branchName,
		ArchivedAt:      archivedAt,
		Winner:          winner,
		CurrentVariants: slices.Clone(expCfg.Variants),
		NextVariants:    slices.Clone(nextVariants),
		Ranking:         rankedRows,
		Analysis:        analysis,
		State:           state,
	}
	return plan, archive, nil
}

func findOrBuildExperimentStats(experiments []ExperimentVariantStats, name string, declaredVariants []string) ExperimentVariantStats {
	for _, exp := range experiments {
		if exp.Name != name {
			continue
		}
		variants := make(map[string]int, len(exp.Variants)+len(declaredVariants))
		maps.Copy(variants, exp.Variants)
		for _, variant := range declaredVariants {
			if _, ok := variants[variant]; !ok {
				variants[variant] = 0
			}
		}
		total := 0
		for _, count := range variants {
			total += count
		}
		return ExperimentVariantStats{Name: name, Variants: variants, Total: total}
	}

	variants := make(map[string]int, len(declaredVariants))
	for _, variant := range declaredVariants {
		variants[variant] = 0
	}
	return ExperimentVariantStats{Name: name, Variants: variants, Total: 0}
}

func rankDarwinVariants(currentVariants []string, counts map[string]int) []DarwinVariantScore {
	if len(currentVariants) == 0 {
		return nil
	}
	indexByName := make(map[string]int, len(currentVariants))
	ranking := make([]DarwinVariantScore, 0, len(currentVariants))
	for i, variant := range currentVariants {
		indexByName[variant] = i
		ranking = append(ranking, DarwinVariantScore{
			Name:           variant,
			Count:          counts[variant],
			CurrentControl: i == 0,
		})
	}
	slices.SortFunc(ranking, func(a, b DarwinVariantScore) int {
		switch {
		case a.Count > b.Count:
			return -1
		case a.Count < b.Count:
			return 1
		case indexByName[a.Name] < indexByName[b.Name]:
			return -1
		case indexByName[a.Name] > indexByName[b.Name]:
			return 1
		default:
			return strings.Compare(a.Name, b.Name)
		}
	})
	return ranking
}

func selectDarwinWinner(override string, ranking []DarwinVariantScore) (string, error) {
	if len(ranking) == 0 {
		return "", errors.New("no variants available to evaluate")
	}
	override = strings.TrimSpace(override)
	if override == "" {
		return ranking[0].Name, nil
	}
	for _, variant := range ranking {
		if variant.Name == override {
			return override, nil
		}
	}
	return "", fmt.Errorf("winner %q is not a declared variant", override)
}

func buildDarwinNextVariants(winner string, currentVariants []string, ranking []DarwinVariantScore, requested []string) []string {
	seen := map[string]struct{}{}
	nextVariants := []string{winner}
	seen[winner] = struct{}{}

	addVariant := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		nextVariants = append(nextVariants, name)
	}

	if len(requested) > 0 {
		for _, name := range requested {
			addVariant(name)
		}
		return nextVariants
	}

	for _, variant := range ranking {
		addVariant(variant.Name)
	}
	for _, variant := range currentVariants {
		addVariant(variant)
	}
	return nextVariants
}

func darwinArchiveLocation(baseDir, workflowID, experimentName string) (string, string) {
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	return filepath.Join(baseDir, workflowID, experimentName, timestamp+".json"), timestamp
}

func darwinRankingRows(ranking []DarwinVariantScore, currentControl, winner string) []DarwinVariantScore {
	rows := make([]DarwinVariantScore, 0, len(ranking))
	for _, variant := range ranking {
		rows = append(rows, DarwinVariantScore{
			Name:              variant.Name,
			Count:             variant.Count,
			CurrentControl:    variant.Name == currentControl,
			PromotedToControl: variant.Name == winner,
		})
	}
	return rows
}

func writeDarwinArchive(archivePath string, archive *DarwinArchive) error {
	if archive == nil {
		return errors.New("archive is required")
	}
	if err := os.MkdirAll(filepath.Dir(archivePath), constants.DirPermPublic); err != nil {
		return err
	}
	content, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(archivePath, content, constants.FilePermPublic)
}

func applyDarwinPromotion(workflowPath, experimentName string, nextVariants []string) error {
	return parser.UpdateWorkflowFrontmatter(workflowPath, func(frontmatter map[string]any) error {
		rawExperiments, ok := frontmatter["experiments"].(map[string]any)
		if !ok {
			return errors.New("workflow frontmatter does not contain an experiments map")
		}
		rawExperiment, ok := rawExperiments[experimentName]
		if !ok {
			return fmt.Errorf("experiment %q not found in workflow frontmatter", experimentName)
		}

		updatedVariants := make([]any, 0, len(nextVariants))
		for _, variant := range nextVariants {
			updatedVariants = append(updatedVariants, variant)
		}

		switch typed := rawExperiment.(type) {
		case []any:
			rawExperiments[experimentName] = updatedVariants
		case map[string]any:
			typed["variants"] = updatedVariants
			rawExperiments[experimentName] = typed
		default:
			return fmt.Errorf("experiment %q has unsupported frontmatter shape", experimentName)
		}
		return nil
	}, false)
}

func printDarwinPlan(plan *DarwinPlan) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Darwin mode: "+plan.WorkflowID+"/"+plan.ExperimentName))
	fmt.Fprintf(os.Stderr, "  Branch:        %s\n", plan.Branch)
	fmt.Fprintf(os.Stderr, "  Winner:        %s\n", plan.Winner)
	fmt.Fprintf(os.Stderr, "  Recommendation:%s\n", plan.Analysis.Recommendation)
	fmt.Fprintf(os.Stderr, "  Current:       %s\n", strings.Join(plan.CurrentVariants, ", "))
	fmt.Fprintf(os.Stderr, "  Next:          %s\n", strings.Join(plan.NextVariants, ", "))
	if plan.Apply {
		fmt.Fprintf(os.Stderr, "  Archive:       %s\n", plan.ArchivePath)
	}

	rows := make([][]string, 0, len(plan.Ranking))
	for _, variant := range plan.Ranking {
		status := []string{}
		if variant.CurrentControl {
			status = append(status, "control")
		}
		if variant.PromotedToControl {
			status = append(status, "promoted")
		}
		label := strings.Join(status, ", ")
		if label == "" {
			label = "-"
		}
		rows = append(rows, []string{variant.Name, strconv.Itoa(variant.Count), label})
	}
	fmt.Fprintf(os.Stderr, "\n%s", console.RenderTable(console.TableConfig{
		Title:   "Variant ranking",
		Headers: []string{"Variant", "Count", "Status"},
		Rows:    rows,
	}))
}
