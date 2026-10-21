package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var requestReviewPolicyCodemodLog = logger.New("cli:codemod_request_review_policy")

func getRequestReviewPolicyCodemod() Codemod {
	return Codemod{
		ID:           "request-review-policy-normalization",
		Name:         "Normalize request_review protected-file policies",
		Description:  "Normalizes the legacy request_review protected-file policy value to request-review in pull-request safe outputs.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !requestReviewPolicyNeedsMigration(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				for _, handler := range []string{"create-pull-request", "push-to-pull-request-branch"} {
					for _, suffix := range [][]string{{"protected-files"}, {"protected-files-policy"}, {"protected-files", "policy"}} {
						path := append([]string{"safe-outputs", handler}, suffix...)
						value, _ := f.Lookup(path...)
						if policy, ok := value.(string); ok && policy == "request_review" {
							if err := f.Set(path, "request-review"); err != nil {
								return err
							}
						}
					}
				}
				return nil
			})
			if applied {
				requestReviewPolicyCodemodLog.Print("Normalized request_review protected-file policy")
			}
			return newContent, applied, err
		},
	}
}

func requestReviewPolicyNeedsMigration(frontmatter map[string]any) bool {
	safeOutputs, ok := frontmatter["safe-outputs"].(map[string]any)
	if !ok {
		return false
	}

	for _, handlerName := range []string{"create-pull-request", "push-to-pull-request-branch"} {
		handler, ok := safeOutputs[handlerName].(map[string]any)
		if !ok {
			continue
		}
		if policy, ok := handler["protected-files"].(string); ok && policy == "request_review" {
			return true
		}
		if policy, ok := handler["protected-files-policy"].(string); ok && policy == "request_review" {
			return true
		}
		protectedFiles, ok := handler["protected-files"].(map[string]any)
		if ok {
			if policy, ok := protectedFiles["policy"].(string); ok && policy == "request_review" {
				return true
			}
		}
	}
	return false
}
