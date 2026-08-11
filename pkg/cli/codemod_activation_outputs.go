package cli

import (
	"regexp"

	"github.com/github/gh-aw/pkg/logger"
)

var activationOutputsCodemodLog = logger.New("cli:codemod_activation_outputs")

var activationOutputPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{name: "text", pattern: regexp.MustCompile(`needs\.activation\.outputs\.text\b`)},
	{name: "title", pattern: regexp.MustCompile(`needs\.activation\.outputs\.title\b`)},
	{name: "body", pattern: regexp.MustCompile(`needs\.activation\.outputs\.body\b`)},
}

// getActivationOutputsCodemod creates a codemod for transforming needs.activation.outputs.* to steps.sanitized.outputs.*
func getActivationOutputsCodemod() Codemod {
	return Codemod{
		ID:           "activation-outputs-to-sanitized-step",
		Name:         "Transform activation outputs to sanitized step",
		Description:  "Transforms 'needs.activation.outputs.{text|title|body}' to 'steps.sanitized.outputs.{text|title|body}' because the activation job cannot reference its own needs outputs.",
		IntroducedIn: "0.9.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Track if any transformations were made
			modified := false
			result := content

			for _, output := range activationOutputPatterns {
				newReplacement := "steps.sanitized.outputs." + output.name

				// Use regex with word boundary to prevent partial matches
				// This ensures we don't match things like "needs.activation.outputs.text_custom"
				// The pattern matches the old expression followed by a non-word character or end of string
				pattern := output.pattern

				// Check if pattern exists in content
				if pattern.MatchString(result) {
					// Perform the replacement
					newContent := pattern.ReplaceAllString(result, newReplacement)
					if newContent != result {
						modified = true
						activationOutputsCodemodLog.Printf("Transformed needs.activation.outputs.%s to steps.sanitized.outputs.%s", output.name, output.name)
						result = newContent
					}
				}
			}

			return result, modified, nil
		},
	}
}
