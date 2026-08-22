package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// NetworkLogEntry contains fields shared by one-line network request logs.
type NetworkLogEntry struct {
	ClientAddr string `json:"client,omitempty"`
	Method     string `json:"method,omitempty"`
	Status     string `json:"status"`
	Decision   string `json:"decision,omitempty"`
	URL        string `json:"url,omitempty"`
}

func networkStatusCode(status string) (int, bool) {
	statusCode, err := strconv.Atoi(status)
	if err != nil {
		return 0, false
	}
	return statusCode, true
}

func networkStatusCodeOrZero(status string) int {
	statusCode, _ := networkStatusCode(status)
	return statusCode
}

func networkStatusFromJSON(raw json.RawMessage) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return "", nil
	}
	var status string
	if err := json.Unmarshal(raw, &status); err == nil {
		return status, nil
	}

	// The audit schema emits status as an undecorated integer JSON number.
	statusCode, err := strconv.Atoi(trimmed)
	if err != nil || statusCode < 0 {
		return "", fmt.Errorf("invalid network status JSON value: %s", trimmed)
	}
	return strconv.Itoa(statusCode), nil
}
