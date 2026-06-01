//go:build !integration

package workflow

import (
	"math"
	"strconv"
	"testing"
)

func TestParseOptionalInt(t *testing.T) {
	t.Run("integer float is accepted", func(t *testing.T) {
		value := parseOptionalInt(7.0)
		if value == nil || *value != 7 {
			t.Fatalf("expected 7, got %v", value)
		}
	})

	t.Run("fractional float is rejected", func(t *testing.T) {
		if value := parseOptionalInt(7.5); value != nil {
			t.Fatalf("expected nil, got %d", *value)
		}
	})

	t.Run("nan and infinity are rejected", func(t *testing.T) {
		if value := parseOptionalInt(math.NaN()); value != nil {
			t.Fatalf("expected nil for NaN, got %d", *value)
		}
		if value := parseOptionalInt(math.Inf(1)); value != nil {
			t.Fatalf("expected nil for +Inf, got %d", *value)
		}
	})

	t.Run("uint64 above max int is rejected", func(t *testing.T) {
		if value := parseOptionalInt(uint64(math.MaxInt) + 1); value != nil {
			t.Fatalf("expected nil, got %d", *value)
		}
	})

	t.Run("large exact float is architecture aware", func(t *testing.T) {
		value := parseOptionalInt(1e12)
		if strconv.IntSize == 32 {
			if value != nil {
				t.Fatalf("expected nil on 32-bit, got %d", *value)
			}
			return
		}
		if value == nil || *value != 1000000000000 {
			t.Fatalf("expected 1000000000000 on 64-bit, got %v", value)
		}
	})
}
