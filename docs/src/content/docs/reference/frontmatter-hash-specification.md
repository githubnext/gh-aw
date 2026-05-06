---
title: Frontmatter Hash Specification
description: Specification for computing deterministic hashes of agentic workflow frontmatter
---

# Frontmatter Hash Specification

This document specifies the algorithm for computing a deterministic hash of agentic workflow frontmatter, including contributions from imported workflows.

## Purpose

The frontmatter hash provides:
1. **Stale lock detection**: Identify when the compiled lock file is out of sync with the source workflow (e.g. after editing the `.md` file without recompiling)
2. **Reproducibility**: Ensure identical configurations produce identical hashes across languages (Go and JavaScript)
3. **Change detection**: Verify that workflow configuration has not changed between compilation and execution

## Hash Algorithm

### 1. Input Collection

Collect all frontmatter from the main workflow and all imported workflows in **breadth-first order** (BFS traversal):

1. **Main workflow frontmatter**: The frontmatter from the root workflow file
2. **Imported workflow frontmatter**: Frontmatter from each imported file in BFS processing order
   - Includes transitively imported files (imports of imports)
   - Agent files (`.github/agents/*.md`) only contribute markdown content, not frontmatter

#### BFS Traversal and Tie-Breaking Rules

The BFS traversal processes imports level by level, starting from the root workflow. When a workflow imports multiple files, they are enqueued left-to-right in the order they appear in the `imports:` list. This ordering is preserved at every level.

**Diamond-import handling**: If a workflow file appears more than once in the import graph (a "diamond" dependency), the **first occurrence** in BFS order determines where that file's frontmatter is merged; all subsequent occurrences of the same file **MUST be silently skipped**. Implementations MUST detect duplicate import paths using canonical path comparison (case-sensitive, no trailing-slash normalization) and discard duplicates without error.

**Example (diamond graph)**:

```
root.md  →  imports: [a.md, b.md]
a.md     →  imports: [shared.md]
b.md     →  imports: [shared.md]
```

BFS queue order: `[root.md, a.md, b.md, shared.md]`  
`shared.md` appears twice but is processed only once (after `a.md` in queue order).  
Canonical hash input order: root → a → b → shared.

This rule ensures that the hash is deterministic regardless of which traversal path first discovers a shared dependency.

### 2. Field Selection

Include the following frontmatter fields in the hash computation:

**Core Configuration:**
- `engine` - AI engine specification
- `on` - Workflow triggers
- `permissions` - GitHub Actions permissions
- `tracker-id` - Workflow tracker identifier

**Tool and Integration:**
- `tools` - Tool configurations (GitHub, Playwright, etc.)
- `mcp-servers` - MCP server configurations
- `network` - Network access permissions
- `safe-outputs` - Safe output configurations
- `mcp-scripts` - Safe input configurations

**Runtime Configuration:**
- `runtimes` - Runtime version specifications (Node.js, Python, etc.)
- `services` - Container services
- `cache` - Caching configuration

**Workflow Structure:**
- `steps` - Custom workflow steps
- `post-steps` - Post-execution steps
- `jobs` - GitHub Actions job definitions

**Metadata:**
- `description` - Workflow description
- `labels` - Workflow labels
- `bots` - Authorized bot list
- `timeout-minutes` - Workflow timeout
- `secret-masking` - Secret masking configuration

**Import Metadata:**
- `imports` - List of imported workflow paths (for traceability)
- `inputs` - Input parameter definitions

**Excluded Fields:**
- Markdown body content (not part of frontmatter)
- Comments and whitespace variations
- Field ordering (normalized during processing)

### 3. Canonical JSON Serialization

Transform the collected frontmatter into a canonical JSON representation:

#### 3.1 Merge Strategy

For each workflow in BFS order:
1. Parse frontmatter into a structured object
2. Merge with accumulated frontmatter using these rules:
   - **Replace**: `engine`, `on`, `tracker-id`, `description`, `timeout-minutes`
   - **Deep merge**: `tools`, `mcp-servers`, `network`, `permissions`, `runtimes`, `cache`, `services`
   - **Append**: `steps`, `post-steps`, `safe-outputs`, `mcp-scripts`, `jobs`
   - **Union**: `labels`, `bots` (deduplicated)
   - **Track**: `imports` (list of all imported paths)

#### 3.2 Normalization Rules

Apply these normalization rules to ensure deterministic output:

1. **Key Sorting**: Sort all object keys alphabetically at every level
2. **Array Ordering**: Preserve array order as-is (no sorting of array elements)
3. **Whitespace**: Use minimal whitespace (no pretty-printing)
4. **Number Format**: Represent numbers without exponents (e.g., `120` not `1.2e2`)
5. **Boolean Values**: Use lowercase `true` and `false`
6. **Null Handling**: Include `null` values explicitly
7. **Empty Containers**: Include empty objects `{}` and empty arrays `[]`
8. **String Escaping**: Use JSON standard escaping (quotes, backslashes, control characters)

#### 3.3 Serialization Format

The canonical JSON includes all frontmatter fields plus version information:

```json
{
  "bots": ["copilot"],
  "cache": {},
  "description": "Daily audit of workflow runs",
  "engine": "claude",
  "imports": ["shared/mcp/gh-aw.md", "shared/jqschema.md"],
  "jobs": {},
  "labels": ["audit", "automation"],
  "mcp-servers": {},
  "network": {"allowed": ["api.github.com"]},
  "on": {"schedule": "daily"},
  "permissions": {"actions": "read", "contents": "read"},
  "post-steps": [],
  "runtimes": {"node": {"version": "20"}},
  "mcp-scripts": {},
  "safe-outputs": {"create-discussion": {"category": "audits"}},
  "services": {},
  "steps": [],
  "template-expressions": ["${{ env.MY_VAR }}"],
  "timeout-minutes": 30,
  "tools": {"repo-memory": {"branch-name": "memory/audit"}},
  "tracker-id": "audit-workflows-daily",
  "versions": {
    "agents": "v0.0.84",
    "awf": "v0.11.2",
    "gh-aw": "dev"
  }
}
```

### 4. Version Information

The hash includes version numbers to ensure hash changes when dependencies are upgraded:

- **gh-aw**: The compiler version (e.g., "0.1.0" or "dev")
- **awf**: The firewall version (e.g., "v0.11.2")
- **agents**: The MCP gateway version (e.g., "v0.0.84")

This ensures that upgrading any component invalidates existing hashes.

1. **Serialize**: Convert the merged and normalized frontmatter to canonical JSON
2. **Add Versions**: Include version information for gh-aw, awf (firewall), and agents (MCP gateway)
3. **Hash**: Compute SHA-256 hash of the JSON string (UTF-8 encoded)
4. **Encode**: Represent the hash as a lowercase hexadecimal string (64 characters)

**Example:**
```
Input JSON: {"engine":"copilot","on":{"schedule":"daily"},"versions":{"agents":"v0.0.84","awf":"v0.11.2","gh-aw":"dev"}}
SHA-256: a1b2c3d4e5f6...  (64 hex characters)
```

### 5. Cross-Language Consistency

Both Go and JavaScript implementations MUST:
- Use the same field selection and merging rules
- Produce identical canonical JSON (byte-for-byte)
- Use SHA-256 hash function
- Encode output as lowercase hexadecimal

**Test cases** must verify identical hashes across both implementations for:
- Empty frontmatter
- Single-file workflows (no imports)
- Multi-level imports (2+ levels deep)
- All field types (strings, numbers, booleans, arrays, objects)
- Special characters and escaping
- All workflows in the repository

## Implementation Notes

### Go Implementation

The current Go implementation (`pkg/parser/frontmatter_hash.go`) uses a **text-based approach** that diverges from the field-selection model described in Section 2 ("Field Selection") of this specification:

- **Actual behavior**: The entire normalized frontmatter text is hashed as a single opaque string (`frontmatter-text` key in the canonical JSON), alongside a sorted list of imported file paths and their normalized texts. This means _all_ frontmatter fields — including excluded ones such as comments — affect the hash value.
- **Specified behavior**: The specification calls for selecting individual named fields and merging them by type (replace, deep-merge, append, union).

**Implication**: The text-based approach is more conservative (any frontmatter change invalidates the hash, including whitespace-only changes after normalization) and simpler to implement cross-language. The trade-off is that it cannot support selective field exclusion without modifying the text normalization step.

**Sync status** (verified 2026-05-06): The Go implementation is consistent with the JavaScript implementation in `actions/setup/js/` for the text-based approach. Both produce identical hashes for the same input. The field-selection model in Section 2 documents the _logical_ intent; the text-based implementation is the authoritative runtime behavior until a future revision aligns them.

- Use `crypto/sha256` for hashing (`crypto/sha256.Sum256`)
- Use `hex.EncodeToString()` for hexadecimal encoding

### JavaScript Implementation

- Uses the same text-based approach as the Go implementation
- Uses Node.js `crypto.createHash('sha256')` for hashing
- Uses `.digest('hex')` for hexadecimal encoding
- The JavaScript cross-language test suite in `pkg/parser/frontmatter_hash_cross_language_test.go` verifies identical output between the two implementations

### Hash Storage and Verification

1. **Compilation**: The Go compiler computes the hash and writes it to the workflow log file
2. **Execution**: The JavaScript custom action:
   - Reads the hash from the log file
   - Recomputes the hash from the workflow file
   - Compares the two hashes
   - Creates a GitHub issue if they differ (indicating frontmatter modification)

## Safeguards

This section describes known risks associated with the frontmatter hash mechanism and the recommended mitigations.

### S-1: Hash Collision Risk

SHA-256 produces a 256-bit output, giving a collision probability of approximately 2⁻¹²⁸ for any two distinct inputs under the birthday paradox. For the expected number of compiled workflows in a repository (typically <10,000), the probability of an accidental collision is negligible and does not require mitigation at the application layer.

However, implementations MUST NOT rely on the hash as a cryptographic commitment or security boundary. The hash is an integrity check for stale-lock detection only.

**Mitigation**: If future use cases require stronger collision resistance (e.g., content-addressed storage), implementations SHOULD upgrade to SHA-512 or SHA3-256 and bump the specification version.

### S-2: Tamper Detection Limits

The frontmatter hash detects accidental drift between the `.md` source and the compiled `.lock.yml` file. It does **not** prevent intentional tampering. Any user with write access to the repository can modify both files simultaneously:

1. Edit the `.md` source.
2. Recompile to regenerate the `.lock.yml` with the new hash.
3. Commit both files in a single push.

This bypass is by design — the hash mechanism is intended to catch _accidental_ stale locks, not to enforce a security boundary.

**Mitigation**: Enforce required code reviews via branch protection rules. Require signed commits for critical workflows. Use separate compilation and merge workflows with protected branches to prevent direct pushes to the default branch.

### S-3: Inclusion of Sensitive Configuration in Hash Input

The canonical JSON used for hash computation includes all frontmatter fields, some of which may encode sensitive topology information (e.g., MCP server addresses in `mcp-servers:`, secret names in `mcp-scripts:`, or branch names in `tools.repo-memory`). This information is embedded in the `.lock.yml` file at compile time and is visible to anyone who can read the repository.

**Mitigation**: Treat repository visibility as the primary access control boundary. Avoid storing secret _values_ in frontmatter (use GitHub Actions secrets instead). Periodically audit lock files for inadvertently committed sensitive configuration.

### S-4: Version-Bump-Forced Recompilation

The hash includes `versions.gh-aw`, `versions.awf`, and `versions.agents`. Upgrading any of these components will invalidate all existing hashes, triggering stale-lock warnings on all workflows until they are recompiled. In a repository with many workflows, this can create a noisy wave of false-positive stale-lock issues.

**Mitigation**: Coordinate component upgrades with a bulk `make recompile` step. Automate recompilation in the upgrade PR so that lock files are always fresh after a version bump.

### S-5: Cross-Language Hash Divergence

The Go and JavaScript implementations must produce byte-for-byte identical canonical JSON. Any divergence in key sorting, number representation, or null/undefined handling between the two implementations will cause the JavaScript runtime to report a false stale-lock mismatch for every workflow run.

**Mitigation**: Maintain a shared test-vector file (at minimum: empty frontmatter, single-field workflow, multi-level imports, all field types). Run cross-language hash tests in CI. Any change to the serialization algorithm in either language MUST be accompanied by updated test vectors verified against both implementations.

---

## Security Considerations

- The hash is **not cryptographically secure** for authentication (no HMAC/signing)
- The hash is designed to **detect stale lock files** — it catches cases where the frontmatter has changed since the lock file was last compiled
- The hash **does not guarantee tamper protection**: anyone with write access to the repository can modify both the `.md` source and the `.lock.yml` file together, bypassing detection
- Always validate workflow sources through proper code review processes

## Versioning

This is version 1.0 of the frontmatter hash specification.

Future versions may:
- Add additional fields
- Change normalization rules
- Use different hash algorithms

Version changes will be documented and backward compatibility maintained where possible.
