package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var cacheIntegrityLog = logger.New("workflow:cache_integrity")

// defaultCacheIntegrityLevel is the integrity level used when no guard policy is configured.
const defaultCacheIntegrityLevel = "none"

// noPolicySentinel is the policy hash used for workflows without an allow-only policy.
const noPolicySentinel = "nopolicy"

// computePolicyHash computes a deterministic 8-character hex hash of the allow-only policy.
// Returns noPolicySentinel when the GitHub tool has no guard policy (i.e., min-integrity is unset).
//
// The hash is computed over the canonical form of all policy fields so that:
//   - Same policy in different order → same hash (sorted, deduped lists)
//   - Any policy field change → new hash → cache miss (correct isolation)
//   - Workflows without policy → sentinel value "nopolicy" (consistent key format)
func computePolicyHash(github *GitHubToolConfig) string {
	if github == nil || github.MinIntegrity == "" {
		cacheIntegrityLog.Print("No guard policy configured, using nopolicy sentinel")
		return noPolicySentinel
	}

	canonical := buildCanonicalPolicy(github)
	hash := sha256.Sum256([]byte(canonical))
	result := hex.EncodeToString(hash[:])[:8]
	cacheIntegrityLog.Printf("Computed policy hash: %s (min-integrity=%s)", result, github.MinIntegrity)
	return result
}

// buildCanonicalPolicy builds the normalized string representation of the allow-only policy.
// All fields are always present (empty if unset), sorted and deduplicated, so the result
// is deterministic regardless of input ordering.
func buildCanonicalPolicy(github *GitHubToolConfig) string {
	var sb strings.Builder

	// blocked-users: sorted, lowercased, deduplicated literal list.
	// When blocked-users is provided as a GitHub Actions expression (BlockedUsersExpr),
	// include it verbatim so that changing the expression produces a different hash.
	sb.WriteString("blocked-users:")
	if github.BlockedUsersExpr != "" {
		// Expression-based: include the raw expression as the canonical form.
		// This ensures that different expressions produce different hashes and that
		// switching from a literal list to an expression (or vice versa) invalidates the cache.
		sb.WriteString("expr:")
		sb.WriteString(github.BlockedUsersExpr)
	} else {
		sb.WriteString(canonicalUserList(github.BlockedUsers))
	}
	sb.WriteString("\n")

	// min-integrity
	sb.WriteString("min-integrity:")
	sb.WriteString(string(github.MinIntegrity))
	sb.WriteString("\n")

	// repos: canonical scope form (sorted array or fixed string)
	sb.WriteString("repos:")
	sb.WriteString(canonicalReposScope(github.AllowedRepos))
	sb.WriteString("\n")

	// trusted-bots: reserved for future use (always empty today)
	sb.WriteString("trusted-bots:\n")

	// trusted-users: sorted, lowercased, deduplicated literal list (via canonicalUserList).
	// When trusted-users is provided as a GitHub Actions expression (TrustedUsersExpr),
	// include it verbatim so that changing the expression produces a different hash.
	sb.WriteString("trusted-users:\n")
	if github.TrustedUsersExpr != "" {
		sb.WriteString("expr:")
		sb.WriteString(github.TrustedUsersExpr)
		sb.WriteString("\n")
	} else {
		users := canonicalUserList(github.TrustedUsers)
		if users != "" {
			sb.WriteString(users)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// canonicalUserList converts a list of user names to a canonical form:
// sorted, lowercased, deduplicated, joined with "\n".
// Returns an empty string for nil or empty lists.
func canonicalUserList(users []string) string {
	if len(users) == 0 {
		return ""
	}

	// Lowercase all entries
	normalized := make([]string, len(users))
	for i, u := range users {
		normalized[i] = strings.ToLower(u)
	}

	// Deduplicate
	seen := make(map[string]struct{}, len(normalized))
	deduped := normalized[:0]
	for _, u := range normalized {
		if _, exists := seen[u]; !exists {
			seen[u] = struct{}{}
			deduped = append(deduped, u)
		}
	}

	// Sort
	sort.Strings(deduped)

	return strings.Join(deduped, "\n")
}

// canonicalReposScope converts a GitHubReposScope to its canonical string form.
//
// Canonical forms:
//   - "all"            → "all"
//   - "public"         → "public"
//   - ["b","a"]        → "a\nb"   (sorted, lowercased)
//   - nil              → ""
func canonicalReposScope(repos GitHubReposScope) string {
	if repos == nil {
		return ""
	}

	switch v := repos.(type) {
	case string:
		// Simple string scope: "all" or "public"
		return strings.ToLower(v)

	case []any:
		// Array of repository patterns: sort, lowercase, deduplicate
		strs := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				strs = append(strs, strings.ToLower(s))
			}
		}
		sort.Strings(strs)
		// Deduplicate
		deduped := strs[:0]
		for i, s := range strs {
			if i == 0 || s != strs[i-1] {
				deduped = append(deduped, s)
			}
		}
		return strings.Join(deduped, "\n")

	case []string:
		// Typed string slice
		strs := make([]string, len(v))
		for i, s := range v {
			strs[i] = strings.ToLower(s)
		}
		sort.Strings(strs)
		// Deduplicate
		deduped := strs[:0]
		for i, s := range strs {
			if i == 0 || s != strs[i-1] {
				deduped = append(deduped, s)
			}
		}
		return strings.Join(deduped, "\n")

	default:
		// Unexpected type: return empty string for deterministic hash computation
		// rather than using fmt.Sprintf which could produce inconsistent results.
		return ""
	}
}

// cacheIntegrityLevel returns the integrity level string for cache key generation.
// Returns defaultCacheIntegrityLevel when no guard policy is configured.
func cacheIntegrityLevel(github *GitHubToolConfig) string {
	if github == nil || github.MinIntegrity == "" {
		return defaultCacheIntegrityLevel
	}
	return string(github.MinIntegrity)
}

// generateIntegrityAwareCacheKey generates the new-format cache key that includes
// the integrity level and policy hash as prefixes.
//
// Format: memory-{integrityLevel}-{policyHash}-[{cacheID}-]{workflowID}-{runID}
//
// The cacheID segment is omitted for the "default" cache ID to maintain a clean key.
// Examples:
//
//	memory-unapproved-7e4d9f12-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}
//	memory-none-nopolicy-session-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}
func generateIntegrityAwareCacheKey(cacheID, integrityLevel, policyHash string) string {
	var key string
	if cacheID == "default" || cacheID == "" {
		key = fmt.Sprintf(
			"memory-%s-%s-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}",
			integrityLevel, policyHash,
		)
	} else {
		key = fmt.Sprintf(
			"memory-%s-%s-%s-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}",
			integrityLevel, policyHash, cacheID,
		)
	}
	cacheIntegrityLog.Printf("Generated integrity-aware cache key: cacheID=%s, integrityLevel=%s, policyHash=%s", cacheID, integrityLevel, policyHash)
	return key
}
