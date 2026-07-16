// Package workflow - BinEval evaluation configuration types and parser.
package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var evalsConfigLog = logger.New("workflow:evals_config")

// EvalDefinition represents a single binary evaluation in a BinEval workflow.
// Each evaluation is answered with YES or NO, either by an LLM judge (Question)
// or by running a deterministic shell script (Run). Exactly one of Question or
// Run must be set; they are mutually exclusive.
type EvalDefinition struct {
	ID       string
	Question string
	// Run is the shell script to execute for deterministic evaluation. The script
	// must exit 0 and print exactly YES or NO to stdout. When set, Question must
	// be empty. Information about the agent output and safe-output items is
	// provided via environment variables:
	//   GH_AW_AGENT_OUTPUT       – path to the agent_output.json file
	//   GH_AW_SAFE_OUTPUT_ITEMS  – path to the safe-output-items.jsonl file
	Run string
	// Model is an optional per-question model override. When set, it takes precedence over
	// EvalsConfig.Model. Use a model alias such as "small" or a full model ID.
	// Only applicable for question-type evaluations; ignored when Run is set.
	Model string
}

// EvalsConfig holds the configuration for BinEval-style evaluations declared in workflow
// frontmatter. Evaluations run after safe-outputs and before the conclusion job.
type EvalsConfig struct {
	// Questions is the ordered list of binary evaluation questions.
	Questions []EvalDefinition
	// Model is the default LLM model to use for evaluations. Use a model alias such as
	// "small" or a full model ID. Per-question Model fields override this value.
	// When empty, the compiler default ("small") is used.
	Model string
	// RunsOn allows overriding the runner for the evals job.
	RunsOn string
}

// HasEvals returns true when the config contains at least one evaluation (question or script).
func (ec *EvalsConfig) HasEvals() bool {
	return ec != nil && len(ec.Questions) > 0
}

// QuestionEvals returns the subset of evaluations that use LLM-judged questions.
func (ec *EvalsConfig) QuestionEvals() []EvalDefinition {
	if ec == nil {
		return nil
	}
	var out []EvalDefinition
	for _, q := range ec.Questions {
		if q.Run == "" {
			out = append(out, q)
		}
	}
	return out
}

// ScriptEvals returns the subset of evaluations that use deterministic shell scripts.
func (ec *EvalsConfig) ScriptEvals() []EvalDefinition {
	if ec == nil {
		return nil
	}
	var out []EvalDefinition
	for _, q := range ec.Questions {
		if q.Run != "" {
			out = append(out, q)
		}
	}
	return out
}

// parseEvalsFromFrontmatter extracts and validates the evals configuration from the
// raw frontmatter map. Returns nil when the evals field is absent.
//
// Supported forms:
//
//	# Shorthand — plain list
//	evals:
//	  - id: builds
//	    question: Does the generated code compile?
//	  - id: output_check
//	    run: ./scripts/check_output.sh   # deterministic script, prints YES or NO
//
//	# Extended — object with questions list and optional model/runs-on
//	evals:
//	  questions:
//	    - id: builds
//	      question: Does the generated code compile?
//	  model: small
func (c *Compiler) parseEvalsFromFrontmatter(frontmatter map[string]any) (*EvalsConfig, error) {
	raw, exists := frontmatter["evals"]
	if !exists || raw == nil {
		return nil, nil
	}

	cfg := &EvalsConfig{}

	switch v := raw.(type) {
	case []any:
		// Shorthand form: plain list of questions
		questions, err := parseEvalDefinitions(v)
		if err != nil {
			return nil, fmt.Errorf("evals: %w", err)
		}
		cfg.Questions = questions

	case map[string]any:
		// Extended form: object with questions and optional model/runs-on
		if questionsRaw, ok := v["questions"]; ok {
			questionsList, ok := questionsRaw.([]any)
			if !ok {
				return nil, fmt.Errorf("evals.questions: must be a list of question objects, got %T", questionsRaw)
			}
			questions, err := parseEvalDefinitions(questionsList)
			if err != nil {
				return nil, fmt.Errorf("evals.questions: %w", err)
			}
			cfg.Questions = questions
		}

		// Parse optional top-level model (default for all questions)
		if modelRaw, ok := v["model"]; ok {
			modelStr, ok := modelRaw.(string)
			if !ok {
				return nil, fmt.Errorf("evals.model: must be a string, got %T", modelRaw)
			}
			cfg.Model = strings.TrimSpace(modelStr)
		}

		// Parse optional runs-on override
		if runsOnRaw, ok := v["runs-on"]; ok {
			cfg.RunsOn = renderRunsOnSnippet(runsOnRaw)
		}

	default:
		return nil, errors.New("evals: must be a list of questions or an object with a questions list")
	}

	if err := validateEvals(cfg); err != nil {
		return nil, err
	}

	perQuestionOverrides := 0
	scriptEvals := 0
	for _, q := range cfg.Questions {
		if q.Model != "" {
			perQuestionOverrides++
		}
		if q.Run != "" {
			scriptEvals++
		}
	}
	evalsConfigLog.Printf("Parsed %d eval definitions (%d script, model: %q, per-question overrides: %d)", len(cfg.Questions), scriptEvals, cfg.Model, perQuestionOverrides)
	return cfg, nil
}

// parseEvalDefinitions converts a []any YAML list into []EvalDefinition.
func parseEvalDefinitions(items []any) ([]EvalDefinition, error) {
	defs := make([]EvalDefinition, 0, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("item %d must be an object with id and question or run fields", i)
		}
		def, err := parseEvalDefinition(m, i)
		if err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}
	return defs, nil
}

// parseEvalDefinition converts a single map entry into an EvalDefinition.
// Exactly one of "question" or "run" must be present (they are mutually exclusive).
func parseEvalDefinition(m map[string]any, idx int) (EvalDefinition, error) {
	idRaw, hasID := m["id"]
	questionRaw, hasQuestion := m["question"]
	runRaw, hasRun := m["run"]

	if !hasID {
		return EvalDefinition{}, fmt.Errorf("item %d: missing required field 'id'", idx)
	}
	if hasQuestion && hasRun {
		return EvalDefinition{}, fmt.Errorf("item %d: 'question' and 'run' are mutually exclusive; set exactly one", idx)
	}
	if !hasQuestion && !hasRun {
		return EvalDefinition{}, fmt.Errorf("item %d: missing required field: set either 'question' or 'run'", idx)
	}

	id, ok := idRaw.(string)
	if !ok || strings.TrimSpace(id) == "" {
		return EvalDefinition{}, fmt.Errorf("item %d: 'id' must be a non-empty string", idx)
	}

	def := EvalDefinition{
		ID: strings.TrimSpace(id),
	}

	if hasQuestion {
		question, ok := questionRaw.(string)
		if !ok || strings.TrimSpace(question) == "" {
			return EvalDefinition{}, fmt.Errorf("item %d: 'question' must be a non-empty string", idx)
		}
		def.Question = strings.TrimSpace(question)
	}

	if hasRun {
		run, ok := runRaw.(string)
		if !ok || strings.TrimSpace(run) == "" {
			return EvalDefinition{}, fmt.Errorf("item %d: 'run' must be a non-empty string", idx)
		}
		def.Run = strings.TrimSpace(run)
	}

	// Optional per-question model override (only meaningful for question-type evals).
	if modelRaw, ok := m["model"]; ok {
		modelStr, ok := modelRaw.(string)
		if !ok {
			return EvalDefinition{}, fmt.Errorf("item %d: 'model' must be a string, got %T", idx, modelRaw)
		}
		def.Model = strings.TrimSpace(modelStr)
	}

	return def, nil
}

// validateEvals checks for duplicate IDs and validates individual eval definitions after parsing.
func validateEvals(cfg *EvalsConfig) error {
	if cfg == nil {
		return nil
	}
	if len(cfg.Questions) == 0 {
		return errors.New("evals: at least one question is required when evals is configured")
	}

	seen := make(map[string]struct{}, len(cfg.Questions))
	for i, q := range cfg.Questions {
		if _, dup := seen[q.ID]; dup {
			return fmt.Errorf("evals: duplicate id %q at index %d", q.ID, i)
		}
		seen[q.ID] = struct{}{}

		if q.Run == "" && strings.TrimSpace(q.Question) == "" {
			return fmt.Errorf("evals: question for id %q must be non-empty", q.ID)
		}
		if q.Run != "" && strings.TrimSpace(q.Run) == "" {
			return fmt.Errorf("evals: run script for id %q must be non-empty", q.ID)
		}
	}
	return nil
}

// ParseEvalsFromFrontmatter extracts and validates the evals configuration from the
// raw frontmatter map. Returns nil when the evals field is absent or invalid.
// This is a public standalone convenience wrapper around the compiler method.
func ParseEvalsFromFrontmatter(frontmatter map[string]any) (*EvalsConfig, error) {
	var c Compiler
	return c.parseEvalsFromFrontmatter(frontmatter)
}
