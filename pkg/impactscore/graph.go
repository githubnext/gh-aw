package impactscore

import (
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strings"
)

var nodeIDUnsafePattern = regexp.MustCompile(`[^a-z0-9._:/-]+`)

// BuildGraph converts normalized work items and workflow definitions into a
// typed graph. The result is deterministic: nodes and edges are stable-sorted.
func BuildGraph(items []WorkItem, workflows []WorkflowDefinition) Graph {
	builder := graphBuilder{
		graph: Graph{
			Nodes: map[string]Node{},
			Out:   map[string][]Edge{},
			In:    map[string][]Edge{},
			Items: map[string]WorkItem{},
		},
	}

	workflowByName := map[string]string{}
	workflowsWithTitlePrefix := []WorkflowDefinition{}
	for _, workflow := range workflows {
		workflowID := safeNodeID("workflow", workflow.Name)
		workflowByName[workflow.Name] = workflowID
		if workflow.TitlePrefix != "" {
			workflowsWithTitlePrefix = append(workflowsWithTitlePrefix, workflow)
		}
		builder.addNode(Node{ID: workflowID, Type: "agentic_workflow", Name: workflow.Name, Title: firstNonEmpty(workflow.SourcePath, workflow.Path)})
		for _, trigger := range cleanStrings(workflow.Triggers) {
			triggerID := safeNodeID("workflow_trigger", trigger)
			builder.addNode(Node{ID: triggerID, Type: "workflow_trigger", Name: trigger})
			builder.addEdge(workflowID, triggerID, "TRIGGERED_BY", trigger)
		}
		for _, tool := range cleanStrings(workflow.Tools) {
			toolID := safeNodeID("workflow_tool", tool)
			builder.addNode(Node{ID: toolID, Type: "workflow_tool", Name: tool})
			builder.addEdge(workflowID, toolID, "USES_TOOL", tool)
		}
		for _, output := range cleanStrings(workflow.Outputs) {
			outputID := safeNodeID("workflow_output", output)
			builder.addNode(Node{ID: outputID, Type: "workflow_output", Name: output})
			builder.addEdge(workflowID, outputID, "PRODUCES_OUTPUT", output)
		}
	}

	for _, item := range items {
		itemID := itemNodeID(item)
		builder.graph.Items[itemID] = item
		metrics := map[string]float64{
			MeasureChangedFiles:            float64(item.ChangedFiles),
			MeasureSensitivePathCount:      float64(item.SensitivePathCount),
			MeasureComponentCount:          float64(item.ComponentCount),
			MeasureCentralityWeightedTouch: item.CentralityWeightedTouch,
			MeasureHotspotWeightedTouch:    item.HotspotWeightedTouch,
			MeasureReleaseNoteImportance:   item.ReleaseNoteImportance,
		}
		maps.Copy(metrics, item.Measures)
		builder.addNode(Node{
			ID:      itemID,
			Type:    "work_item",
			Name:    fmt.Sprintf("#%d", item.Number),
			Title:   item.Title,
			Number:  item.Number,
			State:   item.State,
			Item:    item.Type,
			Metrics: metrics,
		})

		builder.addMany(itemID, item.Labels, "label", "HAS_LABEL")
		builder.addMany(itemID, item.Components, "component", "TOUCHES_COMPONENT")
		builder.addMany(itemID, item.Areas, "area", "TOUCHES_AREA")
		builder.addMany(itemID, item.Signals, "signal", "HAS_SIGNAL")
		builder.addMany(itemID, item.ContextSignals, "context_signal", "CONTEXT_MENTIONS_SIGNAL")
		builder.addMany(itemID, item.SourceWorkflowPaths, "workflow_source", "HAS_WORKFLOW_SOURCE")
		builder.addMany(itemID, item.SourceWorkflowRuns, "workflow_run", "HAS_WORKFLOW_RUN")
		builder.addMany(itemID, item.ArchitectureSpecs, "architecture_spec", "RELATED_TO_ARCHITECTURE_SPEC")
		builder.addMany(itemID, item.ArchitectureConcepts, "architecture_concept", "RELATED_TO_ARCHITECTURE_CONCEPT")
		builder.addMany(itemID, []string{item.StateReason}, dimensionNodeType(DimensionStateReason), DimensionEdgeType(DimensionStateReason))
		for _, key := range sortedKeys(item.Dimensions) {
			builder.addMany(itemID, item.Dimensions[key], dimensionNodeType(key), DimensionEdgeType(key))
		}
		for _, node := range item.GraphNodes {
			builder.addNode(node)
		}
		for _, edge := range item.GraphEdges {
			if edge.Source == "" {
				edge.Source = itemID
			}
			builder.addEdgeRecord(edge)
		}

		for _, text := range cleanStrings(item.ContextEvidence) {
			evidenceID := safeNodeID("context_evidence", fmt.Sprintf("%s:%s", itemID, text))
			builder.addNode(Node{ID: evidenceID, Type: "context_evidence", Name: "context", Title: text})
			builder.addEdge(itemID, evidenceID, "HAS_CONTEXT_EVIDENCE", text)
		}

		if item.ReleaseCategory != "" {
			categoryID := safeNodeID("release_category", item.ReleaseCategory)
			builder.addNode(Node{ID: categoryID, Type: "release_category", Name: item.ReleaseCategory})
			builder.addEdge(itemID, categoryID, "HAS_RELEASE_CATEGORY", item.ReleaseCategory)
		}
		if item.ChangeType != "" {
			changeID := safeNodeID("change_type", item.ChangeType)
			builder.addNode(Node{ID: changeID, Type: "change_type", Name: item.ChangeType})
			builder.addEdge(itemID, changeID, "HAS_CHANGE_TYPE", item.ChangeType)
		}

		for _, workflowName := range cleanStrings(item.SourceWorkflows) {
			workflowID := workflowByName[workflowName]
			if workflowID == "" {
				workflowID = safeNodeID("workflow", workflowName)
				builder.addNode(Node{ID: workflowID, Type: "agentic_workflow", Name: workflowName})
			}
			builder.addEdge(itemID, workflowID, "LIKELY_CREATED_BY_WORKFLOW", workflowName)
		}

		for _, workflow := range workflowsWithTitlePrefix {
			if workflow.TitlePrefix != "" && strings.HasPrefix(item.Title, workflow.TitlePrefix) {
				builder.addEdge(itemID, workflowByName[workflow.Name], "LIKELY_CREATED_BY_WORKFLOW", workflow.TitlePrefix)
			}
		}
	}

	builder.sort()
	return builder.graph
}

type graphBuilder struct {
	graph Graph
}

func (b *graphBuilder) addNode(node Node) {
	if node.ID == "" {
		return
	}
	if existing, ok := b.graph.Nodes[node.ID]; ok {
		if betterNodeName(node.Name, existing.Name) {
			existing.Name = node.Name
		}
		if existing.Title == "" {
			existing.Title = node.Title
		}
		if existing.Metrics == nil {
			existing.Metrics = node.Metrics
		}
		b.graph.Nodes[node.ID] = existing
		return
	}
	b.graph.Nodes[node.ID] = node
}

func betterNodeName(candidate, existing string) bool {
	candidate = strings.TrimSpace(candidate)
	existing = strings.TrimSpace(existing)
	if candidate == "" {
		return false
	}
	if existing == "" {
		return true
	}
	return displayNameScore(candidate) > displayNameScore(existing)
}

func displayNameScore(name string) int {
	score := 0
	if strings.Contains(name, " ") {
		score++
	}
	for _, char := range name {
		if char >= 'A' && char <= 'Z' {
			score++
			break
		}
	}
	return score
}

func (b *graphBuilder) addMany(source string, values []string, nodeType string, edgeType string) {
	for _, value := range cleanStrings(values) {
		target := safeNodeID(nodeType, value)
		b.addNode(Node{ID: target, Type: nodeType, Name: value})
		b.addEdge(source, target, edgeType, value)
	}
}

func (b *graphBuilder) addEdge(source, target, edgeType, evidence string) {
	if source == "" || target == "" || edgeType == "" {
		return
	}
	b.addEdgeRecord(Edge{Source: source, Target: target, Type: edgeType, Weight: 1, Evidence: evidence})
}

func (b *graphBuilder) addEdgeRecord(edge Edge) {
	if edge.Source == "" || edge.Target == "" || edge.Type == "" {
		return
	}
	if edge.Weight == 0 {
		edge.Weight = 1
	}
	b.graph.Edges = append(b.graph.Edges, edge)
	b.graph.Out[edge.Source] = append(b.graph.Out[edge.Source], edge)
	b.graph.In[edge.Target] = append(b.graph.In[edge.Target], edge)
}

func (b *graphBuilder) sort() {
	sort.SliceStable(b.graph.Edges, func(i, j int) bool {
		left := b.graph.Edges[i]
		right := b.graph.Edges[j]
		return strings.Join([]string{left.Source, left.Type, left.Target, left.Evidence}, "\x00") < strings.Join([]string{right.Source, right.Type, right.Target, right.Evidence}, "\x00")
	})
	for source := range b.graph.Out {
		sort.SliceStable(b.graph.Out[source], func(i, j int) bool {
			return b.graph.Out[source][i].Type+b.graph.Out[source][i].Target < b.graph.Out[source][j].Type+b.graph.Out[source][j].Target
		})
	}
	for target := range b.graph.In {
		sort.SliceStable(b.graph.In[target], func(i, j int) bool {
			return b.graph.In[target][i].Type+b.graph.In[target][i].Source < b.graph.In[target][j].Type+b.graph.In[target][j].Source
		})
	}
}

func itemNodeID(item WorkItem) string {
	itemType := strings.TrimSpace(item.Type)
	if itemType == "" {
		itemType = "item"
	}
	return fmt.Sprintf("work_item:%s:%d", itemType, item.Number)
}

func safeNodeID(nodeType, value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = nodeIDUnsafePattern.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		normalized = "unknown"
	}
	return nodeType + ":" + normalized
}

func cleanStrings(values []string) []string {
	seen := map[string]bool{}
	cleaned := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// DimensionEdgeType returns the graph edge type used for a custom dimension.
func DimensionEdgeType(name string) string {
	return "HAS_DIMENSION:" + strings.TrimSpace(name)
}

func dimensionNodeType(name string) string {
	return "dimension:" + strings.TrimSpace(name)
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.TrimSpace(key) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
