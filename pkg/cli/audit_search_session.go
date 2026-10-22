package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	maxAuditSearchSessionBytes = 64 * 1024 * 1024
	maxAuditSearchEvents       = 20000
)

// AuditEvidenceRef identifies an actual JSONL record and its original artifact location.
type AuditEvidenceRef struct {
	Line      int                     `json:"line"`
	ID        string                  `json:"id,omitempty"`
	Type      string                  `json:"type"`
	Timestamp json.RawMessage         `json:"timestamp,omitempty"`
	AgentID   string                  `json:"agent_id,omitempty"`
	Component string                  `json:"component"`
	Phase     string                  `json:"phase,omitempty"`
	Path      string                  `json:"path,omitempty"`
	Index     *int                    `json:"index,omitempty"`
	Native    *AuditEvidenceNativeRef `json:"native,omitempty"`
}

type AuditEvidenceNativeRef struct {
	Path          string `json:"path,omitempty"`
	Index         *int   `json:"index,omitempty"`
	PersistedPath string `json:"persistedPath,omitempty"`
}

type auditSearchSessionEvent struct {
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data"`
	ID         string          `json:"id"`
	AgentID    string          `json:"agentId"`
	Timestamp  json.RawMessage `json:"timestamp"`
	Provenance struct {
		Component string                  `json:"component"`
		Phase     string                  `json:"phase"`
		Path      string                  `json:"path"`
		Index     *int                    `json:"index"`
		Native    *AuditEvidenceNativeRef `json:"native"`
	} `json:"provenance"`
}

func readAuditSearchSession(ctx context.Context, runDir string) (docs []auditSearchDocument, source string, err error) {
	path, err := findUnifiedSessionPath(runDir)
	if err != nil {
		return nil, "", err
	}
	if path == "" {
		return nil, "", errors.New("unified session unavailable; --search requires usage/aw_session.jsonl or aw_session.jsonl")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() || info.Size() > maxAuditSearchSessionBytes {
		return nil, "", errors.New("unified session must be a regular file no larger than 64 MiB for --search")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close unified session search source: %w", closeErr))
		}
	}()
	source, err = filepath.Rel(runDir, path)
	if err != nil {
		return nil, "", err
	}
	docs, err = parseAuditSearchSession(ctx, file)
	if err == nil {
		err = validateAuditSearchStages(docs)
	}
	return docs, filepath.ToSlash(source), err
}

func parseAuditSearchSession(ctx context.Context, input io.Reader) ([]auditSearchDocument, error) {
	limited := &io.LimitedReader{R: input, N: maxAuditSearchSessionBytes + 1}
	reader := bufio.NewReader(limited)
	headerSeen := false
	textBytes := 0
	var docs []auditSearchDocument
	for lineNumber := 1; ; lineNumber++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line, oversized, readErr := readUnifiedSessionLine(reader)
		if limited.N <= 0 {
			return nil, errors.New("unified session exceeds the 64 MiB search input limit")
		}
		if oversized {
			return nil, fmt.Errorf("oversized unified session search record on line %d", lineNumber)
		}
		if len(bytes.TrimSpace(line)) > 0 {
			var event auditSearchSessionEvent
			if err := json.Unmarshal(line, &event); err != nil {
				return nil, fmt.Errorf("invalid unified session search record on line %d", lineNumber)
			}
			if !headerSeen {
				header := unifiedDetectionEvent{Type: event.Type, Data: event.Data}
				header.Provenance.Component = event.Provenance.Component
				if err := validateUnifiedDetectionHeader(header); err != nil {
					return nil, err
				}
				headerSeen = true
			} else {
				if event.Type == "session.format" && event.Provenance.Component == "collector" {
					return nil, errors.New("unified session contains multiple collector format headers")
				}
				if event.Type == "" || event.Provenance.Component == "" || !bytes.HasPrefix(bytes.TrimSpace(event.Data), []byte("{")) {
					return nil, fmt.Errorf("invalid unified session search event on line %d; expected type, data object and provenance", lineNumber)
				}
				if len(docs) >= maxAuditSearchEvents {
					return nil, errors.New("unified session exceeds the 20000-event search input limit")
				}
				doc, err := auditSearchEventDocument(event, lineNumber)
				if err != nil {
					return nil, err
				}
				textBytes += len(doc.text)
				if textBytes > maxAuditSearchSessionBytes {
					return nil, errors.New("decoded unified session search text exceeds 64 MiB")
				}
				docs = append(docs, doc)
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				return nil, fmt.Errorf("read unified session search input: %w", readErr)
			}
			break
		}
	}
	if !headerSeen {
		return nil, errors.New("unified session is missing its leading session.format header")
	}
	return docs, nil
}

func auditSearchEventDocument(event auditSearchSessionEvent, line int) (auditSearchDocument, error) {
	var data any
	decoder := json.NewDecoder(bytes.NewReader(event.Data))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return auditSearchDocument{}, fmt.Errorf("invalid unified session data on line %d", line)
	}
	var text strings.Builder
	if err := flattenAuditSearchData(&text, "", data, 0); err != nil {
		return auditSearchDocument{}, fmt.Errorf("unified session data on line %d: %w", line, err)
	}
	ref := AuditEvidenceRef{
		Line: line, ID: event.ID, Type: event.Type, AgentID: event.AgentID,
		Component: event.Provenance.Component, Phase: event.Provenance.Phase,
		Path: event.Provenance.Path, Index: event.Provenance.Index,
		Native: event.Provenance.Native,
	}
	if len(event.ID) > 128 || len(event.Type) > 128 || len(event.AgentID) > 128 ||
		len(ref.Component) > 128 || len(ref.Phase) > 128 || len(ref.Path) > 512 || len(event.Timestamp) > 128 {
		return auditSearchDocument{}, fmt.Errorf("oversized unified session reference on line %d", line)
	}
	if ref.Native != nil && (len(ref.Native.Path) > 512 || len(ref.Native.PersistedPath) > 512) {
		return auditSearchDocument{}, fmt.Errorf("oversized unified session native reference on line %d", line)
	}
	ref.Timestamp = event.Timestamp
	return auditSearchDocument{
		ref: ref, text: text.String(), outcome: auditSearchOutcome(data),
		stage: strings.Join([]string{ref.Component, ref.Phase, ref.Type}, "/"),
	}, nil
}

// Decode nested string values so escaped Unicode and identifiers remain searchable.
func flattenAuditSearchData(out *strings.Builder, key string, value any, depth int) error {
	if depth > 64 {
		return errors.New("search data exceeds the 64-level nesting limit")
	}
	var scalar string
	switch value := value.(type) {
	case map[string]any:
		for _, child := range slices.Sorted(maps.Keys(value)) {
			if err := flattenAuditSearchData(out, key+"."+child, value[child], depth+1); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range value {
			if err := flattenAuditSearchData(out, key, child, depth+1); err != nil {
				return err
			}
		}
	case string:
		scalar = value
	case json.Number:
		scalar = value.String()
	case bool:
		scalar = strconv.FormatBool(value)
	}
	if scalar != "" {
		if out.Len()+len(key)+len(scalar)+2 > maxUnifiedSessionLineSize {
			return errors.New("decoded search event exceeds the record text limit")
		}
		out.WriteString(key + "=" + scalar + "\n")
	}
	return nil
}
