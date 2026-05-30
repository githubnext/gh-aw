//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// writeJSONL writes lines as JSON objects to path, one per line.
func writeJSONL(t *testing.T, path string, objects []any) {
	t.Helper()
	var sb strings.Builder
	for _, obj := range objects {
		b, err := json.Marshal(obj)
		if err != nil {
			t.Fatalf("writeJSONL marshal: %v", err)
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0600); err != nil {
		t.Fatalf("writeJSONL write: %v", err)
	}
}

// ─── UnifiedTimelineEvent helpers ────────────────────────────────────────────

func TestUnifiedTimelineEvent_Fields(t *testing.T) {
	now := time.Now().UTC()
	evt := UnifiedTimelineEvent{
		Time:       now,
		Source:     TimelineSourceGateway,
		Kind:       TimelineKindToolCall,
		ServerName: "srv",
		ToolName:   "get_file",
		Status:     "success",
		Duration:   42.5,
	}
	if evt.Source != TimelineSourceGateway {
		t.Errorf("Source = %q; want %q", evt.Source, TimelineSourceGateway)
	}
	if evt.Kind != TimelineKindToolCall {
		t.Errorf("Kind = %q; want %q", evt.Kind, TimelineKindToolCall)
	}
}

// ─── parseEventsJSONL ────────────────────────────────────────────────────────

func TestParseEventsJSONL_BasicTypes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	writeJSONL(t, path, []any{
		map[string]any{
			"type":      "session.start",
			"id":        "id1",
			"timestamp": "2024-01-15T10:00:00Z",
			"data":      map[string]any{"sessionId": "sess1", "copilotVersion": "1.0"},
		},
		map[string]any{
			"type":      "user.message",
			"id":        "id2",
			"timestamp": "2024-01-15T10:00:01Z",
			"data":      map[string]any{},
		},
		map[string]any{
			"type":      "tool.execution_start",
			"id":        "id3",
			"timestamp": "2024-01-15T10:00:02Z",
			"data": map[string]any{
				"toolCallId":    "call-1",
				"toolName":      "get_file",
				"mcpServerName": "my-server",
			},
		},
		map[string]any{
			"type":      "tool.execution_complete",
			"id":        "id4",
			"timestamp": "2024-01-15T10:00:03Z",
			"data": map[string]any{
				"toolCallId": "call-1",
				"toolName":   "get_file",
				"success":    true,
			},
		},
	})

	entries, err := parseEventsJSONL(path)
	if err != nil {
		t.Fatalf("parseEventsJSONL: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("got %d entries; want 4", len(entries))
	}
	if entries[2].Data.ToolCallID != "call-1" {
		t.Errorf("ToolCallID = %q; want call-1", entries[2].Data.ToolCallID)
	}
	if entries[3].Data.Success != true {
		t.Errorf("Success = %v; want true", entries[3].Data.Success)
	}
}

func TestParseEventsJSONL_MalformedLinesSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	content := "not-json\n" +
		`{"type":"user.message","id":"id1","timestamp":"2024-01-15T10:00:01Z","data":{}}` + "\n" +
		"also not json\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	entries, err := parseEventsJSONL(path)
	if err != nil {
		t.Fatalf("parseEventsJSONL: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("got %d entries; want 1 (malformed lines should be skipped)", len(entries))
	}
}

// ─── agentEntryToTimelineEvent ────────────────────────────────────────────────

func TestAgentEntryToTimelineEvent_UserMessage(t *testing.T) {
	entry := copilotEventsJSONLEntry{
		Type:      "user.message",
		Timestamp: "2024-01-15T10:00:01Z",
	}
	evt, ok := agentEntryToTimelineEvent(entry, 3)
	if !ok {
		t.Fatal("ok = false; want true for user.message")
	}
	if evt.Kind != TimelineKindAgentTurn {
		t.Errorf("Kind = %q; want %q", evt.Kind, TimelineKindAgentTurn)
	}
	if evt.TurnIndex != 3 {
		t.Errorf("TurnIndex = %d; want 3", evt.TurnIndex)
	}
	if evt.Source != TimelineSourceAgent {
		t.Errorf("Source = %q; want %q", evt.Source, TimelineSourceAgent)
	}
}

func TestAgentEntryToTimelineEvent_ToolStart(t *testing.T) {
	entry := copilotEventsJSONLEntry{
		Type:      "tool.execution_start",
		Timestamp: "2024-01-15T10:00:02Z",
		Data: copilotEventsJSONLEntryData{
			ToolCallID:    "call-abc",
			ToolName:      "search_files",
			MCPServerName: "my-server",
		},
	}
	evt, ok := agentEntryToTimelineEvent(entry, 1)
	if !ok {
		t.Fatal("ok = false; want true for tool.execution_start")
	}
	if evt.Kind != TimelineKindAgentToolStart {
		t.Errorf("Kind = %q; want %q", evt.Kind, TimelineKindAgentToolStart)
	}
	if evt.ToolCallID != "call-abc" {
		t.Errorf("ToolCallID = %q; want call-abc", evt.ToolCallID)
	}
	if evt.ServerName != "my-server" {
		t.Errorf("ServerName = %q; want my-server", evt.ServerName)
	}
}

func TestAgentEntryToTimelineEvent_ToolDoneSuccess(t *testing.T) {
	entry := copilotEventsJSONLEntry{
		Type:      "tool.execution_complete",
		Timestamp: "2024-01-15T10:00:03Z",
		Data: copilotEventsJSONLEntryData{
			ToolCallID: "call-abc",
			ToolName:   "search_files",
			Success:    true,
		},
	}
	evt, ok := agentEntryToTimelineEvent(entry, 1)
	if !ok {
		t.Fatal("ok = false; want true for tool.execution_complete")
	}
	if evt.Kind != TimelineKindAgentToolDone {
		t.Errorf("Kind = %q; want %q", evt.Kind, TimelineKindAgentToolDone)
	}
	if evt.Status != "success" {
		t.Errorf("Status = %q; want success", evt.Status)
	}
	if !evt.Success {
		t.Errorf("Success = false; want true")
	}
}

func TestAgentEntryToTimelineEvent_ToolDoneError(t *testing.T) {
	entry := copilotEventsJSONLEntry{
		Type:      "tool.execution_complete",
		Timestamp: "2024-01-15T10:00:05Z",
		Data: copilotEventsJSONLEntryData{
			ToolCallID: "call-xyz",
			ToolName:   "run_command",
			Success:    false,
		},
	}
	evt, ok := agentEntryToTimelineEvent(entry, 1)
	if !ok {
		t.Fatal("ok = false; want true")
	}
	if evt.Status != "error" {
		t.Errorf("Status = %q; want error", evt.Status)
	}
}

func TestAgentEntryToTimelineEvent_SessionStartSkipped(t *testing.T) {
	entry := copilotEventsJSONLEntry{
		Type:      "session.start",
		Timestamp: "2024-01-15T10:00:00Z",
	}
	_, ok := agentEntryToTimelineEvent(entry, 0)
	if ok {
		t.Error("ok = true; session.start should be skipped (ok = false)")
	}
}

func TestAgentEntryToTimelineEvent_BadTimestamp(t *testing.T) {
	entry := copilotEventsJSONLEntry{
		Type:      "user.message",
		Timestamp: "not-a-timestamp",
	}
	_, ok := agentEntryToTimelineEvent(entry, 1)
	if ok {
		t.Error("ok = true; bad timestamp should return ok = false")
	}
}

func TestAgentEntryToTimelineEvent_UserMessage_WithContent(t *testing.T) {
	entry := copilotEventsJSONLEntry{
		Type:      "user.message",
		Timestamp: "2024-01-15T10:00:01Z",
		Data:      copilotEventsJSONLEntryData{Content: "What files are in the repo?"},
	}
	evt, ok := agentEntryToTimelineEvent(entry, 1)
	if !ok {
		t.Fatal("ok = false; want true for user.message")
	}
	if evt.Kind != TimelineKindAgentTurn {
		t.Errorf("Kind = %q; want %q", evt.Kind, TimelineKindAgentTurn)
	}
	if evt.MessageContent != "What files are in the repo?" {
		t.Errorf("MessageContent = %q; want %q", evt.MessageContent, "What files are in the repo?")
	}
}

func TestAgentEntryToTimelineEvent_AssistantMessage(t *testing.T) {
	entry := copilotEventsJSONLEntry{
		Type:      "assistant.message",
		Timestamp: "2024-01-15T10:00:02Z",
		Data:      copilotEventsJSONLEntryData{Content: "Here are the files."},
	}
	evt, ok := agentEntryToTimelineEvent(entry, 1)
	if !ok {
		t.Fatal("ok = false; want true for assistant.message")
	}
	if evt.Kind != TimelineKindAssistantMessage {
		t.Errorf("Kind = %q; want %q", evt.Kind, TimelineKindAssistantMessage)
	}
	if evt.MessageContent != "Here are the files." {
		t.Errorf("MessageContent = %q; want %q", evt.MessageContent, "Here are the files.")
	}
	if evt.Source != TimelineSourceAgent {
		t.Errorf("Source = %q; want %q", evt.Source, TimelineSourceAgent)
	}
}

func TestAgentEntryToTimelineEvent_Reasoning(t *testing.T) {
	for _, eventType := range []string{"reasoning", "assistant.reasoning"} {
		t.Run(eventType, func(t *testing.T) {
			entry := copilotEventsJSONLEntry{
				Type:      eventType,
				Timestamp: "2024-01-15T10:00:03Z",
				Data:      copilotEventsJSONLEntryData{Content: "I should search for files."},
			}
			evt, ok := agentEntryToTimelineEvent(entry, 1)
			if !ok {
				t.Fatalf("ok = false; want true for %s", eventType)
			}
			if evt.Kind != TimelineKindReasoning {
				t.Errorf("Kind = %q; want %q", evt.Kind, TimelineKindReasoning)
			}
			if evt.MessageContent != "I should search for files." {
				t.Errorf("MessageContent = %q; want %q", evt.MessageContent, "I should search for files.")
			}
		})
	}
}

// ─── collectAgentTimelineEvents ──────────────────────────────────────────────

func TestCollectAgentTimelineEvents_ReturnsNilWhenMissing(t *testing.T) {
	dir := t.TempDir()
	events, err := collectAgentTimelineEvents(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events != nil {
		t.Errorf("events = %v; want nil when no events.jsonl found", events)
	}
}

func TestCollectAgentTimelineEvents_ReadsCanonicalPath(t *testing.T) {
	dir := t.TempDir()
	// Place events.jsonl at the canonical path: sandbox/agent/logs/copilot-session-state/<uuid>/events.jsonl
	sessionDir := filepath.Join(dir, "sandbox", "agent", "logs", "copilot-session-state", "test-uuid-1234")
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		t.Fatal(err)
	}
	eventsPath := filepath.Join(sessionDir, "events.jsonl")
	writeJSONL(t, eventsPath, []any{
		map[string]any{
			"type":      "user.message",
			"id":        "id1",
			"timestamp": "2024-01-15T10:00:01Z",
			"data":      map[string]any{},
		},
		map[string]any{
			"type":      "tool.execution_start",
			"id":        "id2",
			"timestamp": "2024-01-15T10:00:02Z",
			"data": map[string]any{
				"toolCallId": "call-1",
				"toolName":   "get_file",
			},
		},
		map[string]any{
			"type":      "tool.execution_complete",
			"id":        "id3",
			"timestamp": "2024-01-15T10:00:03Z",
			"data":      map[string]any{"toolCallId": "call-1", "toolName": "get_file", "success": true},
		},
	})

	events, err := collectAgentTimelineEvents(dir, false)
	if err != nil {
		t.Fatalf("collectAgentTimelineEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events; want 3", len(events))
	}
	if events[0].Kind != TimelineKindAgentTurn {
		t.Errorf("[0].Kind = %q; want %q", events[0].Kind, TimelineKindAgentTurn)
	}
	if events[0].TurnIndex != 1 {
		t.Errorf("[0].TurnIndex = %d; want 1", events[0].TurnIndex)
	}
}

// ─── BuildUnifiedTimeline ────────────────────────────────────────────────────

func TestBuildUnifiedTimeline_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	events, err := BuildUnifiedTimeline(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("got %d events; want 0 for empty dir", len(events))
	}
}

func TestBuildUnifiedTimeline_SortsMixedSources(t *testing.T) {
	dir := t.TempDir()

	// Gateway: gateway.jsonl with a tool call at t+2s
	gatewayPath := filepath.Join(dir, "gateway.jsonl")
	writeJSONL(t, gatewayPath, []any{
		map[string]any{
			"timestamp":   "2024-01-15T10:00:02.000Z",
			"event":       "tool_call",
			"server_name": "gw-server",
			"tool_name":   "get_file",
			"duration":    100.0,
		},
	})

	// Agent: events.jsonl at canonical path with a turn at t+1s
	sessionDir := filepath.Join(dir, "sandbox", "agent", "logs", "copilot-session-state", "uuid-abc")
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		t.Fatal(err)
	}
	writeJSONL(t, filepath.Join(sessionDir, "events.jsonl"), []any{
		map[string]any{
			"type":      "user.message",
			"id":        "m1",
			"timestamp": "2024-01-15T10:00:01Z",
			"data":      map[string]any{},
		},
	})

	events, err := BuildUnifiedTimeline(dir, false)
	if err != nil {
		t.Fatalf("BuildUnifiedTimeline: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events; want 2", len(events))
	}
	// After sorting: agent turn (t+1) should come before gateway tool call (t+2)
	if events[0].Source != TimelineSourceAgent {
		t.Errorf("events[0].Source = %q; want %q (agent turn should be first)", events[0].Source, TimelineSourceAgent)
	}
	if events[1].Source != TimelineSourceGateway {
		t.Errorf("events[1].Source = %q; want %q (gateway tool call should be second)", events[1].Source, TimelineSourceGateway)
	}
}

// ─── renderUnifiedTimeline ────────────────────────────────────────────────────

func TestRenderUnifiedTimeline_Empty(t *testing.T) {
	out := renderUnifiedTimeline(nil)
	if out != "" {
		t.Errorf("renderUnifiedTimeline(nil) = %q; want empty string", out)
	}
}

func TestRenderUnifiedTimeline_AllSources(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	events := []UnifiedTimelineEvent{
		{
			Time: now, Source: TimelineSourceAgent, Kind: TimelineKindAgentTurn, TurnIndex: 1,
		},
		{
			Time: now.Add(1 * time.Second), Source: TimelineSourceGateway, Kind: TimelineKindToolCall,
			ServerName: "srv", ToolName: "get_file", Status: "success", Duration: 50,
		},
		{
			Time: now.Add(2 * time.Second), Source: TimelineSourceFirewall, Kind: TimelineKindNetworkAllowed,
			Host: "api.example.com:443", HTTPMethod: "CONNECT", HTTPStatus: 200,
		},
		{
			Time: now.Add(3 * time.Second), Source: TimelineSourceAgent, Kind: TimelineKindAgentToolStart,
			ToolName: "search_code", ServerName: "code-srv", ToolCallID: "c1",
		},
		{
			Time: now.Add(4 * time.Second), Source: TimelineSourceAgent, Kind: TimelineKindAgentToolDone,
			ToolName: "search_code", ToolCallID: "c1", Success: true, Status: "success",
		},
	}

	out := renderUnifiedTimeline(events)
	if out == "" {
		t.Fatal("renderUnifiedTimeline returned empty string; want non-empty")
	}
	// Should mention all three source labels
	for _, want := range []string{"GW", "FW", "AG"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing source label %q", want)
		}
	}
	// Should mention event kind labels
	for _, want := range []string{"tool_call", "net_allowed", "agent_turn", "tool_start", "tool_done"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing kind label %q", want)
		}
	}
	// Summary header should be present
	if !strings.Contains(out, "Total Events") {
		t.Error("output missing 'Total Events' summary line")
	}
}

func TestRenderUnifiedTimeline_AgentCountsInSummary(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	events := []UnifiedTimelineEvent{
		{Time: now, Source: TimelineSourceAgent, Kind: TimelineKindAgentTurn, TurnIndex: 1},
		{Time: now.Add(1 * time.Second), Source: TimelineSourceAgent, Kind: TimelineKindAgentToolStart, ToolName: "t"},
		{Time: now.Add(2 * time.Second), Source: TimelineSourceAgent, Kind: TimelineKindAgentToolDone, ToolName: "t", Success: true, Status: "success"},
	}
	out := renderUnifiedTimeline(events)
	if !strings.Contains(out, "Agent") {
		t.Error("output missing 'Agent' summary line")
	}
	// Should show turns=1, tool_start=1, tool_done=1
	if !strings.Contains(out, "turns=1") {
		t.Errorf("output missing turns=1; got:\n%s", out)
	}
}

// ─── rendering primitives ────────────────────────────────────────────────────

func TestRenderAgentTurnRow(t *testing.T) {
	evt := UnifiedTimelineEvent{
		Time:      time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Source:    TimelineSourceAgent,
		Kind:      TimelineKindAgentTurn,
		TurnIndex: 2,
	}
	row := renderAgentTurnRow(evt)
	if len(row) != 5 {
		t.Fatalf("row len = %d; want 5", len(row))
	}
	if row[1] != "AG" {
		t.Errorf("Src = %q; want AG", row[1])
	}
	if !strings.Contains(row[3], "turn 2") {
		t.Errorf("Detail = %q; want 'turn 2'", row[3])
	}
}

func TestRenderAgentToolStartRow_WithServer(t *testing.T) {
	evt := UnifiedTimelineEvent{
		Time:       time.Date(2024, 1, 15, 10, 0, 1, 0, time.UTC),
		Source:     TimelineSourceAgent,
		Kind:       TimelineKindAgentToolStart,
		ServerName: "my-server",
		ToolName:   "search_files",
		ToolCallID: "call-1",
	}
	row := renderAgentToolStartRow(evt)
	if !strings.Contains(row[3], "my-server/search_files") {
		t.Errorf("Detail = %q; want 'my-server/search_files'", row[3])
	}
}

func TestRenderAgentToolStartRow_WithoutServer(t *testing.T) {
	evt := UnifiedTimelineEvent{
		Time:     time.Date(2024, 1, 15, 10, 0, 1, 0, time.UTC),
		Source:   TimelineSourceAgent,
		Kind:     TimelineKindAgentToolStart,
		ToolName: "run_command",
	}
	row := renderAgentToolStartRow(evt)
	if row[3] != "run_command" {
		t.Errorf("Detail = %q; want 'run_command'", row[3])
	}
}

func TestRenderAgentToolDoneRow_StatusFromField(t *testing.T) {
	evt := UnifiedTimelineEvent{
		Time:    time.Date(2024, 1, 15, 10, 0, 2, 0, time.UTC),
		Source:  TimelineSourceAgent,
		Kind:    TimelineKindAgentToolDone,
		Success: false,
		Status:  "error",
	}
	row := renderAgentToolDoneRow(evt)
	if row[4] != "error" {
		t.Errorf("Status = %q; want error", row[4])
	}
}

func TestRenderAgentToolDoneRow_StatusFromSuccessFlag(t *testing.T) {
	evt := UnifiedTimelineEvent{
		Time:    time.Date(2024, 1, 15, 10, 0, 2, 0, time.UTC),
		Source:  TimelineSourceAgent,
		Kind:    TimelineKindAgentToolDone,
		Success: true,
		// Status not explicitly set — should derive from Success
	}
	row := renderAgentToolDoneRow(evt)
	if row[4] != "success" {
		t.Errorf("Status = %q; want success (derived from Success=true)", row[4])
	}
}

// ─── timelineEventIcon / timelineEventKindLabel / timelineSourceLabel ─────────

func TestTimelineEventIcon_AllKinds(t *testing.T) {
	kinds := []TimelineEventKind{
		TimelineKindToolCall,
		TimelineKindDIFCFiltered,
		TimelineKindGuardPolicyBlocked,
		TimelineKindNetworkAllowed,
		TimelineKindNetworkBlocked,
		TimelineKindAgentTurn,
		TimelineKindAgentToolStart,
		TimelineKindAgentToolDone,
		TimelineKindAssistantMessage,
		TimelineKindReasoning,
	}
	for _, k := range kinds {
		icon := timelineEventIcon(k)
		if icon == "" || icon == "·" {
			t.Errorf("timelineEventIcon(%q) = %q; want non-default icon", k, icon)
		}
	}
}

func TestTimelineSourceLabel_Agent(t *testing.T) {
	if got := timelineSourceLabel(TimelineSourceAgent); got != "AG" {
		t.Errorf("timelineSourceLabel(TimelineSourceAgent) = %q; want AG", got)
	}
}

// ─── renderMessageSnippet ─────────────────────────────────────────────────────

func TestRenderMessageSnippet_Empty(t *testing.T) {
	noop := noopStyleRenderer{}
	out := renderMessageSnippet("", "  ", noop, noop)
	if out != "" {
		t.Errorf("renderMessageSnippet(\"\") = %q; want empty string", out)
	}
}

func TestRenderMessageSnippet_SingleLine(t *testing.T) {
	noop := noopStyleRenderer{}
	out := renderMessageSnippet("hello world", "  ", noop, noop)
	if !strings.Contains(out, "hello world") {
		t.Errorf("renderMessageSnippet single line = %q; want to contain 'hello world'", out)
	}
	if strings.Contains(out, "…") {
		t.Errorf("renderMessageSnippet single line = %q; should not contain truncation marker", out)
	}
}

func TestRenderMessageSnippet_TruncatesAfterMaxLines(t *testing.T) {
	noop := noopStyleRenderer{}
	content := "line1\nline2\nline3\nline4\nline5"
	out := renderMessageSnippet(content, "  ", noop, noop)
	if !strings.Contains(out, "line1") || !strings.Contains(out, "line2") || !strings.Contains(out, "line3") {
		t.Errorf("renderMessageSnippet = %q; want first 3 lines present", out)
	}
	if strings.Contains(out, "line4") || strings.Contains(out, "line5") {
		t.Errorf("renderMessageSnippet = %q; should not contain lines beyond max (%d)", out, streamMaxMessageLines)
	}
	if !strings.Contains(out, "…") {
		t.Errorf("renderMessageSnippet = %q; want truncation marker '…'", out)
	}
}

func TestRenderMessageSnippet_SkipsBlankLines(t *testing.T) {
	noop := noopStyleRenderer{}
	content := "\n\nfirst line\n\nsecond line\n"
	out := renderMessageSnippet(content, "  ", noop, noop)
	if !strings.Contains(out, "first line") || !strings.Contains(out, "second line") {
		t.Errorf("renderMessageSnippet = %q; want non-blank lines shown", out)
	}
}
