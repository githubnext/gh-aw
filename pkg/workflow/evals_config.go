// Package workflow - BinEval evaluation configuration types and parser.
package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var evalsConfigLog = logger.New("workflow:evals_config")

// EvalDefinition represents a single binary evaluation question in a BinEval workflow.
// Each question is evaluated independently and answered with YES or NO.
type EvalDefinition struct {
	ID       string `yaml:"id"`
	Question string `yaml:"question"`
}

// EvalsConfig holds the configuration for BinEval-style evaluations declared in workflow
// frontmatter. Evaluations run after safe-outputs and before the conclusion job.
type EvalsConfig struct {
	// Questions is the ordered list of binary evaluation questions.
	Questions []EvalDefinition
	// EngineConfig allows overriding the evaluation engine (model, API target, etc.).
	// When nil, a default small model is used.
	EngineConfig *EngineConfig
	// RunsOn allows overriding the runner for the evals job.
	RunsOn string
}

// HasEvals returns true when the config contains at least one evaluation question.
func (ec *EvalsConfig) HasEvals() bool {
	return ec != nil && len(ec.Questions) > 0
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
//
//	# Extended — object with questions list and optional engine-config
//	evals:
//	  questions:
//	    - id: builds
//	      question: Does the generated code compile?
//	  engine-config:
//	    model: gpt-4o-mini
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
		// Extended form: object with questions and optional engine-config
		if questionsRaw, ok := v["questions"]; ok {
			if questionsList, ok := questionsRaw.([]any); ok {
				questions, err := parseEvalDefinitions(questionsList)
				if err != nil {
					return nil, fmt.Errorf("evals.questions: %w", err)
				}
				cfg.Questions = questions
			}
		}

		// Parse engine-config if present
		if engineRaw, ok := v["engine-config"]; ok {
			if engineMap, ok := engineRaw.(map[string]any); ok {
				_, engineConfig := c.ExtractEngineConfig(map[string]any{"engine": engineMap})
				cfg.EngineConfig = engineConfig
				evalsConfigLog.Printf("Parsed evals engine-config")
			} else if engineStr, ok := engineRaw.(string); ok {
				cfg.EngineConfig = &EngineConfig{ID: engineStr}
			}
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

	evalsConfigLog.Printf("Parsed %d eval definitions", len(cfg.Questions))
	return cfg, nil
}

// parseEvalDefinitions converts a []any YAML list into []EvalDefinition.
func parseEvalDefinitions(items []any) ([]EvalDefinition, error) {
	defs := make([]EvalDefinition, 0, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("item %d must be an object with id and question fields", i)
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
func parseEvalDefinition(m map[string]any, idx int) (EvalDefinition, error) {
	idRaw, hasID := m["id"]
	questionRaw, hasQuestion := m["question"]

	if !hasID {
		return EvalDefinition{}, fmt.Errorf("item %d: missing required field 'id'", idx)
	}
	if !hasQuestion {
		return EvalDefinition{}, fmt.Errorf("item %d: missing required field 'question'", idx)
	}

	id, ok := idRaw.(string)
	if !ok || strings.TrimSpace(id) == "" {
		return EvalDefinition{}, fmt.Errorf("item %d: 'id' must be a non-empty string", idx)
	}

	question, ok := questionRaw.(string)
	if !ok || strings.TrimSpace(question) == "" {
		return EvalDefinition{}, fmt.Errorf("item %d: 'question' must be a non-empty string", idx)
	}

	return EvalDefinition{
		ID:       strings.TrimSpace(id),
		Question: strings.TrimSpace(question),
	}, nil
}

// validateEvals checks for duplicate IDs and non-empty questions after parsing.
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

		if strings.TrimSpace(q.Question) == "" {
			return fmt.Errorf("evals: question for id %q must be non-empty", q.ID)
		}
	}
	return nil
}

// evalsBranchName returns the git branch name used to persist evaluation results.
// Format: evals/<workflow-id>
func evalsBranchName(workflowID string) string {
	return "evals/" + workflowID
}
