package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var stdinLog = logger.New("cli:stdin")

// readRunIDsFromStdin reads workflow run IDs or URLs from r, one per line.
// Blank lines and lines starting with '#' are ignored.
func readRunIDsFromStdin(r io.Reader) ([]string, error) {
	stdinLog.Print("Reading run IDs from stdin")
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
		stdinLog.Printf("Error while reading stdin: %v", err)
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}
	stdinLog.Printf("Read %d run ID(s) from stdin", len(runIDs))
	return runIDs, nil
}
