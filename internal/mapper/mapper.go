package mapper

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
)

// MapErrorToSpans tries to map a JSON Schema error (instancePath + meta) to YAML spans.
// It returns one or more candidate spans ordered by confidence.
func MapErrorToSpans(yamlBytes []byte, instancePath string, meta ErrorMeta) ([]Span, error) {
	segments, err := decodeJSONPointer(instancePath)
	if err != nil {
		return nil, err
	}

	// Parse YAML with goccy/go-yaml to get AST with positions.
	file, err := parser.ParseBytes(yamlBytes, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("yaml parse error: %w", err)
	}

	// Start traversal from root document node
	if len(file.Docs) == 0 {
		return []Span{documentFallbackSpan()}, nil
	}

	root := file.Docs[0].Body

	// Attempt exact traversal using segments
	node, parent, _ := traverseBySegments(root, segments)

	// Handle different error kinds based on whether we found the node
	switch meta.Kind {
	case "type":
		if node != nil {
			if valueSpan, ok := valueNodeSpan(node); ok {
				valueSpan.Reason = "type mismatch: highlighting value"
				return []Span{valueSpan}, nil
			}
			// fallback to node span
			return []Span{nodeSpan(node, 0.9, "type mismatch: highlighting node")}, nil
		}

	case "additionalProperties":
		if node != nil {
			// meta.Property often holds the offending key
			if meta.Property != "" {
				if keyNode := findKeyInMapping(parent, meta.Property); keyNode != nil {
					return []Span{nodeSpan(keyNode, 0.98, "additional property key")}, nil
				}
			}
			return []Span{nodeSpan(node, 0.6, "additionalProperties fallback")}, nil
		}

	case "required":
		// For required properties, the node typically won't exist
		// Try to find the parent mapping that should contain this property
		if len(segments) > 0 {
			parentSegments := segments[:len(segments)-1]
			if parentNode, _, _ := traverseBySegments(root, parentSegments); parentNode != nil {
				anchor := computeInsertionAnchor(parentNode, meta.Property)
				return []Span{anchor}, nil
			}
		}
		// If no specific parent found, try the provided parent from traversal
		if parent != nil {
			anchor := computeInsertionAnchor(parent, meta.Property)
			return []Span{anchor}, nil
		}

	default:
		if node != nil {
			return []Span{nodeSpan(node, 0.8, "generic mapping")}, nil
		}
	}

	// If node not found, use fallback heuristics:
	// - try to find nearest existing sibling by name
	// - search for meta.Property literal in text
	// - return parent mapping insertion position
	candidates := fallbackHeuristics(file, yamlBytes, segments, meta)
	if len(candidates) > 0 {
		return candidates, nil
	}

	// As last resort, return full document span with low confidence
	return []Span{documentFallbackSpan()}, nil
}

// traverseBySegments walks the AST using segments. Returns (node, parentNode, parentKeyNode).
// node is the AST node for the final segment (value node); parent is its parent mapping/sequence node.
// parentKeyNode is the key node within the parent mapping if applicable.
func traverseBySegments(root ast.Node, segments []string) (ast.Node, ast.Node, ast.Node) {
	current := root
	var parent ast.Node
	var parentKey ast.Node

	for _, segment := range segments {
		parent = current
		parentKey = nil

		switch node := current.(type) {
		case *ast.MappingNode:
			found := false
			for _, valueNode := range node.Values {
				if keyMatches(valueNode.Key, segment) {
					current = valueNode.Value
					parentKey = valueNode.Key
					found = true
					break
				}
			}
			if !found {
				return nil, parent, parentKey
			}

		case *ast.SequenceNode:
			if !isIndex(segment) {
				return nil, parent, parentKey
			}
			idx := parseIndex(segment)
			if idx < 0 || idx >= len(node.Values) {
				return nil, parent, parentKey
			}
			// SequenceNode.Values contains Node directly, not SequenceEntryNode
			current = node.Values[idx]

		default:
			// Can't traverse further
			return nil, parent, parentKey
		}
	}

	return current, parent, parentKey
}

// keyMatches checks if a mapping key node matches the expected segment string
func keyMatches(keyNode ast.MapKeyNode, segment string) bool {
	switch key := keyNode.(type) {
	case *ast.StringNode:
		return key.Value == segment
	case *ast.MappingKeyNode:
		return key.Value.GetToken().Value == segment
	default:
		// Try to get the token value for other key types
		if token := key.GetToken(); token != nil {
			return token.Value == segment
		}
		return false
	}
}

// parseIndex safely parses a segment as an integer index
func parseIndex(segment string) int {
	if i, err := parseSegmentAsIndex(segment); err == nil {
		return i
	}
	return -1
}

// parseSegmentAsIndex parses a segment as an array index (helper for parseIndex)
func parseSegmentAsIndex(segment string) (int, error) {
	// This is a simple wrapper around the existing isIndex logic
	// Using the same function from pointer.go
	if !isIndex(segment) {
		return -1, fmt.Errorf("not an index")
	}
	// Since isIndex uses strconv.Atoi, we can safely use it here too
	return mustParseInt(segment), nil
}

// mustParseInt parses an integer, panicking on error (only called after isIndex check)
func mustParseInt(s string) int {
	// Since we've already validated with isIndex, this should never fail
	if i, err := parseIntFromString(s); err == nil {
		return i
	}
	return 0 // fallback, though this should never happen
}

// parseIntFromString is a helper to parse integer from string
func parseIntFromString(s string) (int, error) {
	// Use the same logic as in pointer.go isIndex
	return parseIntHelper(s)
}

// parseIntHelper implements the integer parsing logic
func parseIntHelper(segment string) (int, error) {
	// Import strconv locally to match pointer.go pattern
	result := 0
	for _, r := range segment {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid digit")
		}
		digit := int(r - '0')
		result = result*10 + digit
		// Prevent overflow for very large numbers
		if result > 1000000 { // reasonable limit for array indices
			return 0, fmt.Errorf("index too large")
		}
	}
	return result, nil
}

// valueNodeSpan tries to map an AST node to a Span that highlights the value token.
func valueNodeSpan(node ast.Node) (Span, bool) {
	if token := node.GetToken(); token != nil {
		return tokenToSpan(token, 0.95, "exact value node"), true
	}
	return Span{}, false
}

// nodeSpan builds a Span from AST node positions with confidence and reason.
func nodeSpan(node ast.Node, conf float64, reason string) Span {
	if token := node.GetToken(); token != nil {
		return tokenToSpan(token, conf, reason)
	}
	// Fallback span if no token available
	return Span{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 1, Confidence: conf * 0.5, Reason: reason + " (no position)"}
}

// tokenToSpan converts a token to a Span
func tokenToSpan(token *token.Token, confidence float64, reason string) Span {
	pos := token.Position
	return Span{
		StartLine:  pos.Line,
		StartCol:   pos.Column,
		EndLine:    pos.Line,
		EndCol:     pos.Column + len(token.Value),
		Confidence: confidence,
		Reason:     reason,
	}
}

// findKeyInMapping searches mapping children for key; return the key AST node if found.
func findKeyInMapping(parent ast.Node, key string) ast.Node {
	if mappingNode, ok := parent.(*ast.MappingNode); ok {
		for _, valueNode := range mappingNode.Values {
			if keyMatches(valueNode.Key, key) {
				return valueNode.Key
			}
		}
	}
	return nil
}

// computeInsertionAnchor determines where a missing key would be inserted:
// Prefer after last child key in parent mapping: return a Span with that location and confidence.
func computeInsertionAnchor(parent ast.Node, propertyName string) Span {
	if mappingNode, ok := parent.(*ast.MappingNode); ok {
		if len(mappingNode.Values) > 0 {
			// Position after the last mapping value
			lastValue := mappingNode.Values[len(mappingNode.Values)-1]
			if token := lastValue.Value.GetToken(); token != nil {
				pos := token.Position
				return Span{
					StartLine:  pos.Line + 1,
					StartCol:   pos.Column,
					EndLine:    pos.Line + 1,
					EndCol:     pos.Column,
					Confidence: 0.75,
					Reason:     fmt.Sprintf("insertion anchor for missing property '%s'", propertyName),
				}
			}
		}
		// If no values, insert at mapping start
		if token := mappingNode.GetToken(); token != nil {
			pos := token.Position
			return Span{
				StartLine:  pos.Line,
				StartCol:   pos.Column + 1,
				EndLine:    pos.Line,
				EndCol:     pos.Column + 1,
				Confidence: 0.7,
				Reason:     fmt.Sprintf("empty mapping insertion anchor for '%s'", propertyName),
			}
		}
	}

	// Fallback
	return Span{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 1, Confidence: 0.3, Reason: "insertion anchor fallback"}
}

// fallbackHeuristics uses text search for meta.Property and heuristic scoring,
// but prefer AST nodes if found.
func fallbackHeuristics(file *ast.File, yamlBytes []byte, segments []string, meta ErrorMeta) []Span {
	var candidates []Span

	// Try to find property name in the text if available
	if meta.Property != "" {
		if spans := searchPropertyInText(yamlBytes, meta.Property); len(spans) > 0 {
			candidates = append(candidates, spans...)
		}
	}

	// Try to find the closest parent that exists
	if len(segments) > 0 {
		for i := len(segments) - 1; i > 0; i-- {
			parentSegments := segments[:i]
			if parentNode, _, _ := traverseBySegments(file.Docs[0].Body, parentSegments); parentNode != nil {
				span := nodeSpan(parentNode, 0.4, fmt.Sprintf("parent context for missing segments at depth %d", i))
				candidates = append(candidates, span)
				break
			}
		}
	}

	return candidates
}

// searchPropertyInText searches for property names in the YAML text and returns candidate spans
func searchPropertyInText(yamlBytes []byte, property string) []Span {
	content := string(yamlBytes)
	lines := strings.Split(content, "\n")
	var spans []Span

	for lineNum, line := range lines {
		if idx := strings.Index(line, property); idx != -1 {
			span := Span{
				StartLine:  lineNum + 1,
				StartCol:   idx + 1,
				EndLine:    lineNum + 1,
				EndCol:     idx + len(property) + 1,
				Confidence: 0.6,
				Reason:     fmt.Sprintf("text search match for property '%s'", property),
			}
			spans = append(spans, span)
		}
	}

	return spans
}

// documentFallbackSpan returns a low-confidence span covering the entire document
func documentFallbackSpan() Span {
	return Span{
		StartLine:  1,
		StartCol:   1,
		EndLine:    1,
		EndCol:     1,
		Confidence: 0.2,
		Reason:     "document-level fallback",
	}
}
