package importinpututil

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ResolvePathValue resolves either a top-level input key ("count") or a one-level
// dotted object sub-key ("config.apiKey") from import inputs.
func ResolvePathValue(inputs map[string]any, inputPath string) (any, bool) {
	top, sub, hasDot := strings.Cut(inputPath, ".")
	if !hasDot {
		value, ok := inputs[top]
		return value, ok
	}
	topVal, topOK := inputs[top]
	if !topOK {
		return nil, false
	}
	obj, isMap := topVal.(map[string]any)
	if !isMap {
		return nil, false
	}
	value, ok := obj[sub]
	return value, ok
}

// FormatResolvedValue formats a resolved import input value for textual
// substitution. []any/map[string]any and typed slices/maps are normalized and
// JSON-marshaled, nil returns ("", false), and scalars use fmt.Sprintf("%v", v).
func FormatResolvedValue(value any) (string, bool) {
	switch v := value.(type) {
	case []any:
		return marshalValue(v)
	case map[string]any:
		return marshalValue(v)
	case nil:
		return "", false
	default:
		return formatReflectiveValue(v)
	}
}

func formatReflectiveValue(value any) (string, bool) {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice:
		return marshalValue(normalizeSlice(rv))
	case reflect.Map:
		return marshalValue(normalizeMap(rv))
	default:
		return fmt.Sprintf("%v", value), true
	}
}

func marshalValue(value any) (string, bool) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func normalizeSlice(rv reflect.Value) []any {
	normalized := make([]any, rv.Len())
	for i := range rv.Len() {
		normalized[i] = rv.Index(i).Interface()
	}
	return normalized
}

func normalizeMap(rv reflect.Value) map[string]any {
	keys := make([]string, 0, rv.Len())
	for _, key := range rv.MapKeys() {
		keys = append(keys, key.String())
	}
	sort.Strings(keys)
	normalized := make(map[string]any, rv.Len())
	for _, k := range keys {
		normalized[k] = rv.MapIndex(reflect.ValueOf(k)).Interface()
	}
	return normalized
}
