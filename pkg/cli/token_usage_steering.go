package cli

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/fileutil"
)

func countAPIProxySteeringEvents(runDir string) int {
	eventsPath := findAPIProxyEventsFile(runDir)
	if eventsPath == "" {
		return 0
	}
	count, err := parseAPIProxySteeringEvents(eventsPath)
	if err != nil {
		tokenUsageLog.Printf("Failed to parse API proxy events file %s: %v", eventsPath, err)
		return 0
	}
	return count
}

func findAPIProxyEventsFile(runDir string) string {
	primary := filepath.Join(runDir, "sandbox", "firewall", "logs", proxyEventsJSONLPath)
	if fileutil.FileExists(primary) {
		return primary
	}

	entries, err := os.ReadDir(runDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "firewall-audit-logs") || strings.HasPrefix(name, "firewall-logs") {
			candidate := filepath.Join(runDir, name, proxyEventsJSONLPath)
			if fileutil.FileExists(candidate) {
				return candidate
			}
		}
	}

	return ""
}

// proxyEventsEntry is a JSONL record from api-proxy-logs/events.jsonl.
// The event name appears under one of four field names depending on the proxy version;
// the message field is present on steering events.
type proxyEventsEntry struct {
	// Event name appears under one of these four keys; all are checked.
	Event          string `json:"event"`
	Type           string `json:"type"`
	EventNameSnake string `json:"event_name"`
	EventNameCamel string `json:"eventName"`
	// Message text (present on steering events).
	Message string `json:"message"`
	// Optional RFC3339/RFC3339Nano timestamp (not always present).
	Timestamp string `json:"timestamp"`
}

// eventName returns the normalised event name from whichever field is populated.
func (e proxyEventsEntry) eventName() string {
	for _, v := range []string{e.Event, e.Type, e.EventNameSnake, e.EventNameCamel} {
		if v = strings.TrimSpace(v); v != "" {
			return strings.ToLower(v)
		}
	}
	return ""
}

// scanSteeringEntries reads all valid steering proxyEventsEntry records from r.
// Lines that fail the quick-keyword check or JSON decoding are silently skipped.
// The caller is responsible for the lifetime of r.
func scanSteeringEntries(r io.Reader) ([]proxyEventsEntry, error) {
	var entries []proxyEventsEntry
	scanner := bufio.NewScanner(r)
	buf := make([]byte, maxScannerBufferSize)
	scanner.Buffer(buf, maxScannerBufferSize)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !containsSteeringKeyword(line) {
			continue
		}
		var entry proxyEventsEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if isSteeringEvent(entry.eventName(), strings.TrimSpace(entry.Message)) {
			entries = append(entries, entry)
		}
	}
	return entries, scanner.Err()
}

func parseAPIProxySteeringEvents(filePath string) (int, error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return 0, err
	}
	defer file.Close()
	entries, err := scanSteeringEntries(file)
	return len(entries), err
}

func containsSteeringKeyword(line string) bool {
	return strings.Contains(line, "steering") ||
		strings.Contains(line, "STEERING") ||
		strings.Contains(line, "Steering")
}

// isSteeringEvent matches AWF proxy steering events using both event name and
// message format from the firewall specification.
func isSteeringEvent(eventName, message string) bool {
	switch eventName {
	case tokenSteeringEventName:
		return strings.HasPrefix(message, awfTokenWarningPrefix)
	case timeoutSteeringEventName:
		return strings.HasPrefix(message, awfTimeWarningPrefix)
	default:
		return false
	}
}
