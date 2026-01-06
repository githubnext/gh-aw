# Secrets Information Flow - Technical Specification

**Status**: Completed  
**Date**: 2026-01-06  
**Analysis Scope**: 125 compiled `.lock.yml` workflow files

## Quick Reference

### Statistics
- **Workflows with Secrets**: 125 (100%)
- **Jobs with Secrets**: 570
- **Steps with Secrets**: 1,648
- **Unique Secret Types**: 25
- **github.token References**: 930

### Top 5 Secrets by Usage
1. `GITHUB_TOKEN` - 1,228 occurrences
2. `GH_AW_GITHUB_TOKEN` - 1,101 occurrences
3. `GH_AW_GITHUB_MCP_SERVER_TOKEN` - 736 occurrences
4. `COPILOT_GITHUB_TOKEN` - 405 occurrences
5. `CLAUDE_CODE_OAUTH_TOKEN` - 155 occurrences

## Information Flow Map

### Secret Journey Through System

```
┌──────────────────────────────────────────────────────┐
│ 1. GitHub Secrets Store (encrypted at rest)          │
└────────────────────┬─────────────────────────────────┘
                     │ Runtime injection
                     ↓
┌──────────────────────────────────────────────────────┐
│ 2. GitHub Actions Context (in-memory)                │
└────────────────────┬─────────────────────────────────┘
                     │ Workflow execution
                     ↓
     ┌───────────────┴────────────────┐
     ↓                                ↓
┌─────────────────┐        ┌──────────────────┐
│ 3a. Job Env     │        │ 3b. Step Env     │
│ (job scope)     │        │ (step scope)     │
└────────┬────────┘        └────────┬─────────┘
         │                          │
         └──────────┬───────────────┘
                    ↓
     ┌──────────────┴───────────────┐
     ↓                              ↓
┌─────────────────┐      ┌──────────────────┐
│ 4a. Env Vars    │      │ 4b. Action Input │
│ (process scope) │      │ (with: params)   │
└────────┬────────┘      └────────┬─────────┘
         ↓                        ↓
┌─────────────────┐      ┌──────────────────┐
│ 5a. Shell/Bash  │      │ 5b. JavaScript   │
│ Commands        │      │ github-script    │
└─────────────────┘      └──────────────────┘
```

### Storage Locations Analysis

| Location | State | Duration | Write to Disk | Access Method |
|----------|-------|----------|---------------|---------------|
| GitHub Secrets Store | Encrypted | Permanent | Yes (encrypted) | GitHub UI/API |
| Actions Context | In-memory | Workflow run | No | `${{ secrets.X }}` |
| Runner Memory | In-memory | Job duration | No | Process memory |
| Environment Vars | In-memory | Step/job | No | `$VAR_NAME` |
| Temp Files (potential) | Plaintext | Until cleanup | **Yes** ⚠️ | File operations |
| Artifacts | Encrypted | 90 days | Yes (encrypted) | Download |
| Logs | Masked text | 90 days | Yes (masked) | View logs |

**Key Risk**: Temporary files (`/tmp/gh-aw`) - This is why redaction system is critical.

## Secret Flow Patterns

### Pattern 1: Environment Variable (Most Common)

**Usage**: 1,648 steps

```yaml
steps:
  - name: Example
    env:
      GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}
    run: gh api --token "$GH_TOKEN" /repos/owner/repo
```

**Flow**: Secrets Store → Context → Env Var → Shell Process

**Locations**:
1. GitHub Secrets Store (encrypted)
2. GitHub Actions context (in-memory)
3. Environment variable (in-memory)
4. Shell process (in-memory)

### Pattern 2: Action Input (GitHub Script)

**Usage**: ~800 steps (estimated)

```yaml
steps:
  - uses: actions/github-script@v8
    with:
      github-token: ${{ secrets.GITHUB_TOKEN }}
      script: |
        // Token automatically authenticated
        const repos = await github.rest.repos.get({...});
```

**Flow**: Secrets Store → Context → Action Input → JavaScript

**Locations**:
1. GitHub Secrets Store (encrypted)
2. GitHub Actions context (in-memory)
3. Action input (in-memory)
4. JavaScript/Node.js process (in-memory)
5. Octokit client (in-memory)

### Pattern 3: Token Cascade (Best Practice)

**Usage**: Majority of workflows

```yaml
with:
  github-token: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
```

**Purpose**: 
- Primary: Highest privilege (MCP operations)
- Fallback 1: Write access (safe outputs)
- Fallback 2: Default read-only

**Benefit**: Works across different deployment environments without workflow changes

### Pattern 4: Redaction (Security)

**Usage**: Before every artifact upload

```yaml
- name: Redact secrets in logs
  env:
    GH_AW_SECRET_NAMES: 'COPILOT_GITHUB_TOKEN,GH_AW_GITHUB_TOKEN,GITHUB_TOKEN'
    SECRET_COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
    SECRET_GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}
    SECRET_GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  with:
    script: |
      const { main } = require('/tmp/gh-aw/actions/redact_secrets.cjs');
      await main();
```

**Process**:
1. List secret names in `GH_AW_SECRET_NAMES`
2. Pass actual values as `SECRET_{NAME}` environment variables
3. Scan all files in `/tmp/gh-aw` (text files only)
4. Replace exact matches with `abc***...` pattern
5. Write back to files

**Files Scanned**: `.txt`, `.json`, `.log`, `.md`, `.mdx`, `.yml`, `.jsonl`

## Job-Level Secret Patterns

### Typical Agent Job (Most Common)

**Steps**: 7-12 steps with secrets

1. **Clone repo-memory** - `github.token` for git auth
2. **Checkout PR** - Enhanced token with fallback
3. **Validate secrets** - Check availability
4. **Install tools** - Authenticated downloads
5. **Setup MCPs** - MCP server configuration
6. **Execute agent** - All credentials provided
7. **Redact secrets** - Clean before upload

**Total Secrets**: 3-5 different secrets typically

### Minimal Job (Conclusion)

**Steps**: 3-5 steps with secrets

1. **Process outputs** - Read safe outputs file
2. **Create/update entities** - Issue, PR, discussion
3. **Update comments** - Status updates

**Total Secrets**: 1-2 (usually `GH_AW_GITHUB_TOKEN` or `GITHUB_TOKEN`)

## Security Analysis

### Attack Surface

| Attack Vector | Risk Level | Mitigations in Place |
|---------------|-----------|---------------------|
| Environment leak | **Medium** | Auto-masking, redaction system |
| Script injection | **High** | Env vars, sanitization |
| Log exposure | **Low** | Auto-masking, structured logging |
| Artifact upload | **High** | Dedicated redaction step |
| PR from fork | **Medium** | Restricted secrets, read-only token |

### Protection Layers

1. **GitHub Built-in**
   - Automatic secret masking in logs
   - Encrypted secrets store
   - Token expiration
   - Scope-based permissions

2. **gh-aw Custom**
   - Pre-upload redaction (`redact_secrets.cjs`)
   - Template injection prevention
   - Environment variable usage
   - Token validation steps

3. **Workflow Design**
   - Separate activation/execution jobs
   - Limited secret scope (step-level)
   - Token cascade with fallbacks
   - Explicit permission blocks

## github.token Special Case

**Type**: Automatic token, not stored in Secrets

**Usage**: 930 references across all workflows

**Primary Uses**:
1. Git authentication (remote URL with token)
2. Environment variable (`GH_TOKEN=${{ github.token }}`)
3. Basic GitHub API calls

**Important Notes**:
- Never used directly in JavaScript files
- JavaScript gets tokens via env vars or action inputs
- Auto-masked in logs
- Permissions controlled by workflow `permissions:` block

**Common Pattern**:
```yaml
- run: |
    git remote set-url origin "https://x-access-token:${{ github.token }}@${SERVER_URL}/${REPO}.git"
```

## Secret Hierarchy

### Privilege Levels

```
Level 1 (Highest): GH_AW_GITHUB_MCP_SERVER_TOKEN
  ├─ Full GitHub API access
  ├─ Repository write
  ├─ MCP server operations
  └─ 736 usages

Level 2 (Write): GH_AW_GITHUB_TOKEN
  ├─ Create/update issues, PRs, comments
  ├─ Push to branches
  ├─ Most API endpoints
  └─ 1,101 usages

Level 3 (Read): GITHUB_TOKEN
  ├─ Read repository
  ├─ Limited write operations
  ├─ Auto-provided
  └─ 1,228 usages
```

### AI Engine Tokens (Separate Hierarchy)

- `COPILOT_GITHUB_TOKEN` - 405 usages
- `ANTHROPIC_API_KEY` - 155 usages
- `CLAUDE_CODE_OAUTH_TOKEN` - 155 usages
- `OPENAI_API_KEY` - 60 usages
- `CODEX_API_KEY` - 60 usages

## Redaction System Deep Dive

### Implementation: `redact_secrets.cjs`

**Location**: `actions/setup/js/redact_secrets.cjs`

**Algorithm**:
1. Parse `GH_AW_SECRET_NAMES` (comma-separated list)
2. For each name, read `SECRET_{NAME}` env var
3. Sort secrets by length (longest first) - handles overlaps
4. Recursively scan `/tmp/gh-aw` directory
5. For each matching file extension:
   - Read content as UTF-8
   - Use string split/join (not regex) for exact matching
   - Replace with first 3 chars + asterisks
   - Write back to file
6. Log redaction count (not values)

**Example**:
```javascript
// Secret: "ghp_1234567890abcdefghijklmnopqrstuvwxyz"
// Redacted: "ghp***********************************"
```

**Limitations**:
- Only scans `/tmp/gh-aw` directory
- Text files only (no binary)
- Minimum length: 8 characters
- Exact string matching (no partial)

**Performance**:
- Typical scan: 50-200 files
- Typical time: 1-3 seconds
- No impact on workflow duration

## Recommendations

### For Developers

1. **Always use token cascade** in workflows
2. **Validate secrets early** with dedicated step
3. **Run redaction before artifacts** without exception
4. **Use environment variables** instead of direct interpolation
5. **Minimize secret scope** to specific steps

### For Security

1. **Audit workflows** that use sensitive secrets quarterly
2. **Monitor redaction logs** for suspicious patterns
3. **Test redaction** with sample secrets
4. **Review token permissions** and right-size
5. **Rotate secrets** on schedule (90 days recommended)

### For Operations

1. **Document all secrets** with purpose and owner
2. **Separate environments** (prod/staging/dev secrets)
3. **Use least privilege** tokens when possible
4. **Monitor secret usage** via GitHub audit log
5. **Have rotation plan** for compromised secrets

## Key Findings

### What Secrets Are Used For

1. **GitHub API Operations** (90% of usage)
   - Create issues, PRs, discussions
   - Add comments, labels, reactions
   - Update project boards
   - Push to branches

2. **AI Engine Authentication** (8% of usage)
   - Copilot CLI
   - Claude API
   - OpenAI/Codex API
   - Custom AI engines

3. **External Integrations** (2% of usage)
   - Notion, Slack, Sentry
   - Search APIs (Tavily, Brave)
   - Cloud providers (Azure)
   - Monitoring (Datadog)

### Where Secrets Live (Process Memory)

**Timeline**:
1. **Workflow start** → Secrets loaded into context (0ms)
2. **Job start** → Job-level env vars set (0-5ms)
3. **Step execution** → Step-level env vars set (0-5ms)
4. **Process spawn** → Secrets copied to child process (0-10ms)
5. **Step complete** → Environment cleared (0ms)
6. **Job complete** → Process memory freed (0ms)

**Key Insight**: Secrets exist in plaintext in process memory only during active execution.

### Where Secrets DO NOT Live

✅ **Never in**:
- Git repository files (checked in code)
- Unencrypted artifacts
- Unmasked logs
- Database storage
- Network transmission (except HTTPS)

⚠️ **Potentially in** (risk area):
- Temporary files under `/tmp/gh-aw` (hence redaction)
- Core dumps (if runner crashes - rare)
- Swap space (if runner low on memory - rare)

## Related Documentation

- **Full Analysis**: `docs/src/content/docs/security/secrets-information-flow.md`
- **Redaction Code**: `actions/setup/js/redact_secrets.cjs`
- **Template Prevention**: `specs/template-injection-prevention.md`
- **GitHub Actions Security**: `specs/github-actions-security-best-practices.md`

## Analysis Methodology

### Data Collection

1. **Static Analysis**
   - Parsed all 125 `.lock.yml` files
   - Extracted secret references via regex patterns
   - Mapped to jobs and steps
   - Counted occurrences per secret type

2. **Code Analysis**
   - Examined JavaScript action files
   - Reviewed shell scripts
   - Analyzed redaction system
   - Documented flow patterns

3. **Pattern Recognition**
   - Identified common usage patterns
   - Documented fallback mechanisms
   - Mapped information flow
   - Categorized security controls

### Tools Used

- Python 3 (YAML parsing, analysis)
- PyYAML (YAML processing)
- Regex (pattern matching)
- Manual code review (flow validation)

---

**Document Version**: 1.0  
**Completed**: 2026-01-06  
**Review Date**: 2026-04-06 (quarterly review recommended)
