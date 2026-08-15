package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstForbiddenCharInParamValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want rune
	}{
		{name: "empty string", in: "", want: 0},
		{name: "all lowercase letters", in: "abcxyz", want: 0},
		{name: "all uppercase letters", in: "ABCXYZ", want: 0},
		{name: "all digits", in: "0123456789", want: 0},
		{name: "underscore allowed", in: "a_b", want: 0},
		{name: "dot allowed", in: "a.b", want: 0},
		{name: "hyphen allowed", in: "a-b", want: 0},
		{name: "mixed allowed chars", in: "gpt-4o_v1.2", want: 0},
		{name: "space forbidden", in: "a b", want: ' '},
		{name: "slash forbidden as first char", in: "/abc", want: '/'},
		{name: "slash forbidden mid string", in: "abc/def", want: '/'},
		{name: "question mark forbidden", in: "abc?def", want: '?'},
		{name: "unicode letter forbidden", in: "abcé", want: 'é'},
		{name: "forbidden char not at start returns first occurrence", in: "ab!cd!ef", want: '!'},
		{name: "only forbidden char", in: "!", want: '!'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := firstForbiddenCharInParamValue(tt.in)
			assert.Equal(t, tt.want, got, "firstForbiddenCharInParamValue(%q)", tt.in)
		})
	}
}
