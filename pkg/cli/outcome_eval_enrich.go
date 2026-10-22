package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var outcomeEvalEnrichLog = logger.New("cli:outcome_eval_enrich")

type agentOutputEntry struct {
	Type        string `json:"type"`
	IssueNumber int    `json:"issue_number"`
	ItemNumber  int    `json:"item_number"`
	PullNumber  int    `json:"pull_number"`
	Branch      string `json:"branch"`
	Title       string `json:"title"`
}

// enrichItemsFromAgentOutput reads the raw agent output (safeoutputs.jsonl) and fills in
// missing fields like issue_number and item_number that the executed manifest may omit.
func enrichItemsFromAgentOutput(items []CreatedItemReport, runDir string, defaultRepo string, additionalDirs ...string) []CreatedItemReport {
	outcomeEvalEnrichLog.Printf("Enriching items from agent output: items=%d, runDir=%s", len(items), runDir)
	rawByType := readAgentOutputEntries(runDir, additionalDirs)
	typeCounters := make(map[string]int)
	for i := range items {
		t := items[i].Type
		idx := typeCounters[t]
		typeCounters[t]++
		if items[i].Number > 0 {
			continue
		}
		entries := rawByType[t]
		if idx < 0 || idx >= len(entries) {
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

func readAgentOutputEntries(runDir string, additionalDirs []string) map[string][]agentOutputEntry {
	rawByType := make(map[string][]agentOutputEntry)
	seenPaths := make(map[string]struct{})
	seenContents := make(map[[sha256.Size]byte]struct{})
	dirs := append([]string{runDir}, additionalDirs...)
	for _, dir := range dirs {
		rawPath := filepath.Join(dir, "safeoutputs.jsonl")
		if _, ok := seenPaths[rawPath]; ok {
			continue
		}
		seenPaths[rawPath] = struct{}{}
		content, err := os.ReadFile(rawPath)
		if err != nil {
			continue
		}
		digest := sha256.Sum256(content)
		if _, ok := seenContents[digest]; ok {
			continue
		}
		seenContents[digest] = struct{}{}
		scanner := bufio.NewScanner(bytes.NewReader(content))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var entry agentOutputEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}
			rawByType[entry.Type] = append(rawByType[entry.Type], entry)
		}
		if err := scanner.Err(); err != nil {
			outcomeEvalEnrichLog.Printf("Failed reading safeoutputs.jsonl at %s: %v", rawPath, err)
		}
	}
	outcomeEvalEnrichLog.Printf("Parsed safeoutputs.jsonl: %d distinct types", len(rawByType))
	return rawByType
}
