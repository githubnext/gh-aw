package cli

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/agentdrain"
)

// Preserve explicit outcomes even when Drain3 masks their distinguishing values.
func auditSearchOutcome(data any) string {
	object, ok := data.(map[string]any)
	if !ok {
		return ""
	}
	fields := make(map[string]string)
	for _, key := range []string{"success", "isError", "exitCode", "status", "result", "decision", "event", "toolName"} {
		if value, ok := auditSearchOutcomeScalar(object[key]); ok {
			fields[key] = value
		}
	}
	if object["error"] != nil {
		fields["error.present"] = "true"
	}
	for _, pair := range [][2]string{{"error", "code"}, {"result", "kind"}, {"result", "isError"}} {
		if nested, ok := object[pair[0]].(map[string]any); ok {
			if value, ok := auditSearchOutcomeScalar(nested[pair[1]]); ok {
				fields[pair[0]+"."+pair[1]] = value
			}
		}
	}
	// String lengths make separators in untrusted values unambiguous.
	var out strings.Builder
	for _, key := range []string{"success", "isError", "exitCode", "status", "result", "decision", "event", "toolName", "error.present", "error.code", "result.kind", "result.isError"} {
		if value, ok := fields[key]; ok {
			out.WriteString(key + ":" + strconv.Itoa(len(value)) + ":" + value + ";")
		}
	}
	return out.String()
}

func auditSearchOutcomeScalar(value any) (string, bool) {
	switch value := value.(type) {
	case string:
		return "string:" + value, true
	case bool:
		return "bool:" + strconv.FormatBool(value), true
	case json.Number:
		return "number:" + value.String(), true
	}
	return "", false
}

func validateAuditSearchStages(docs []auditSearchDocument) error {
	stages := make(map[string]struct{})
	for _, doc := range docs {
		stages[doc.stage] = struct{}{}
		if len(stages) > 256 {
			return errors.New("unified session exceeds the 256-stage search input limit")
		}
	}
	return nil
}

func partitionAuditEvidenceOutcomes(docs []auditSearchDocument, clusters map[string]agentdrain.Cluster) map[string]agentdrain.Cluster {
	partitions := make(map[string]agentdrain.Cluster)
	ids := make(map[string]int)
	next := make(map[string]int)
	for i := range docs {
		doc := &docs[i]
		original := auditEvidenceClusterKey(*doc)
		key := original + "\n" + doc.outcome
		id, ok := ids[key]
		if !ok {
			next[doc.stage]++
			id = next[doc.stage]
			ids[key] = id
		}
		doc.clusterID = id
		partitionKey := auditEvidenceClusterKey(*doc)
		cluster, ok := partitions[partitionKey]
		if !ok {
			cluster = clusters[original]
			cluster.ID = id
			cluster.Size = 0
		}
		cluster.Size++
		partitions[partitionKey] = cluster
	}
	return partitions
}

func auditEvidenceCrowded(docs []auditSearchDocument, ranked []auditSearchRank, limit int) bool {
	if len(ranked) <= limit {
		return false
	}
	seen := make(map[string]struct{})
	for _, hit := range ranked[:limit+1] {
		doc := docs[hit.index] //nolint:uncheckedsliceindex // ranks are produced from this document slice
		key := doc.stage + "\n" + doc.outcome
		if _, ok := seen[key]; ok {
			return true
		}
		seen[key] = struct{}{}
	}
	return false
}

// Mining is only useful if similar candidates could crowd the requested slots.
// Scores and their order are fixed before any clustering.
func groupAuditEvidence(ctx context.Context, docs []auditSearchDocument, ranked []auditSearchRank, query string, limit int) (map[string]agentdrain.Cluster, error) {
	var mined map[string]agentdrain.Cluster
	enabled := make(map[string]struct{})
	if auditEvidenceCrowded(docs, ranked, limit) {
		var err error
		mined, err = clusterAuditEvidence(ctx, docs)
		if err != nil {
			return nil, err
		}
		enabled = auditEvidenceUsefulGroups(docs, ranked[:limit+1], mined, query)
	}
	clusters := make(map[string]agentdrain.Cluster)
	for i := range docs {
		doc := &docs[i]
		key := auditEvidenceClusterKey(*doc)
		if _, ok := enabled[key]; ok {
			clusters[key] = mined[key]
			continue
		}
		doc.clusterID = 0
		doc.anomaly = 0
		key = auditEvidenceClusterKey(*doc)
		cluster := clusters[key]
		cluster.Size++
		clusters[key] = cluster
	}
	return clusters, nil
}

func auditEvidenceUsefulGroups(docs []auditSearchDocument, candidates []auditSearchRank, clusters map[string]agentdrain.Cluster, query string) map[string]struct{} {
	counts := make(map[string]int)
	savedBytes := make(map[string]int)
	for _, hit := range candidates {
		doc := docs[hit.index] //nolint:uncheckedsliceindex // ranks are produced from this document slice
		key := auditEvidenceClusterKey(doc)
		counts[key]++
		if counts[key] > 1 {
			savedBytes[key] += len(auditEvidenceExcerpt(doc, query).Text)
		}
	}
	enabled := make(map[string]struct{})
	for key, count := range counts {
		templateBytes := min(len(strings.Join(clusters[key].Template, " ")), auditSearchExcerptBytes)
		if count > 1 && savedBytes[key] > templateBytes {
			enabled[key] = struct{}{}
		}
	}
	return enabled
}
