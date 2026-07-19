package trimleftright

import "testing"

func TestLooksSuspiciousCutset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		cutset string
		want   bool
	}{
		{name: "empty", cutset: "", want: false},
		{name: "single rune", cutset: "a", want: false},
		{name: "repeated alnum", cutset: "foo", want: true},
		{name: "unique alnum", cutset: "aeiou", want: false},
		{name: "contains whitespace", cutset: " \t", want: false},
		{name: "contains punctuation", cutset: "a!", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksSuspiciousCutset(tt.cutset)
			if got != tt.want {
				t.Fatalf("looksSuspiciousCutset(%q) = %v, want %v", tt.cutset, got, tt.want)
			}
		})
	}
}
