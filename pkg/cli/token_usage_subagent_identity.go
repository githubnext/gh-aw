package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Older unified sessions omit the top-level agentId of sub-agent lifecycle
// events. Their lifecycle instances are then keyed by data.toolCallId while the
// per-agent metrics snapshot is keyed by the agent ID, so one sub-agent becomes
// two instances, and subagent.configured records, which carry neither ID, lose
// their reasoning effort. The same run's native agent-session.jsonl records
// both data.toolCallId and the top-level agentId on each subagent.started
// event; that runner-written bridge reconciles the two halves.

const nativeSubagentSessionFile = "agent-session.jsonl"

// subagentStartEvidence is the identity evidence of one subagent.started event.
type subagentStartEvidence struct {
	identity      string
	agentID       string
	toolCallID    string
	definition    string
	displayName   string
	start         time.Time
	startValid    bool
	index         json.RawMessage
	nativeIndex   json.RawMessage
	persistedPath string
}

type subagentMetricNames struct {
	definition  string
	displayName string
}

func (models *subagentSessionModels) recordStartEvidence(identity string, event sessionSubagentEvent, data sessionSubagentData) {
	start, err := subagentSessionStart(event)
	models.starts = append(models.starts, subagentStartEvidence{
		identity: identity, agentID: event.AgentID, toolCallID: data.ToolCallID,
		definition: data.AgentName, displayName: data.AgentDisplayName,
		start: start, startValid: err == nil && !start.IsZero(),
		index: event.Provenance.Index, nativeIndex: event.Provenance.Native.Index,
		persistedPath: event.Provenance.Native.PersistedPath,
	})
}

func (models *subagentSessionModels) hasStartsWithoutAgentID() bool {
	for _, start := range models.starts {
		if start.agentID == "" && start.toolCallID != "" {
			return true
		}
	}
	return false
}

// readNativeSubagentSession parses the run's canonical native session trace as
// identity evidence. It returns nil when the trace is absent or unusable, so
// that callers keep the unified analysis unchanged.
func readNativeSubagentSession(runDir string) *subagentSessionModels {
	path := filepath.Join(runDir, nativeSubagentSessionFile)
	if _, err := os.Lstat(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			tokenUsageSubagentLog.Printf("skipping sub-agent identity reconciliation: %v", err)
		}
		return nil
	}
	if err := validateSubagentSessionSource(path); err != nil {
		tokenUsageSubagentLog.Printf("skipping sub-agent identity reconciliation: %v", err)
		return nil
	}
	native, err := parseSessionSubagentFile(path, false)
	if err != nil {
		tokenUsageSubagentLog.Printf("skipping sub-agent identity reconciliation: %s: %v", nativeSubagentSessionFile, err)
		return nil
	}
	return native
}

type subagentIdentityLink struct {
	identity string
	agentID  string
	effort   string
}

// reconcileSubagentIdentities merges each lifecycle instance whose
// subagent.started event lacks agentId into the instance of its canonical agent
// ID. The link requires the same session ID and exactly one native start with
// the same data.toolCallId and a top-level agentId; agent definition, display
// name, start time, and native event index must not conflict. The lifecycle
// instance must carry no accounting of its own, and the agent ID must carry no
// lifecycle or configuration evidence of its own. Instances without such a
// unique, conflict-free link keep their previous output.
func (models *subagentSessionModels) reconcileSubagentIdentities(native *subagentSessionModels) {
	if native == nil || models.sessionID == "" || models.sessionID != native.sessionID {
		return
	}
	nativeStarts := make(map[string]subagentStartEvidence)
	nativeStartCounts := make(map[string]int)
	for _, start := range native.starts {
		if start.toolCallID != "" && start.agentID != "" {
			nativeStarts[start.toolCallID] = start
			nativeStartCounts[start.toolCallID]++
		}
	}
	identityStarts := make(map[string]int)
	explicitAgents := make(map[string]struct{})
	for _, start := range models.starts {
		identityStarts[start.identity]++
		if start.agentID != "" {
			explicitAgents[start.agentID] = struct{}{}
		}
	}

	links := make([]subagentIdentityLink, 0)
	claims := make(map[string]int)
	for _, start := range models.starts {
		if start.agentID != "" || start.toolCallID == "" || identityStarts[start.identity] != 1 {
			continue
		}
		if start.persistedPath != "" && start.persistedPath != nativeSubagentSessionFile {
			continue
		}
		if nativeStartCounts[start.toolCallID] != 1 {
			continue
		}
		match := nativeStarts[start.toolCallID]
		if _, explicit := explicitAgents[match.agentID]; explicit || match.agentID == start.identity {
			continue
		}
		if !models.subagentStartsAgree(start, match) {
			tokenUsageSubagentLog.Printf("not linking sub-agent tool call %s: conflicting native evidence", start.toolCallID)
			continue
		}
		effort := ""
		if row := native.agents[match.agentID]; row != nil {
			effort = row.Effort
		}
		links = append(links, subagentIdentityLink{identity: start.identity, agentID: match.agentID, effort: effort})
		claims[match.agentID]++
	}
	for _, link := range links {
		if claims[link.agentID] == 1 {
			models.mergeSubagentInstance(link)
		}
	}
}

func (models *subagentSessionModels) subagentStartsAgree(unified, native subagentStartEvidence) bool {
	if conflictingSubagentNames(unified.definition, native.definition) ||
		conflictingSubagentNames(unified.displayName, native.displayName) {
		return false
	}
	if unified.startValid && native.startValid && !unified.start.Equal(native.start) {
		return false
	}
	if len(unified.nativeIndex) > 0 && len(native.index) > 0 &&
		!bytes.Equal(bytes.TrimSpace(unified.nativeIndex), bytes.TrimSpace(native.index)) {
		return false
	}
	if names, ok := models.metricNames[native.agentID]; ok {
		// Unified names may be missing, so native names are compared with the
		// metrics names too.
		for _, start := range []subagentStartEvidence{unified, native} {
			if conflictingSubagentNames(start.definition, names.definition) ||
				conflictingSubagentNames(start.displayName, names.displayName) {
				return false
			}
		}
	}
	if usage := models.agentUsage[unified.identity]; usage != nil && hasAgentUsageAccounting(usage) {
		// The lifecycle half carries its own accounting; merging could drop or
		// double it.
		return false
	}
	if len(models.actualModels[unified.identity]) > 0 || len(models.actualCounts[unified.identity]) > 0 {
		return false
	}
	if models.hasLifecycleEvidence(native.agentID) {
		// The metrics-keyed instance has its own lifecycle or configuration
		// events (mixed IDs); merging would overwrite them.
		return false
	}
	return true
}

func hasAgentUsageAccounting(usage *AgentUsageBreakdown) bool {
	return usage.Requests != 0 || len(usage.Models) > 0 || len(usage.requestUsages) > 0 ||
		usage.TokenCoreMetrics != (TokenCoreMetrics{}) || usage.AIC != 0 || usage.TotalApiDurationMs != 0
}

// hasLifecycleEvidence reports whether agentID holds lifecycle or
// configuration evidence beyond what a per-agent metrics snapshot records.
func (models *subagentSessionModels) hasLifecycleEvidence(agentID string) bool {
	if row := models.agents[agentID]; row != nil &&
		(row.InvocationCount != 0 || row.CompletedCount != 0 || row.FailedCount != 0 || row.IncompleteCount != 0 ||
			row.Effort != "" || row.ResolvedModel != "" || row.Error != "" || row.ErrorCode != "") {
		return true
	}
	usage := models.agentUsage[agentID]
	return usage != nil &&
		(usage.CompletedCount != 0 || usage.FailedCount != 0 || usage.IncompleteCount != 0 ||
			usage.Effort != "" || len(usage.RequestedModels) > 0 || len(usage.ResolvedModels) > 0)
}

func conflictingSubagentNames(left, right string) bool {
	return left != "" && right != "" && left != right
}

// mergeSubagentInstance moves the lifecycle instance of link.identity onto the
// agent ID that keys its per-agent metrics. The metrics snapshot is kept as is;
// only the lifecycle outcome, identity, and missing effort are added.
func (models *subagentSessionModels) mergeSubagentInstance(link subagentIdentityLink) {
	lifecycle := models.agents[link.identity]
	if lifecycle == nil {
		return
	}
	merged := *lifecycle
	if metric := models.agents[link.agentID]; metric != nil {
		merged.AgentName = firstNonEmptyModel(merged.AgentName, metric.AgentName)
		merged.RequestedModel = firstNonEmptyModel(merged.RequestedModel, metric.RequestedModel)
		merged.ResolvedModel = firstNonEmptyModel(merged.ResolvedModel, metric.ResolvedModel)
	}
	if merged.Effort == "" {
		merged.Effort = link.effort
	}
	models.agents[link.agentID] = &merged
	delete(models.agents, link.identity)

	instance := models.agentUsage[link.identity]
	target := models.agentUsage[link.agentID]
	switch {
	case instance == nil:
	case target == nil:
		target = instance
	default:
		target.AgentName = firstNonEmptyModel(instance.AgentName, target.AgentName)
		target.AgentType = "subagent"
		target.SourceEngine = firstNonEmptyModel(target.SourceEngine, instance.SourceEngine)
		target.InstanceCount = max(instance.InstanceCount, 1)
		target.CompletedCount = instance.CompletedCount
		target.FailedCount = instance.FailedCount
		target.IncompleteCount = instance.IncompleteCount
		target.Effort = instance.Effort
		for _, model := range instance.RequestedModels {
			target.RequestedModels = appendUnique(target.RequestedModels, model)
		}
		for _, model := range instance.ResolvedModels {
			target.ResolvedModels = appendUnique(target.ResolvedModels, model)
		}
		target.definitionName = firstNonEmptyModel(instance.definitionName, target.definitionName)
		displayNames := slices.Clone(instance.instanceDisplayNames)
		for _, displayName := range target.instanceDisplayNames {
			displayNames = appendUnique(displayNames, displayName)
		}
		target.instanceDisplayNames = displayNames
	}
	if target != nil {
		if target.Effort == "" {
			target.Effort = link.effort
		}
		models.agentUsage[link.agentID] = target
	}
	delete(models.agentUsage, link.identity)
	delete(models.actualCounts, link.identity)
	delete(models.actualModels, link.identity)
	tokenUsageSubagentLog.Printf("linked sub-agent lifecycle %s to agent %s via native toolCallId", link.identity, link.agentID)
}
