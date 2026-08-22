package cli

import (
	"encoding/json"
	"strconv"
	"strings"
)

// NetworkLogEntry contains fields shared by one-line network request logs.
type NetworkLogEntry struct {
	Client   string `json:"client,omitempty"`
	Method   string `json:"method,omitempty"`
	Status   string `json:"status"`
	Decision string `json:"decision,omitempty"`
	URL      string `json:"url,omitempty"`
}

func networkStatusCode(status string) (int, bool) {
	if statusCode, err := strconv.Atoi(status); err == nil {
		return statusCode, true
	}

	if idx := strings.LastIndex(status, "/"); idx != -1 && idx+1 < len(status) {
		if statusCode, err := strconv.Atoi(status[idx+1:]); err == nil {
			return statusCode, true
		}
	}

	return 0, false
}

func networkStatusFromJSON(raw json.RawMessage) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, "\"") {
		var status string
		if err := json.Unmarshal(raw, &status); err != nil {
			return "", err
		}
		return status, nil
	}
	return trimmed, nil
}
