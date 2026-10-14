//go:build !integration

package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectUnifiedSteeringEventTimestamp(t *testing.T) {
	tests := []struct {
		name       string
		clocks     string
		provenance string
		want       string
	}{
		{"timestamp string", `"timestamp":"2026-10-10T10:00:00.123Z"`, "", "2026-10-10T10:00:00.123Z"},
		{"ts string", `"ts":"2026-10-10T10:00:01Z"`, "", "2026-10-10T10:00:01Z"},
		{"time string", `"time":"2026-10-10T10:00:02Z"`, "", "2026-10-10T10:00:02Z"},
		{"timestamp precedence", `"timestamp":"2026-10-10T10:00:00Z","ts":"2026-10-10T10:00:01Z","time":"2026-10-10T10:00:02Z"`, "", "2026-10-10T10:00:00Z"},
		{"ts precedence", `"ts":"2026-10-10T10:00:01Z","time":"2026-10-10T10:00:02Z"`, "", "2026-10-10T10:00:01Z"},
		{"null timestamp", `"timestamp":null,"ts":"2026-10-10T10:00:01Z"`, "", "2026-10-10T10:00:01Z"},
		{"empty timestamp", `"timestamp":"","time":"2026-10-10T10:00:02Z"`, "", "2026-10-10T10:00:02Z"},
		{"milliseconds timestamp", `"timestamp":1000`, "", "1970-01-01T00:00:01Z"},
		{"milliseconds time", `"time":1250`, "", "1970-01-01T00:00:01.25Z"},
		{"seconds ts", `"ts":1.25`, `,"timestampUnit":"seconds"`, "1970-01-01T00:00:01.25Z"},
		{"normalized seconds ts", `"ts":1.25`, `,"timestampMs":1250`, "1970-01-01T00:00:01.25Z"},
		{"normalized seconds time", `"time":1.25`, `,"timestampMs":1250`, "1970-01-01T00:00:01.25Z"},
		{"epoch zero", `"ts":0`, `,"timestampMs":0`, "1970-01-01T00:00:00Z"},
		{"null normalized clock", `"ts":1250`, `,"timestampMs":null`, "1970-01-01T00:00:01.25Z"},
		{"payload timestamp retained", `"timestamp":null`, "", "2026-10-10T09:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event sessionSubagentEvent
			require.NoError(t, json.Unmarshal([]byte(`{"type":"firewall.steering",`+tt.clocks+
				`,"provenance":{"component":"firewall","phase":"agent"`+tt.provenance+
				`},"data":{"event":"token_steering","message":"[AWF TOKEN WARNING] Low tokens","timestamp":"2026-10-10T09:00:00Z"}}`), &event))
			log := &apiProxySteeringLog{eventCounts: make(map[string]int)}
			collectUnifiedSteeringEvent(log, event)
			require.Equal(t, map[string]int{"token_steering": 1}, log.eventCounts)
			events := gatewaySteeringEventsFromEntries(log.entries)
			require.Len(t, events, 1)
			require.Equal(t, tt.want, events[0].Timestamp)
		})
	}
}
