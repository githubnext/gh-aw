// Package workflow - BinEval evaluation configuration types and parser.
package workflow

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var evalsConfigLog = logger.New("workflow:evals_config")

// EvalDefinition represents a single binary evaluation question in a BinEval workflow.
// Each question is evaluated independently and answered with YES or NO.
type EvalDefinition struct {
	ID       string
	Question string
	// StructuredCheck optionally derives deterministic evidence from a structured output item.
	StructuredCheck *EvalStructuredCheck
	// Model is an optional per-question model override. When set, it takes precedence over
	// EvalsConfig.Model. Use a model alias such as "small" or a full model ID.
	Model string
}

// EvalStructuredCheck compares one structured output item against the configured values
// for the run's UTC weekday.
type EvalStructuredCheck struct {
	ItemType             string                       `json:"item_type"`
	ResultKey            string                       `json:"result_key"`
	ExpectedByUTCWeekday map[string]map[string]string `json:"expected_by_utc_weekday"`
}

// EvalsConfig holds the configuration for BinEval-style evaluations declared in workflow
// frontmatter. Evaluations run after safe-outputs and before the conclusion job.
type EvalsConfig struct {
	// Questions is the ordered list of binary evaluation questions.
	Questions []EvalDefinition
	// IncludeAssistantText adds the agent's final assistant message to evaluation context.
	IncludeAssistantText bool
	// Model is the default LLM model to use for evaluations. Use a model alias such as
	// "small" or a full model ID. Per-question Model fields override this value.
	// When empty, the compiler default ("evals") is used.
	Model string
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
//	# Extended — object with questions list and optional execution settings
//	evals:
//	  questions:
//	    - id: builds
//	      question: Does the generated code compile?
//	  include-assistant-text: true
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
				return nil, fmt.Errorf("evals.questions must be a list of question objects, got %T. Example:\nevals:\n  questions:\n    - id: readme\n      question: Does the README explain setup?", questionsRaw)
			}
			questions, err := parseEvalDefinitions(questionsList)
			if err != nil {
				return nil, fmt.Errorf("evals.questions: %w", err)
			}
			cfg.Questions = questions
		}
		if err := parseEvalsOptions(v, cfg); err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("evals must be a list of questions or an object with a questions list. Example:\nevals:\n  - id: readme\n    question: Does the README explain setup?")
	}

	if err := validateEvals(cfg); err != nil {
		return nil, err
	}

	perQuestionOverrides := 0
	for _, q := range cfg.Questions {
		if q.Model != "" {
			perQuestionOverrides++
		}
	}
	evalsConfigLog.Printf("Parsed %d eval definitions (model: %q, per-question overrides: %d)", len(cfg.Questions), cfg.Model, perQuestionOverrides)
	return cfg, nil
}

func parseEvalsOptions(values map[string]any, cfg *EvalsConfig) error {
	if includeAssistantTextRaw, ok := values["include-assistant-text"]; ok {
		includeAssistantText, ok := includeAssistantTextRaw.(bool)
		if !ok {
			return fmt.Errorf("evals.include-assistant-text must be a boolean, got %T", includeAssistantTextRaw)
		}
		cfg.IncludeAssistantText = includeAssistantText
	}

	if modelRaw, ok := values["model"]; ok {
		modelStr, ok := modelRaw.(string)
		if !ok {
			return fmt.Errorf("evals.model must be a string, got %T. Example:\nevals:\n  model: small", modelRaw)
		}
		cfg.Model = strings.TrimSpace(modelStr)
	}

	if runsOnRaw, ok := values["runs-on"]; ok {
		cfg.RunsOn = renderRunsOnSnippet(runsOnRaw)
	}
	return nil
}

// parseEvalDefinitions converts a []any YAML list into []EvalDefinition.
func parseEvalDefinitions(items []any) ([]EvalDefinition, error) {
	defs := make([]EvalDefinition, 0, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("evals item %d must be an object with 'id' and 'question' fields, got a different type. Example:\nevals:\n  - id: readme\n    question: Does the README explain setup?", i)
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
		return EvalDefinition{}, fmt.Errorf("evals item %d is missing the required 'id' field. Example:\nevals:\n  - id: readme\n    question: Does the README explain setup?", idx)
	}
	if !hasQuestion {
		return EvalDefinition{}, fmt.Errorf("evals item %d is missing the required 'question' field. Example:\nevals:\n  - id: readme\n    question: Does the README explain setup?", idx)
	}

	id, ok := idRaw.(string)
	if !ok || strings.TrimSpace(id) == "" {
		return EvalDefinition{}, fmt.Errorf("evals item %d has an 'id' that is not a non-empty string. Example:\nevals:\n  - id: readme\n    question: Does the README explain setup?", idx)
	}

	question, ok := questionRaw.(string)
	if !ok || strings.TrimSpace(question) == "" {
		return EvalDefinition{}, fmt.Errorf("evals item %d has a 'question' that is not a non-empty string. Example:\nevals:\n  - id: readme\n    question: Does the README explain setup?", idx)
	}

	def := EvalDefinition{
		ID:       strings.TrimSpace(id),
		Question: strings.TrimSpace(question),
	}

	if checkRaw, ok := m["structured_check"]; ok {
		check, err := parseEvalStructuredCheck(checkRaw, idx)
		if err != nil {
			return EvalDefinition{}, err
		}
		def.StructuredCheck = check
	}

	// Optional per-question model override.
	if modelRaw, ok := m["model"]; ok {
		modelStr, ok := modelRaw.(string)
		if !ok {
			return EvalDefinition{}, fmt.Errorf("evals item %d has a 'model' that must be a string, got %T. Example:\nevals:\n  - id: readme\n    question: Does the README explain setup?\n    model: small", idx, modelRaw)
		}
		def.Model = strings.TrimSpace(modelStr)
	}

	return def, nil
}

func parseEvalStructuredCheck(raw any, idx int) (*EvalStructuredCheck, error) {
	check, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("evals item %d 'structured_check' must be an object", idx)
	}
	itemType, ok := check["item_type"].(string)
	if !ok || strings.TrimSpace(itemType) == "" {
		return nil, fmt.Errorf("evals item %d 'structured_check.item_type' must be a non-empty string", idx)
	}
	resultKey, ok := check["result_key"].(string)
	if !ok || strings.TrimSpace(resultKey) == "" {
		return nil, fmt.Errorf("evals item %d 'structured_check.result_key' must be a non-empty string", idx)
	}
	expectedRaw, ok := check["expected_by_utc_weekday"].(map[string]any)
	if !ok || len(expectedRaw) == 0 {
		return nil, fmt.Errorf("evals item %d 'structured_check.expected_by_utc_weekday' must be a non-empty object", idx)
	}

	expectedByUTCWeekday := make(map[string]map[string]string, len(expectedRaw))
	for weekday, valuesRaw := range expectedRaw {
		day, err := strconv.Atoi(weekday)
		if err != nil || day < 0 || day > 6 || strconv.Itoa(day) != weekday {
			return nil, fmt.Errorf("evals item %d 'structured_check.expected_by_utc_weekday' key %q must be a UTC weekday from 0 to 6", idx, weekday)
		}
		values, ok := valuesRaw.(map[string]any)
		if !ok || len(values) == 0 {
			return nil, fmt.Errorf("evals item %d 'structured_check.expected_by_utc_weekday.%s' must be a non-empty object", idx, weekday)
		}
		expectedValues := make(map[string]string, len(values))
		for field, valueRaw := range values {
			value, ok := valueRaw.(string)
			if !ok || strings.TrimSpace(field) == "" || strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("evals item %d 'structured_check.expected_by_utc_weekday.%s' values must use non-empty fields and string values", idx, weekday)
			}
			expectedValues[strings.TrimSpace(field)] = value
		}
		expectedByUTCWeekday[weekday] = expectedValues
	}

	return &EvalStructuredCheck{
		ItemType:             strings.TrimSpace(itemType),
		ResultKey:            strings.TrimSpace(resultKey),
		ExpectedByUTCWeekday: expectedByUTCWeekday,
	}, nil
}

// validateEvals checks for duplicate IDs and non-empty questions after parsing.
func validateEvals(cfg *EvalsConfig) error {
	if cfg == nil {
		return nil
	}
	if len(cfg.Questions) == 0 {
		return errors.New("evals requires at least one question when configured. Example:\nevals:\n  - id: readme\n    question: Does the README explain setup?")
	}

	seen := make(map[string]struct{}, len(cfg.Questions))
	for i, q := range cfg.Questions {
		if _, dup := seen[q.ID]; dup {
			return fmt.Errorf("evals has a duplicate id %q at index %d. Use a unique 'id' for each question. Example:\nevals:\n  - id: readme\n    question: Does the README explain setup?\n  - id: security\n    question: Are secrets handled safely?", q.ID, i)
		}
		seen[q.ID] = struct{}{}

		if strings.TrimSpace(q.Question) == "" {
			return fmt.Errorf("evals question for id %q must be non-empty. Example:\nevals:\n  - id: %s\n    question: Does the README explain setup?", q.ID, q.ID)
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
