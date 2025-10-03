package workflow

import (
	"reflect"
	"testing"
)

func TestGetAllCommentEvents(t *testing.T) {
	events := GetAllCommentEvents()

	// Should have exactly 4 events
	if len(events) != 4 {
		t.Errorf("Expected 4 comment events, got %d", len(events))
	}

	// Check that all expected events are present
	expectedEvents := map[string][]string{
		"issues":                          {"opened", "edited", "reopened"},
		"issue_comment":                   {"created", "edited"},
		"pull_request":                    {"opened", "edited", "reopened"},
		"pull_request_review_comment":     {"created", "edited"},
	}

	for _, event := range events {
		expected, ok := expectedEvents[event.EventName]
		if !ok {
			t.Errorf("Unexpected event name: %s", event.EventName)
			continue
		}

		if !reflect.DeepEqual(event.Types, expected) {
			t.Errorf("For event %s, expected types %v, got %v", event.EventName, expected, event.Types)
		}
	}
}

func TestGetCommentEventByIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantEvent  string
		wantNil    bool
	}{
		{
			name:       "short identifier 'issue'",
			identifier: "issue",
			wantEvent:  "issues",
		},
		{
			name:       "full identifier 'issues'",
			identifier: "issues",
			wantEvent:  "issues",
		},
		{
			name:       "short identifier 'comment'",
			identifier: "comment",
			wantEvent:  "issue_comment",
		},
		{
			name:       "full identifier 'issue_comment'",
			identifier: "issue_comment",
			wantEvent:  "issue_comment",
		},
		{
			name:       "short identifier 'pr'",
			identifier: "pr",
			wantEvent:  "pull_request",
		},
		{
			name:       "full identifier 'pull_request'",
			identifier: "pull_request",
			wantEvent:  "pull_request",
		},
		{
			name:       "short identifier 'pr_review'",
			identifier: "pr_review",
			wantEvent:  "pull_request_review_comment",
		},
		{
			name:       "full identifier 'pull_request_review_comment'",
			identifier: "pull_request_review_comment",
			wantEvent:  "pull_request_review_comment",
		},
		{
			name:       "invalid identifier",
			identifier: "invalid",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCommentEventByIdentifier(tt.identifier)

			if tt.wantNil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected non-nil result, got nil")
				return
			}

			if result.EventName != tt.wantEvent {
				t.Errorf("Expected event name %s, got %s", tt.wantEvent, result.EventName)
			}
		})
	}
}

func TestParseCommandEvents(t *testing.T) {
	tests := []struct {
		name       string
		eventsValue any
		want       []string
		wantNil    bool
	}{
		{
			name:       "nil value returns default",
			eventsValue: nil,
			wantNil:    true,
		},
		{
			name:       "wildcard string returns default",
			eventsValue: "*",
			wantNil:    true,
		},
		{
			name:       "single event string",
			eventsValue: "issue",
			want:       []string{"issue"},
		},
		{
			name:       "array of event strings",
			eventsValue: []any{"issue", "comment"},
			want:       []string{"issue", "comment"},
		},
		{
			name:       "empty array returns default",
			eventsValue: []any{},
			wantNil:    true,
		},
		{
			name:       "array with non-strings is filtered",
			eventsValue: []any{"issue", 123, "comment"},
			want:       []string{"issue", "comment"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommandEvents(tt.eventsValue)

			if tt.wantNil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("Expected %v, got %v", tt.want, result)
			}
		})
	}
}

func TestFilterCommentEvents(t *testing.T) {
	tests := []struct {
		name        string
		identifiers []string
		wantCount   int
		wantEvents  []string
	}{
		{
			name:        "nil identifiers returns all events",
			identifiers: nil,
			wantCount:   4,
			wantEvents:  []string{"issues", "issue_comment", "pull_request", "pull_request_review_comment"},
		},
		{
			name:        "empty identifiers returns all events",
			identifiers: []string{},
			wantCount:   4,
			wantEvents:  []string{"issues", "issue_comment", "pull_request", "pull_request_review_comment"},
		},
		{
			name:        "single identifier",
			identifiers: []string{"issue"},
			wantCount:   1,
			wantEvents:  []string{"issues"},
		},
		{
			name:        "multiple identifiers",
			identifiers: []string{"issue", "comment"},
			wantCount:   2,
			wantEvents:  []string{"issues", "issue_comment"},
		},
		{
			name:        "invalid identifiers are filtered out",
			identifiers: []string{"issue", "invalid", "comment"},
			wantCount:   2,
			wantEvents:  []string{"issues", "issue_comment"},
		},
		{
			name:        "short and full identifiers",
			identifiers: []string{"pr", "pull_request_review_comment"},
			wantCount:   2,
			wantEvents:  []string{"pull_request", "pull_request_review_comment"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterCommentEvents(tt.identifiers)

			if len(result) != tt.wantCount {
				t.Errorf("Expected %d events, got %d", tt.wantCount, len(result))
			}

			gotEvents := GetCommentEventNames(result)
			if !reflect.DeepEqual(gotEvents, tt.wantEvents) {
				t.Errorf("Expected events %v, got %v", tt.wantEvents, gotEvents)
			}
		})
	}
}

func TestGetCommentEventNames(t *testing.T) {
	mappings := []CommentEventMapping{
		{EventName: "issues", Types: []string{"opened"}},
		{EventName: "issue_comment", Types: []string{"created"}},
	}

	result := GetCommentEventNames(mappings)

	expected := []string{"issues", "issue_comment"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
