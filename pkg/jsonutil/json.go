package jsonutil

import (
	"bytes"
	"encoding/json"
	"strings"
)

// MarshalCompactNoHTMLEscape marshals a value to compact JSON without HTML escaping.
// It trims the trailing newline emitted by json.Encoder.
func MarshalCompactNoHTMLEscape(v any) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return "", err
	}

	return strings.TrimSuffix(buf.String(), "\n"), nil
}
