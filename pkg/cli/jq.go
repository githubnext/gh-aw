package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/itchyny/gojq"
)

var jqLog = logger.New("cli:jq")

// ApplyJqFilter applies a jq filter to JSON input using the gojq library
func ApplyJqFilter(jsonInput string, jqFilter string) (string, error) {
	jqLog.Printf("Applying jq filter: %s (input size: %d bytes)", jqFilter, len(jsonInput))

	// Validate filter is not empty
	if jqFilter == "" {
		return "", fmt.Errorf("jq filter cannot be empty")
	}

	// Parse the jq query
	query, err := gojq.Parse(jqFilter)
	if err != nil {
		jqLog.Printf("failed to parse jq filter: %v", err)
		return "", fmt.Errorf("failed to parse jq filter: %w", err)
	}

	// Parse the JSON input
	var input any
	dec := json.NewDecoder(strings.NewReader(jsonInput))
	if err := dec.Decode(&input); err != nil {
		jqLog.Printf("failed to parse JSON input: %v", err)
		return "", fmt.Errorf("failed to parse JSON input: %w", err)
	}

	// Run the query
	var results []string
	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			jqLog.Printf("jq filter execution error: %v", err)
			return "", fmt.Errorf("jq filter execution error: %w", err)
		}

		// Marshal result to JSON
		result, err := json.Marshal(v)
		if err != nil {
			jqLog.Printf("failed to marshal result: %v", err)
			return "", fmt.Errorf("failed to marshal result: %w", err)
		}
		results = append(results, string(result))
	}

	// Join results with newlines (matching jq behavior)
	output := strings.Join(results, "\n")
	if len(results) > 0 {
		output += "\n"
	}

	jqLog.Printf("jq filter succeeded (output size: %d bytes)", len(output))
	return output, nil
}
