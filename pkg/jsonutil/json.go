package jsonutil

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var jsonutilLog = logger.New("jsonutil:json")

// MarshalCompactNoHTMLEscape marshals a value to compact JSON without HTML escaping.
// It trims the trailing newline emitted by json.Encoder.
func MarshalCompactNoHTMLEscape(v any) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		jsonutilLog.Printf("MarshalCompactNoHTMLEscape encode failed: %v", err)
		return "", err
	}

	result := strings.TrimSuffix(buf.String(), "\n")
	jsonutilLog.Printf("MarshalCompactNoHTMLEscape produced %d bytes", len(result))
	return result, nil
}
