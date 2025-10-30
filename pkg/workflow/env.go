package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var envLog = logger.New("workflow:env")

// writeHeadersToYAML writes a map of headers to YAML format with proper comma placement
// indent is the indentation string to use for each header line (e.g., "                  ")
func writeHeadersToYAML(yaml *strings.Builder, headers map[string]string, indent string) {
	if len(headers) == 0 {
		envLog.Print("No headers to write")
		return
	}

	envLog.Printf("Writing %d headers to YAML", len(headers))

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

	envLog.Print("Headers written successfully")
}
