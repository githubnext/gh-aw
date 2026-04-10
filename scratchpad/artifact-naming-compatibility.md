# Artifact Naming Backward/Forward Compatibility

## Overview

The `gh aw logs` and `gh aw audit` commands maintain full backward and forward compatibility with both old and new artifact naming schemes.

## How It Works

### Artifact Download Process

1. **GitHub Actions Upload**: Workflows upload files with artifact names:
   - Old naming (pre-v5): `aw_info.json`, `safe_output.jsonl`, `agent_output.json`, `prompt.txt`
   - New naming (v5+): `aw-info`, `safe-output`, `agent-output`, `prompt`

2. **GitHub CLI Download**: When running `gh run download <run-id>`:
   - Creates a directory for each artifact using the artifact name
   - Extracts files into that directory preserving original filenames
   - Example: Artifact `aw-info` containing `aw_info.json` → `aw-info/aw_info.json`

3. **Flattening**: The `flattenSingleFileArtifacts()` function:
   - Detects directories containing exactly one file
   - Moves the file to the root directory
   - Removes the empty artifact directory
   - Example: `aw-info/aw_info.json` → `aw_info.json`

4. **CLI Commands**: Both `logs` and `audit` commands expect files at root:
   - `aw_info.json` - Engine configuration
   - `safe_output.jsonl` - Safe outputs
   - `agent_output.json` - Agent outputs
   - `prompt.txt` - Input prompt

## Compatibility Matrix

### Single-File Artifacts

These artifacts contain exactly one file and are flattened to the root directory by `flattenSingleFileArtifacts()`:

| Artifact Name (Old) | Artifact Name (New) | File in Artifact | After Flattening | CLI Expects |
|---------------------|---------------------|------------------|------------------|-------------|
| `aw_info.json` | `aw-info` | `aw_info.json` | `aw_info.json` | ✅ |
| `safe_output.jsonl` | `safe-output` | `safe_output.jsonl` | `safe_output.jsonl` | ✅ |
| `agent_output.json` | `agent-output` | `agent_output.json` | `agent_output.json` | ✅ |
| `prompt.txt` | `prompt` | `prompt.txt` | `prompt.txt` | ✅ |
| `threat-detection.log` | `detection` | `detection.log` | `detection.log` | ✅ |

### Multi-File Artifacts

These artifacts are initially downloaded by `gh run download` as directory trees that retain their internal structure. However, unlike the single-file artifact handling above, `gh aw logs` / `gh aw audit` may perform additional post-processing for some multi-file artifacts (notably `agent` and `activation`) to move expected files into the final layout used by the CLI.

| Artifact Name | Constant | Contents | Notes |
|---------------|----------|----------|-------|
| `firewall-audit-logs` | `constants.FirewallAuditArtifactName` | AWF structured audit/observability logs | Uploaded by all firewall-enabled workflows; retains directory structure after download |
| `agent` | `constants.AgentArtifactName` | Unified agent job outputs (logs, safe outputs, token usage) | Downloaded as a directory tree, then post-processed by CLI flattening/reorganization helpers |
| `activation` | `constants.ActivationArtifactName` | Activation job output (`aw_info.json`, `prompt.txt`) | Downloaded as a directory tree, then post-processed by CLI flattening helpers for downstream use |

#### `firewall-audit-logs` Directory Structure

The `firewall-audit-logs` artifact (constant: `constants.FirewallAuditArtifactName`) is uploaded by all firewall-enabled agentic workflows. It is **separate** from the `agent` artifact and must be downloaded independently.

```
firewall-audit-logs/
├── api-proxy-logs/
│   └── token-usage.jsonl        ← Token usage data (input/output/cache tokens per request)
├── squid-logs/
│   └── access.log               ← Network policy log (domain allow/deny decisions)
├── audit.jsonl                  ← Firewall audit trail (policy matches, rule evaluations)
└── policy-manifest.json         ← Policy configuration snapshot
```

**Downloading firewall audit logs with `gh run download`:**

```bash
# Download only the firewall-audit-logs artifact
gh run download <run-id> -n firewall-audit-logs

# The data is then at:
#   firewall-audit-logs/api-proxy-logs/token-usage.jsonl
#   firewall-audit-logs/squid-logs/access.log
#   firewall-audit-logs/audit.jsonl
#   firewall-audit-logs/policy-manifest.json
```

**Recommended: Use `gh aw logs` instead of `gh run download`:**

The `gh aw logs` command knows the correct artifact names and handles backward compatibility automatically:

```bash
# Download and analyze all logs (including firewall data)
gh aw logs <run-id>

# Download only firewall artifacts
gh aw logs <run-id> --artifacts firewall

# Output as JSON for programmatic use
gh aw logs <run-id> --artifacts firewall --json
```

> **⚠️ Common mistake:** Downloading `agent-artifacts` or `agent` and expecting to find `token-usage.jsonl` there. Token usage data lives in the `firewall-audit-logs` artifact, not in the agent artifact.

## Testing

Comprehensive tests ensure compatibility:
- `TestArtifactNamingBackwardCompatibility`: Tests both old and new naming
- `TestAuditCommandFindsNewArtifacts`: Verifies audit command works with new names
- `TestFlattenSingleFileArtifactsWithAuditFiles`: Tests flattening with new names

## Key Insight

The separation of concerns ensures compatibility:
- **Artifact Names**: Metadata for GitHub Actions (can change)
- **File Names**: Actual file content (preserved)
- **Flattening**: Bridges the gap between artifact structure and CLI expectations

This design means the CLI doesn't need to know about artifact naming changes - it always looks for the same filenames at the root level, regardless of how they were packaged as artifacts.
