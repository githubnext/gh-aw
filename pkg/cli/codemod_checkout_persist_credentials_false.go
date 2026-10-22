package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var checkoutPersistCredentialsFalseCodemodLog = logger.New("cli:codemod_checkout_persist_credentials_false")

// getCheckoutPersistCredentialsFalseCodemod ensures checkout steps set with.persist-credentials: false.
func getCheckoutPersistCredentialsFalseCodemod() Codemod {
	return Codemod{
		ID:           "checkout-persist-credentials-false",
		Name:         "Add persist-credentials: false to checkout steps",
		Description:  "Ensures actions/checkout steps set with.persist-credentials: false in steps-like sections for strict-mode safety.",
		IntroducedIn: "1.0.44",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			agentSections := []string{"pre-steps", "steps", "post-steps", "pre-agent-steps"}
			if !hasTopLevelSection(frontmatter, agentSections) && !hasAgentJobSection(frontmatter, agentSections) {
				return content, false, nil
			}

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Edit(func(document *aw.Document) error {
					scopes := []map[string]any{document.Frontmatter}
					if jobs, ok := document.Frontmatter["jobs"].(map[string]any); ok {
						if agent, ok := jobs["agent"].(map[string]any); ok {
							scopes = append(scopes, agent)
						}
					}
					for _, scope := range scopes {
						for _, section := range agentSections {
							steps, ok := scope[section].([]any)
							if !ok {
								continue
							}
							for _, value := range steps {
								step, ok := value.(map[string]any)
								if !ok {
									continue
								}
								ensureCheckoutPersistCredentials(step)
							}
						}
					}
					return nil
				})
			})
			if applied {
				checkoutPersistCredentialsFalseCodemodLog.Print("Added persist-credentials: false to actions/checkout step with blocks")
			}
			return newContent, applied, err
		},
	}
}

func ensureCheckoutPersistCredentials(step map[string]any) {
	uses, ok := step["uses"].(string)
	if !ok || !isCheckoutUsesValue(uses) {
		return
	}
	with, ok := step["with"].(map[string]any)
	if !ok {
		if value, exists := step["with"]; exists && value != nil {
			return
		}
		with = map[string]any{}
		step["with"] = with
	}
	if _, exists := with["persist-credentials"]; !exists {
		with["persist-credentials"] = false
	}
}

func hasTopLevelSection(frontmatter map[string]any, sections []string) bool {
	for _, section := range sections {
		if _, ok := frontmatter[section]; ok {
			return true
		}
	}
	return false
}

func hasAgentJobSection(frontmatter map[string]any, sections []string) bool {
	jobsMap, ok := frontmatter["jobs"].(map[string]any)
	if !ok {
		return false
	}
	agentMap, ok := jobsMap["agent"].(map[string]any)
	return ok && hasTopLevelSection(agentMap, sections)
}

func isCheckoutUsesValue(raw string) bool {
	value := strings.TrimSpace(raw)
	value = strings.Trim(value, "\"'")
	value = strings.ToLower(value)
	return strings.HasPrefix(value, "actions/checkout@") || value == "actions/checkout"
}
