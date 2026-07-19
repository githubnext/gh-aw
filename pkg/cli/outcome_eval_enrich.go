package cli

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var outcomeEvalEnrichLog = logger.New("cli:outcome_eval_enrich")

type enrichItemsFromAgentOutputRawEntry struct {
	Type        string `json:"type"`
	IssueNumber int    `json:"issue_number"`
	ItemNumber  int    `json:"item_number"`
	PullNumber  int    `json:"pull_number"`
	Branch      string `json:"branch"`
	Title       string `json:"title"`
}

// enrichItemsFromAgentOutput reads the raw agent output (safeoutputs.jsonl) and fills in
// missing fields like issue_number and item_number that the executed manifest may omit.
func enrichItemsFromAgentOutput(items []CreatedItemReport, runDir string, defaultRepo string) []CreatedItemReport {
	outcomeEvalEnrichLog.Printf("Enriching items from agent output: items=%d, runDir=%s", len(items), runDir)
	rawPath := filepath.Join(runDir, "safeoutputs.jsonl")
	f, err := os.Open(rawPath)
	if err != nil {
		outcomeEvalEnrichLog.Printf("No safeoutputs.jsonl available at %s: %v", rawPath, err)
		return items
	}
	defer f.Close()

	// Parse raw entries to extract issue/item numbers by type and order
	rawByType := enrichItemsFromAgentOutputReadRaw(f)
	outcomeEvalEnrichLog.Printf("Parsed safeoutputs.jsonl: %d distinct types", len(rawByType))
	return enrichItemsFromAgentOutputApply(items, rawByType, defaultRepo)
}

func enrichItemsFromAgentOutputReadRaw(f *os.File) map[string][]enrichItemsFromAgentOutputRawEntry {
	var rawByType = make(map[string][]enrichItemsFromAgentOutputRawEntry)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry enrichItemsFromAgentOutputRawEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		rawByType[entry.Type] = append(rawByType[entry.Type], entry)
	}
	return rawByType
}

func enrichItemsFromAgentOutputApply(items []CreatedItemReport, rawByType map[string][]enrichItemsFromAgentOutputRawEntry, defaultRepo string) []CreatedItemReport {
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
