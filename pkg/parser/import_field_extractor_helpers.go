package parser

import (
	"encoding/json"
	"strings"
)

// import_field_extractor_helpers.go contains reusable low-level field extraction
// helpers shared by import field extraction domains.

func (acc *importAccumulator) extractFirstWinsJSONField(fm map[string]any, fullPath, field string, target *string) {
	if *target != "" {
		return
	}
	fieldJSON, err := extractFieldJSONFromMap(fm, field, "")
	if err != nil || fieldJSON == "" || fieldJSON == "null" {
		return
	}
	*target = fieldJSON
	parserLog.Printf("Extracted %s from import: %s", field, fullPath)
}

func (acc *importAccumulator) appendJSONBuilderField(fm map[string]any, field, emptyValue string, builder *strings.Builder) {
	content, err := extractFieldJSONFromMap(fm, field, emptyValue)
	if err != nil || content == "" || content == emptyValue {
		return
	}
	builder.WriteString(content + "\n")
}

func (acc *importAccumulator) appendJSONSliceField(fm map[string]any, field, emptyValue string, target *[]string) {
	content, err := extractFieldJSONFromMap(fm, field, emptyValue)
	if err != nil || content == "" || content == emptyValue {
		return
	}
	*target = append(*target, content)
}

func (acc *importAccumulator) appendYAMLBuilderField(fm map[string]any, field string, builder *strings.Builder) {
	content, err := extractYAMLFieldFromMap(fm, field)
	if err != nil || content == "" {
		return
	}
	builder.WriteString(content + "\n")
}

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
