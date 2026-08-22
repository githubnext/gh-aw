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
	if statusCode, err := strconv.Atoi(status); err == nil {
		return statusCode, true
	}

	// Squid access logs combine the proxy decision and HTTP status (for example,
	// "TCP_MISS/200"). Keep the shared parser capable of handling that format
	// even when the current numeric-status call sites use plain status codes.
	if idx := strings.LastIndex(status, "/"); idx != -1 && idx+1 < len(status) {
		if statusCode, err := strconv.Atoi(status[idx+1:]); err == nil {
			return statusCode, true
		}
	}

	return 0, false
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
