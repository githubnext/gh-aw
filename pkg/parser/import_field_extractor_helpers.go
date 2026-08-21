// Package parser provides functions for parsing and processing workflow markdown files.
// import_field_extractor_helpers.go provides shared helper utilities used across
// import-field extractor concern files.
package parser

import "encoding/json"

func mergeJSONStringListField(
	fm map[string]any,
	field, emptyValue string,
	seen map[string]bool,
	merged *[]string,
	extractor func(map[string]any, string) (string, error),
) {
	content, err := extractor(fm, field)
	if err != nil || content == "" || content == emptyValue {
		return
	}
	var imported []string
	if jsonErr := json.Unmarshal([]byte(content), &imported); jsonErr != nil {
		return
	}
	for _, value := range imported {
		if !seen[value] {
			seen[value] = true
			*merged = append(*merged, value)
		}
	}
}

func parseStringSliceField(value any, keepEmpty bool) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, v := range values {
		if s, ok := v.(string); ok {
			if s == "" && !keepEmpty {
				continue
			}
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
