package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var outcomeEvalEnrichLog = logger.New("cli:outcome_eval_enrich")

type rawOutcomeEntry struct {
	Type        string `json:"type"`
	IssueNumber int    `json:"issue_number"`
	ItemNumber  int    `json:"item_number"`
	PullNumber  int    `json:"pull_number"`
}

// enrichItemsFromAgentOutput reads the raw agent output (safeoutputs.jsonl) and fills in
// missing fields like issue_number and item_number that the executed manifest may omit.
func enrichItemsFromAgentOutput(items []CreatedItemReport, runDir string, defaultRepo string) []CreatedItemReport {
	outcomeEvalEnrichLog.Printf("Enriching items from agent output: items=%d, runDir=%s", len(items), runDir)
	rawByType, err := readRawOutcomeEntries(runDir)
	if err != nil {
		outcomeEvalEnrichLog.Printf("Cannot read safeoutputs.jsonl: %v", err)
		return items
	}

	// Match items to raw entries by type and order to fill in missing numbers
	typeCounters := make(map[string]int)
	for i := range items {
		t := items[i].Type
		idx := typeCounters[t]
		typeCounters[t]++

		if items[i].Number > 0 {
			continue // Already has a number
		}

		entries := rawByType[t]
		if idx >= len(entries) {
			continue
		}
		raw := entries[idx]

		num := raw.IssueNumber
		if num == 0 {
			num = raw.ItemNumber
		}
		if num == 0 {
			num = raw.PullNumber
		}
		if num > 0 {
			items[i].Number = num
		}
		if items[i].Repo == "" {
			items[i].Repo = defaultRepo
		}
	}

	return items
}

func readRawOutcomeEntries(runDir string) (map[string][]rawOutcomeEntry, error) {
	rawByType := make(map[string][]rawOutcomeEntry)
	seen := make(map[[sha256.Size]byte]struct{})
	err := filepath.WalkDir(runDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() && path != runDir && artifactMatchesFilter(entry.Name(), []string{"agent-output-fallback", "agent", "safe-outputs-items"}) {
			return filepath.SkipDir
		}
		if entry.IsDir() || entry.Name() != "safeoutputs.jsonl" {
			return nil
		}
		return appendRawOutcomeEntries(path, rawByType, seen)
	})
	return rawByType, err
}

func appendRawOutcomeEntries(path string, entries map[string][]rawOutcomeEntry, seen map[[sha256.Size]byte]struct{}) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if _, mirrored := seen[digest]; mirrored {
		return nil
	}
	seen[digest] = struct{}{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry rawOutcomeEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries[entry.Type] = append(entries[entry.Type], entry)
	}
	return scanner.Err()
}
