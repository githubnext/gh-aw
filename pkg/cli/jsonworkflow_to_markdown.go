package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var jsonWorkflowLog = logger.New("cli:jsonworkflow_to_markdown")

// JSONWorkflow is a generic JSON workflow definition for import.
// All fields are optional; unrecognised top-level keys are collected in Extra
// and preserved as a YAML comment block so that no information is silently
// discarded.
type JSONWorkflow struct {
	// Identification
	ID   string `json:"id"`
	Name string `json:"name"`
	// Human-readable description → frontmatter description:
	Description string `json:"description"`
	// Main body / prompt text → markdown body after frontmatter
	Instructions string `json:"instructions"`
	// Preferred AI engine → frontmatter engine:
	Engine string `json:"engine"`
	// Trigger configuration → frontmatter on: (passed through as-is)
	On any `json:"on"`
	// Tags → frontmatter tags:
	Tags []string `json:"tags"`
	// Extra holds any top-level keys not listed above so they can be preserved
	// as a comment block.
	Extra map[string]any `json:"-"`
}

// UnmarshalJSON implements json.Unmarshaler so that unknown keys are captured in Extra.
func (w *JSONWorkflow) UnmarshalJSON(data []byte) error {
	// Decode into the typed fields via a type alias (avoids infinite recursion).
	type jsonWorkflowAlias JSONWorkflow
	var alias jsonWorkflowAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*w = JSONWorkflow(alias)

	// Capture all top-level keys into a raw map.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	knownKeys := map[string]bool{
		"id": true, "name": true, "description": true,
		"instructions": true, "engine": true, "on": true, "tags": true,
	}
	for k, v := range raw {
		if !knownKeys[k] {
			if w.Extra == nil {
				w.Extra = make(map[string]any)
			}
			var decoded any
			if err := json.Unmarshal(v, &decoded); err != nil {
				w.Extra[k] = string(v)
			} else {
				w.Extra[k] = decoded
			}
		}
	}
	return nil
}

// ConvertOptions configures ConvertJSONWorkflowToMarkdown.
type ConvertOptions struct {
	// NameOverride, when non-empty, replaces the filename derived from the JSON.
	NameOverride string
}

// GeneratedWorkflow is the output of ConvertJSONWorkflowToMarkdown.
type GeneratedWorkflow struct {
	// Filename is the kebab-cased base name (without .md extension).
	Filename string
	// Markdown is the complete file content: YAML frontmatter followed by the prompt body.
	Markdown string
	// Warnings lists fields that could not be fully translated.
	Warnings []string
}

// ConvertJSONWorkflowToMarkdown converts a JSONWorkflow into a gh-aw markdown workflow
// file.  The conversion is best-effort and deterministic: any field that cannot be
// mapped to a known frontmatter key or body section is preserved as a YAML comment
// block inside the frontmatter, and a corresponding warning is added to
// GeneratedWorkflow.Warnings.
func ConvertJSONWorkflowToMarkdown(a *JSONWorkflow, opts ConvertOptions) (*GeneratedWorkflow, error) {
	if a == nil {
		return nil, errors.New("JSONWorkflow must not be nil")
	}

	var warnings []string

	// ── Derive filename ─────────────────────────────────────────────────────────
	filename := opts.NameOverride
	if filename == "" {
		filename = filenameFromJSONWorkflow(a)
	}

	jsonWorkflowLog.Printf("Converting JSON workflow: id=%q name=%q filename=%q", a.ID, a.Name, filename)

	// ── Build frontmatter ────────────────────────────────────────────────────────
	var fm strings.Builder
	fm.WriteString("---\n")

	if a.Description != "" {
		fm.WriteString("description: ")
		fm.WriteString(yamlQuoteString(a.Description))
		fm.WriteString("\n")
	}

	if a.Engine != "" {
		fm.WriteString("engine: ")
		fm.WriteString(a.Engine)
		fm.WriteString("\n")
	}

	if a.On != nil {
		onYAML, err := marshalFrontmatterValue(a.On)
		if err == nil {
			fm.WriteString("on:\n")
			for line := range strings.SplitSeq(onYAML, "\n") {
				if line == "" {
					continue
				}
				fm.WriteString("  ")
				fm.WriteString(line)
				fm.WriteString("\n")
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("could not serialize 'on' field: %v", err))
		}
	}

	if len(a.Tags) > 0 {
		fm.WriteString("tags:\n")
		for _, tag := range a.Tags {
			fm.WriteString("  - ")
			fm.WriteString(yamlQuoteString(tag))
			fm.WriteString("\n")
		}
	}

	// Emit unknown fields as YAML comments so the file remains valid YAML while
	// preserving the original data for the operator to inspect.
	if len(a.Extra) > 0 {
		extraYAML, err := marshalFrontmatterValue(a.Extra)
		if err == nil {
			fm.WriteString("# Unsupported fields preserved from source JSON:\n")
			for line := range strings.SplitSeq(extraYAML, "\n") {
				if line == "" {
					continue
				}
				fm.WriteString("# ")
				fm.WriteString(line)
				fm.WriteString("\n")
			}
			// Sort keys for deterministic warning output.
			extraKeys := make([]string, 0, len(a.Extra))
			for k := range a.Extra {
				extraKeys = append(extraKeys, k)
			}
			sort.Strings(extraKeys)
			for _, k := range extraKeys {
				warnings = append(warnings, fmt.Sprintf("field %q has no gh-aw frontmatter equivalent and was preserved as a comment", k))
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("could not serialize unsupported fields: %v", err))
		}
	}

	fm.WriteString("---\n")

	// ── Build body ───────────────────────────────────────────────────────────────
	var body strings.Builder

	// Heading from name (or ID as fallback).
	heading := a.Name
	if heading == "" {
		heading = a.ID
	}
	if heading != "" {
		body.WriteString("# ")
		body.WriteString(heading)
		body.WriteString("\n\n")
	}

	if a.Instructions != "" {
		body.WriteString(strings.TrimRight(a.Instructions, "\n"))
		body.WriteString("\n")
	}

	markdown := fm.String() + "\n" + body.String()

	return &GeneratedWorkflow{
		Filename: filename,
		Markdown: markdown,
		Warnings: warnings,
	}, nil
}

// filenameFromJSONWorkflow derives a kebab-cased filename slug from the workflow's id
// or name fields.
func filenameFromJSONWorkflow(a *JSONWorkflow) string {
	candidate := a.ID
	if candidate == "" {
		candidate = a.Name
	}
	if candidate == "" {
		return "imported-workflow"
	}
	return stringutil.SanitizeForFilename(toKebabCase(candidate))
}

// toKebabCase converts a string to kebab-case:
//   - whitespace and underscores → "-"
//   - sequences of non-alphanumeric chars → single "-"
//   - result is lower-cased
//
// nonAlphanumSeq matches one or more consecutive non-alphanumeric characters.
// It is compiled once at package init because regex compilation is expensive and
// the pattern is immutable.
var nonAlphanumSeq = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func toKebabCase(s string) string {
	// Normalize whitespace and underscores to dashes first.
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = nonAlphanumSeq.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return strings.ToLower(s)
}

// yamlQuoteString wraps s in double quotes if it contains characters that would
// require YAML quoting, otherwise returns it as-is.
func yamlQuoteString(s string) string {
	// Simple heuristic: quote if s contains a colon, hash, newline, or leading/trailing
	// whitespace – all of which require quoting in YAML plain scalars.
	if strings.ContainsAny(s, ":#\n\r") || s != strings.TrimSpace(s) || s == "" {
		// Escape backslashes and double-quotes inside the value, then escape
		// literal newlines/carriage-returns so the result is a valid single-line
		// YAML double-quoted scalar.
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		escaped = strings.ReplaceAll(escaped, "\r", `\r`)
		return `"` + escaped + `"`
	}
	return s
}

// marshalFrontmatterValue serializes v as indented YAML (without the leading "---").
// It uses encoding/json as an intermediate form to avoid importing a YAML library.
func marshalFrontmatterValue(v any) (string, error) {
	// Re-encode to JSON then convert to a minimal YAML representation.
	// This produces valid YAML for the subset of JSON types we care about.
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	// Convert JSON representation to simple YAML.
	// For objects and arrays the JSON encoding is already valid YAML.
	return string(raw), nil
}
