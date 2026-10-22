package aw

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

// Document represents an AW .md source file before imports or compiler defaults
// have been applied. Frontmatter is the authoritative configuration.
type Document struct {
	Frontmatter map[string]any
	Markdown    string
	Location    SourceLocation

	source *documentSource
}

type SourceLocation struct {
	FrontmatterLines []string
	FrontmatterStart int
	FieldLines       map[string]int
}

type documentSource struct {
	header      string
	frontmatter string
}

// Parse accepts main workflows, shared imports, redirect placeholders, and
// plain Markdown. Semantic workflow validation happens during compilation.
func Parse(content string) (*Document, error) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return nil, err
	}
	frontmatter, err := yaml.Marshal(result.Frontmatter)
	if err != nil {
		return nil, fmt.Errorf("snapshotting AW frontmatter: %w", err)
	}
	header, body := splitSource(content, result)
	return &Document{
		Frontmatter: result.Frontmatter,
		Markdown:    body,
		Location: SourceLocation{
			FrontmatterLines: result.FrontmatterLines,
			FrontmatterStart: result.FrontmatterStart,
			FieldLines:       result.FieldLines,
		},
		source: &documentSource{
			header:      header,
			frontmatter: string(frontmatter),
		},
	}, nil
}

func splitSource(content string, result *parser.FrontmatterResult) (string, string) {
	if result.FrontmatterStart == 0 {
		return "", content
	}
	body := content
	for range len(result.FrontmatterLines) + 2 {
		_, remaining, found := strings.Cut(body, "\n")
		body = remaining
		if !found {
			break
		}
	}
	return content[:len(content)-len(body)], body
}

// Emit preserves unchanged regions verbatim. New or edited frontmatter uses
// canonical YAML; Markdown edits are emitted exactly as supplied.
func Emit(document *Document) (string, error) {
	if document == nil {
		return "", errors.New("AW document cannot be nil")
	}
	frontmatter, err := yaml.Marshal(document.Frontmatter)
	if err != nil {
		return "", fmt.Errorf("serializing AW frontmatter: %w", err)
	}
	header := ""
	if document.Location.FrontmatterStart > 0 || len(document.Frontmatter) > 0 ||
		(document.source == nil && document.Frontmatter != nil) {
		header = "---\n" + string(frontmatter) + "---\n"
	}
	if original := document.source; original != nil {
		if string(frontmatter) == original.frontmatter {
			header = original.header
		}
	}
	return header + document.Markdown, nil
}

// FrontmatterResult returns an independent parser view. Compiler preprocessing
// may mutate it without modifying the authored document or its source metadata.
func (document *Document) FrontmatterResult() *parser.FrontmatterResult {
	markdown := document.Markdown
	if document.Location.FrontmatterStart > 0 || len(document.Frontmatter) > 0 {
		markdown = strings.TrimSpace(markdown)
	}
	return &parser.FrontmatterResult{
		Frontmatter:      cloneMapping(document.Frontmatter),
		Markdown:         markdown,
		FrontmatterLines: slices.Clone(document.Location.FrontmatterLines),
		FrontmatterStart: document.Location.FrontmatterStart,
		FieldLines:       maps.Clone(document.Location.FieldLines),
	}
}

func cloneMapping(source map[string]any) map[string]any {
	result := maps.Clone(source)
	for key, value := range result {
		result[key] = cloneValue(value)
	}
	return result
}

func cloneValue(value any) any {
	switch node := value.(type) {
	case map[string]any:
		return cloneMapping(node)
	case map[string]string:
		return maps.Clone(node)
	case []string:
		return slices.Clone(node)
	case []any:
		result := make([]any, 0, len(node))
		for _, item := range node {
			result = append(result, cloneValue(item))
		}
		return result
	default:
		return value
	}
}
