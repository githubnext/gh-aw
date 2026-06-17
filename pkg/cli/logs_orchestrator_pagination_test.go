//go:build !integration

package cli

import (
	"testing"
	"time"
)

func TestShouldStopPagination(t *testing.T) {
	tests := []struct {
		name         string
		totalFetched int
		batchSize    int
		want         bool
	}{
		{
			name:         "stop when raw batch is smaller than requested",
			totalFetched: 249,
			batchSize:    250,
			want:         true,
		},
		{
			name:         "continue when raw batch is full",
			totalFetched: 250,
			batchSize:    250,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldStopPagination(tt.totalFetched, tt.batchSize); got != tt.want {
				t.Fatalf("shouldStopPagination(%d, %d) = %v, want %v", tt.totalFetched, tt.batchSize, got, tt.want)
			}
		})
	}
}

func TestSelectPaginationCursorDate(t *testing.T) {
	oldestFetched := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	filteredRuns := []WorkflowRun{
		{CreatedAt: time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)},
		{CreatedAt: time.Date(2026, 6, 16, 11, 0, 0, 0, time.UTC)},
	}

	cursor, ok := selectPaginationCursorDate(filteredRuns, oldestFetched)
	if !ok {
		t.Fatal("expected cursor to be set when raw oldest fetched run is available")
	}
	if cursor != oldestFetched.Format(time.RFC3339) {
		t.Fatalf("expected cursor %s, got %s", oldestFetched.Format(time.RFC3339), cursor)
	}
}

func TestSelectPaginationCursorDateFallsBackToFilteredRuns(t *testing.T) {
	filteredOldest := time.Date(2026, 6, 15, 18, 30, 0, 0, time.UTC)
	filteredRuns := []WorkflowRun{
		{CreatedAt: time.Date(2026, 6, 15, 19, 0, 0, 0, time.UTC)},
		{CreatedAt: filteredOldest},
	}

	cursor, ok := selectPaginationCursorDate(filteredRuns, time.Time{})
	if !ok {
		t.Fatal("expected cursor to be set from filtered runs")
	}
	if cursor != filteredOldest.Format(time.RFC3339) {
		t.Fatalf("expected fallback cursor %s, got %s", filteredOldest.Format(time.RFC3339), cursor)
	}
}

func TestSelectPaginationCursorDateNoCursor(t *testing.T) {
	cursor, ok := selectPaginationCursorDate(nil, time.Time{})
	if ok {
		t.Fatalf("expected no cursor, got %s", cursor)
	}
}
