package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ruleIDPattern matches valid CTR-* rule identifiers (e.g. CTR-001, CTR-INJ-001).
var ruleIDPattern = regexp.MustCompile(`^CTR-[A-Z0-9]+(-[A-Z0-9]+)*$`)

// ThreatDetectionSuppression represents a single compiler threat-detection rule suppression.
// It corresponds to one entry in the `threat-detection-suppress` frontmatter field.
type ThreatDetectionSuppression struct {
	// Rule is the CTR-* rule identifier being suppressed (e.g. "CTR-001").
	Rule string `json:"rule"`
	// Reason is a human-readable explanation. Required — compiler rejects suppressions
	// without a reason.
	Reason string `json:"reason"`
	// Expires is an optional ISO 8601 date (YYYY-MM-DD) after which the suppression is
	// no longer active. When empty the suppression is treated as permanent.
	Expires string `json:"expires,omitempty"`
}

// IsExpired reports whether the suppression has passed its expiry date.
// Returns false when Expires is empty (permanent suppression).
func (s *ThreatDetectionSuppression) IsExpired(now time.Time) bool {
	if s.Expires == "" {
		return false
	}
	expiry, err := time.Parse("2006-01-02", s.Expires)
	if err != nil {
		return false
	}
	return now.After(expiry)
}

// parseThreatDetectionSuppress parses the raw `threat-detection-suppress` frontmatter
// value into a typed slice. It validates each entry and returns an error if any entry
// is missing a required field or has an invalid format.
func parseThreatDetectionSuppress(raw any) ([]ThreatDetectionSuppression, error) {
	if raw == nil {
		return nil, nil
	}

	items, ok := raw.([]any)
	if !ok {
		return nil, errors.New("threat-detection-suppress must be an array of suppression objects")
	}

	suppressions := make([]ThreatDetectionSuppression, 0, len(items))
	for i, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("threat-detection-suppress[%d]: each entry must be an object with 'rule' and 'reason'", i)
		}

		sup, err := parseSingleSuppression(i, entry)
		if err != nil {
			return nil, err
		}

		if sup.IsExpired(time.Now().UTC()) {
			frontmatterTypesLog.Printf("threat-detection-suppress[%d]: suppression for rule %q expired on %s — remove or update the entry", i, sup.Rule, sup.Expires)
		}

		suppressions = append(suppressions, sup)
	}
	return suppressions, nil
}

// parseSingleSuppression validates and converts one raw suppression map entry.
func parseSingleSuppression(idx int, entry map[string]any) (ThreatDetectionSuppression, error) {
	var sup ThreatDetectionSuppression

	// --- rule (required, CTR-* pattern) ---
	ruleRaw, hasRule := entry["rule"]
	if !hasRule {
		return sup, fmt.Errorf("threat-detection-suppress[%d]: missing required field 'rule'", idx)
	}
	rule, ok := ruleRaw.(string)
	if !ok || rule == "" {
		return sup, fmt.Errorf("threat-detection-suppress[%d]: 'rule' must be a non-empty string", idx)
	}
	if !ruleIDPattern.MatchString(rule) {
		return sup, fmt.Errorf(
			"threat-detection-suppress[%d]: 'rule' %q does not match the required CTR-* pattern (e.g. CTR-001, CTR-INJ-001)",
			idx, rule)
	}
	sup.Rule = rule

	// --- reason (required, non-empty) ---
	reasonRaw, hasReason := entry["reason"]
	if !hasReason {
		return sup, fmt.Errorf(
			"threat-detection-suppress[%d]: missing required field 'reason' for rule %q — "+
				"every suppression must include a human-readable explanation",
			idx, rule)
	}
	reason, ok := reasonRaw.(string)
	if !ok || strings.TrimSpace(reason) == "" {
		return sup, fmt.Errorf(
			"threat-detection-suppress[%d]: 'reason' for rule %q must be a non-empty string — "+
				"provide a human-readable explanation for the suppression",
			idx, rule)
	}
	sup.Reason = reason

	// --- expires (optional, ISO 8601 YYYY-MM-DD) ---
	if expiresRaw, hasExpires := entry["expires"]; hasExpires {
		expires, ok := expiresRaw.(string)
		if !ok || expires == "" {
			return sup, fmt.Errorf("threat-detection-suppress[%d]: 'expires' for rule %q must be a date string (YYYY-MM-DD)", idx, rule)
		}
		if _, err := time.Parse("2006-01-02", expires); err != nil {
			return sup, fmt.Errorf(
				"threat-detection-suppress[%d]: 'expires' %q for rule %q is not a valid ISO 8601 date (expected YYYY-MM-DD)",
				idx, expires, rule)
		}
		sup.Expires = expires
	}

	return sup, nil
}

// buildSuppressionManifestEntries formats active suppressions for inclusion in the
// lock file manifest section. Each entry is emitted as a comment line.
// Permanent suppressions (no expiry) are flagged with SLA_BREACH.
// Expired suppressions are omitted.
func buildSuppressionManifestEntries(suppressions []ThreatDetectionSuppression, now time.Time) []string {
	if len(suppressions) == 0 {
		return nil
	}

	var lines []string
	for _, sup := range suppressions {
		if sup.IsExpired(now) {
			// Expired suppressions are not recorded in the manifest.
			continue
		}
		// Permanent suppressions (no expiry) trigger SLA tracking for reject-level rules.
		if sup.Expires == "" {
			lines = append(lines, fmt.Sprintf("# SLA_BREACH: suppress rule=%s reason=%q (no expiry set — add expires: YYYY-MM-DD to acknowledge)", sup.Rule, sup.Reason))
		} else {
			lines = append(lines, fmt.Sprintf("# suppress rule=%s reason=%q expires=%s", sup.Rule, sup.Reason, sup.Expires))
		}
	}
	return lines
}
