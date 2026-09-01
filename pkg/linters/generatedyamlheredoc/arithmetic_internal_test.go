//go:build !integration

package generatedyamlheredoc

import "testing"

func TestHasOpenShellArithmeticExpression(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"empty string", "", false},
		{"plain text no arithmetic", "echo hello world", false},
		{"open arithmetic dollar form", "x=$((1 + ", true},
		{"closed arithmetic dollar form", "x=$((1 + 2))", false},
		{"open arithmetic bare double paren", " ((1+2)", true},
		{"closed arithmetic bare double paren", "((1+2))", false},
		{"single open paren only", "(1+2)", false},
		{"nested arithmetic both open", "$((1+$((2", true},
		{"nested arithmetic outer closed inner open", "$((1+$((2))", true},
		{"nested arithmetic fully closed", "$((1+$((2))))", false},
		{"arithmetic inside single quotes ignored", "'$((1+2'", false},
		{"arithmetic inside double quotes not ignored due to boundary", "\"$((1+2\"", true},
		{"escaped double paren", "\\(( not arithmetic", false},
		{"closing without opening ignored", "1+2))", false},
		{"single quote toggling", "'it''s $((", false},
		{"double quote toggling", "\"a\"$((b", true},
		{"backslash before quote inside single quote", "'\\''$((", false},
		{"word boundary before double paren opens arithmetic", "foo ((bar", true},
		{"non boundary before double paren no arithmetic", "foo((bar", false},
		{"dollar prefixed always opens regardless of boundary", "x$((y", true},
		{"multiple closes exceeding opens stay at zero", "$((1))) ))", false},
		{"trailing backslash at end of line", "abc\\", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasOpenShellArithmeticExpression(tt.line); got != tt.want {
				t.Errorf("hasOpenShellArithmeticExpression(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
