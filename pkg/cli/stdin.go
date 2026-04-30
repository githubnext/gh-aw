package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// readRunIDsFromStdin reads workflow run IDs or URLs from r, one per line.
// Blank lines and lines starting with '#' are ignored.
func readRunIDsFromStdin(r io.Reader) ([]string, error) {
	var runIDs []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		runIDs = append(runIDs, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}
	return runIDs, nil
}
