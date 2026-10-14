package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

func unifiedSteeringTimestamp(event sessionSubagentEvent) string {
	for _, raw := range []json.RawMessage{event.Timestamp, event.TS, event.Time} {
		raw = bytes.TrimSpace(raw)
		if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
			continue
		}
		if bytes.HasPrefix(raw, []byte(`"`)) {
			var timestamp string
			if json.Unmarshal(raw, &timestamp) == nil && strings.TrimSpace(timestamp) != "" {
				return timestamp
			}
			continue
		}
		var number float64
		if json.Unmarshal(raw, &number) != nil {
			continue
		}
		// The collector persists the source-unit conversion in provenance.
		// This covers seconds-based audit streams without guessing from magnitude.
		if normalized := bytes.TrimSpace(event.Provenance.TimestampMs); len(normalized) > 0 &&
			!bytes.Equal(normalized, []byte("null")) && json.Unmarshal(normalized, &number) == nil {
			event.Timestamp = normalized
			event.Provenance.TimestampUnit = "milliseconds"
		} else {
			event.Timestamp = raw
		}
		if timestamp, err := subagentSessionStart(event); err == nil {
			return timestamp.UTC().Format(time.RFC3339Nano)
		}
	}
	return ""
}
