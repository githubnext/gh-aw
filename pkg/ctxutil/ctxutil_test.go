package ctxutil

import (
	"context"
	"testing"
)

func TestOrBackground(t *testing.T) {
	type key string
	const testKey key = "test-key"

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "nil context returns background",
			ctx:  nil,
		},
		{
			name: "non-nil context returned unchanged",
			ctx:  context.WithValue(context.Background(), testKey, "value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OrBackground(tt.ctx)
			if got == nil {
				t.Fatal("OrBackground returned nil")
			}
			if tt.ctx == nil {
				if got != context.Background() {
					t.Errorf("expected context.Background() for nil input, got %v", got)
				}
			} else {
				if got != tt.ctx {
					t.Errorf("expected original context to be returned unchanged, got %v", got)
				}
				if got.Value(testKey) != "value" {
					t.Errorf("expected value to be preserved, got %v", got.Value(testKey))
				}
			}
		})
	}
}
