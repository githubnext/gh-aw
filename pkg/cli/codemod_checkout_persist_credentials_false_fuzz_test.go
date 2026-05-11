//go:build !integration

package cli

import (
	"strings"
	"testing"
)

const maxCheckoutRefLength = 16

func FuzzCheckoutPersistCredentialsFalseCodemod(f *testing.F) {
	f.Add(true, "@v5", false, false, false, false, uint8(0))
	f.Add(true, "", true, false, false, false, uint8(1))
	f.Add(true, "@v4", true, true, true, false, uint8(2))
	f.Add(true, "@v4", true, true, false, false, uint8(3))
	f.Add(false, "@v4", false, false, false, false, uint8(0))

	f.Fuzz(func(t *testing.T, isCheckout bool, ref string, hasWith bool, hasPersist bool, persistTrue bool, inlineUses bool, sectionSelector uint8) {
		codemod := getCheckoutPersistCredentialsFalseCodemod()
		section := []string{"pre-steps", "steps", "post-steps", "pre-agent-steps"}[int(sectionSelector)%4]

		ref = sanitizeCheckoutRef(ref)
		uses := "actions/cache@v4"
		if isCheckout {
			if ref == "" {
				uses = "actions/checkout"
			} else {
				uses = "actions/checkout" + ref
			}
		}

		step := map[string]any{}
		if inlineUses {
			step["uses"] = uses
		} else {
			step["name"] = "checkout-step"
			step["uses"] = uses
		}

		if hasWith {
			with := map[string]any{"fetch-depth": 0}
			if hasPersist {
				with["persist-credentials"] = persistTrue
			}
			step["with"] = with
		}

		frontmatter := map[string]any{
			"on":       "push",
			section:    []any{step},
			"workflow": "fuzz",
		}

		content := buildCheckoutFuzzContent(section, uses, hasWith, hasPersist, persistTrue, inlineUses)

		result, applied, err := codemod.Apply(content, frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectMutation := isCheckout && (!hasWith || !hasPersist)

		if expectMutation {
			if !applied {
				t.Fatalf("expected codemod to apply; section=%s uses=%s", section, uses)
			}
			if !strings.Contains(result, "persist-credentials: false") {
				t.Fatalf("expected result to include persist-credentials: false")
			}
			return
		}

		if applied {
			t.Fatalf("expected codemod not to apply; section=%s uses=%s", section, uses)
		}
		if result != content {
			t.Fatalf("unexpected content mutation when codemod not applied")
		}
	})
}

func buildCheckoutFuzzContent(section, uses string, hasWith, hasPersist, persistTrue, inlineUses bool) string {
	usesLine := "    uses: " + uses
	if inlineUses {
		usesLine = "  - uses: " + uses
	}

	lines := []string{
		"---",
		"on: push",
		section + ":",
	}
	if !inlineUses {
		lines = append(lines, "  - name: checkout-step")
	}
	lines = append(lines, usesLine)

	if hasWith {
		lines = append(lines, "    with:", "      fetch-depth: 0")
		if hasPersist {
			value := "false"
			if persistTrue {
				value = "true"
			}
			lines = append(lines, "      persist-credentials: "+value)
		}
	}

	lines = append(lines, "---")
	return strings.Join(lines, "\n") + "\n"
}

func sanitizeCheckoutRef(ref string) string {
	if ref == "" {
		return ""
	}
	if len(ref) > maxCheckoutRefLength {
		ref = ref[:maxCheckoutRefLength]
	}
	var b strings.Builder
	for _, r := range ref {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		}
	}
	clean := b.String()
	if clean == "" {
		return ""
	}
	if clean[0] != '@' {
		return "@" + clean
	}
	return clean
}
