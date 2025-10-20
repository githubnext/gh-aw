package workflow

import (
	"fmt"
	"sort"
	"strings"
)

// writeHeadersToYAML writes a map of headers to YAML format with proper comma placement
// indent is the indentation string to use for each header line (e.g., "                  ")
func writeHeadersToYAML(yaml *strings.Builder, headers map[string]string, indent string) {
	if len(headers) == 0 {
		return
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Write each header with proper comma placement
	for i, key := range keys {
		value := headers[key]
		if i < len(keys)-1 {
			// Not the last header, add comma
			fmt.Fprintf(yaml, "%s\"%s\": \"%s\",\n", indent, key, value)
		} else {
			// Last header, no comma
			fmt.Fprintf(yaml, "%s\"%s\": \"%s\"\n", indent, key, value)
		}
	}
}
