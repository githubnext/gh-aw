package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strconv"
	"unicode/utf8"
)

// Reject duplicate keys before decoding: provider signatures authenticate bytes,
// not a parser's choice between conflicting interpretations.
func parseJSON(data []byte, target any) error {
	if !utf8.Valid(data) {
		return errors.New("invalid UTF-8 JSON")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := checkJSON(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return errors.New("invalid JSON shape")
	}
	return nil
}

func checkJSON(d *json.Decoder, depth int) error {
	if depth > 64 {
		return errors.New("JSON nesting limit exceeded")
	}
	t, err := d.Token()
	if err != nil {
		return errors.New("invalid JSON")
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	keys := map[string]struct{}{}
	for d.More() {
		if delim == '{' {
			key, err := d.Token()
			if err != nil {
				return errors.New("invalid JSON object")
			}
			name, ok := key.(string)
			_, duplicate := keys[name]
			if !ok || duplicate {
				return errors.New("duplicate JSON key")
			}
			keys[name] = struct{}{}
		}
		if err := checkJSON(d, depth+1); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}

func valuesAt(v any, path []string) []any {
	if len(path) > 0 {
		if a, ok := v.([]any); ok {
			if path[0] == "*" {
				var result []any
				for _, item := range a {
					result = append(result, valuesAt(item, path[1:])...)
				}
				return result
			}
			i, err := strconv.Atoi(path[0])
			if err == nil && i >= 0 && i < len(a) {
				return valuesAt(a[i], path[1:])
			}
		}
		if m, ok := v.(map[string]any); ok {
			return valuesAt(m[path[0]], path[1:])
		}
		return nil
	}
	if v == nil {
		return nil
	}
	return []any{v}
}

func at(v any, path ...string) any {
	values := valuesAt(v, path)
	if len(values) > 0 {
		if len(values) > 1 {
			return nil
		}
		return values[0]
	}
	return nil
}

func text(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func decimal(v any) string {
	n, ok := v.(json.Number)
	if !ok {
		return ""
	}
	i, err := n.Int64()
	if err != nil || i <= 0 || i > 9007199254740991 {
		return ""
	}
	return strconv.FormatInt(i, 10)
}

func equal(a, b any) bool {
	// All decoded numeric values use json.Number, retaining type-sensitive
	// comparisons without converting immutable identities to float64.
	switch v := a.(type) {
	case string:
		w, ok := b.(string)
		return ok && v == w
	case bool:
		w, ok := b.(bool)
		return ok && v == w
	case json.Number:
		w, ok := b.(json.Number)
		return ok && v == w
	}
	return false
}

func distinct(values []any) []any {
	result := []any{}
	for _, value := range values {
		found := false
		for _, prior := range result {
			found = found || equal(value, prior)
		}
		if !found {
			result = append(result, value)
		}
	}
	slices.SortFunc(result, func(a, b any) int {
		return cmp.Compare(scalarKey(a), scalarKey(b))
	})
	return result
}

func scalarKey(v any) string {
	switch value := v.(type) {
	case string:
		return "s" + value
	case bool:
		return "b" + strconv.FormatBool(value)
	case json.Number:
		return "n" + value.String()
	}
	return ""
}
