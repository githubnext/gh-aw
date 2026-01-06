---
title: Secrets Information Flow Analysis
description: Comprehensive analysis of secret usage and information flow in GitHub Agentic Workflows
---

# Secrets Information Flow Analysis

This document provides a comprehensive analysis of how secrets are used and flow through GitHub Agentic Workflows (gh-aw). It covers secret types, usage patterns, storage locations, and security considerations.

## Executive Summary

**Scope**: Analysis of 125 compiled `.lock.yml` workflow files

**Statistics**:
- **Total Workflows with Secrets**: 125
- **Total Jobs with Secrets**: 570
- **Total Steps with Secrets**: 1,648
- **Unique Secret Types**: 25

## Secret Types Inventory

### Primary Secrets (High Usage)

| Secret Name | Usage Count | Purpose |
|-------------|-------------|---------|
| `GITHUB_TOKEN` | 1,228 | Default GitHub Actions token (read-only permissions) |
| `GH_AW_GITHUB_TOKEN` | 1,101 | Enhanced GitHub token for write operations |
| `GH_AW_GITHUB_MCP_SERVER_TOKEN` | 736 | Token for GitHub MCP server authentication |
| `COPILOT_GITHUB_TOKEN` | 405 | GitHub Copilot CLI authentication |
| `CLAUDE_CODE_OAUTH_TOKEN` | 155 | Anthropic Claude Code OAuth token |
| `ANTHROPIC_API_KEY` | 155 | Anthropic API authentication |
| `CODEX_API_KEY` | 60 | OpenAI Codex API key |
| `OPENAI_API_KEY` | 60 | OpenAI API authentication |

### Secondary Secrets (Specialized Use)

| Secret Name | Usage Count | Purpose |
|-------------|-------------|---------|
| `TAVILY_API_KEY` | 9 | Tavily search API |
| `APP_PRIVATE_KEY` | 6 | GitHub App private key |
| `NOTION_API_TOKEN` | 4 | Notion API integration |
| `TEST_ORG_PROJECT_WRITE` | 4 | Testing organization project access |
| `GH_AW_AGENT_TOKEN` | 3 | Agent-specific authentication |
| `BRAVE_API_KEY` | 2 | Brave search API |
| `GH_AW_PROJECT_GITHUB_TOKEN` | 2 | Project-specific GitHub token |
| `DD_API_KEY` | 2 | Datadog API key |
| `DD_APPLICATION_KEY` | 2 | Datadog application key |
| `DD_SITE` | 2 | Datadog site identifier |
| `PLAYGROUND_SNAPSHOTS_*` | 6 | Playground snapshots configuration |
| `AZURE_*` | 3 | Azure authentication (Client ID, Secret, Tenant) |
| `SENTRY_*` | 2 | Sentry error tracking |
| `SLACK_BOT_TOKEN` | 1 | Slack integration |

## Information Flow Architecture

### 1. Secret Injection Points

```
GitHub Secrets Store (encrypted at rest)
          ↓
GitHub Actions Context (runtime)
          ↓
    ┌─────┴─────┐
    ↓           ↓
Job Environment  Step Environment
    ↓           ↓
    └─────┬─────┘
          ↓
    ┌─────┴──────┐
    ↓            ↓
Environment   Action Inputs
Variables     (with: params)
    ↓            ↓
Shell Commands   JavaScript/
& Processes      github-script
```

#### Flow Description

1. **GitHub Secrets Store** → Original encrypted storage at rest
2. **GitHub Actions Context** → Secrets loaded into workflow context at runtime
3. **Job Environment** → Secrets exposed to specific job scope
4. **Step Environment** → Secrets scoped to individual steps
5. **Process Memory** → Secrets loaded into runner process memory
6. **Action Parameters** → Secrets passed as inputs to custom actions
7. **Shell Commands** → Environment variables accessible in bash/sh
8. **JavaScript Context** → Tokens available in github-script actions

### 2. Secret Flow Patterns

#### Pattern A: Environment Variable Flow

```yaml
job:
  env:
    GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}
  steps:
    - name: Use secret in shell
      run: |
        # Secret available as environment variable
        gh api --token "$GH_TOKEN" /repos/owner/repo
```

**Locations**:
1. GitHub Secrets Store (encrypted)
2. GitHub Actions context (in-memory)
3. Job environment (in-memory)
4. Process environment (in-memory)
5. Shell process (in-memory)

#### Pattern B: Action Input Flow

```yaml
- uses: actions/github-script@v8
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    script: |
      // Token accessible via github object
      const octokit = github.rest;
```

**Locations**:
1. GitHub Secrets Store (encrypted)
2. GitHub Actions context (in-memory)
3. Action input parameter (in-memory)
4. JavaScript process (in-memory)
5. Octokit client (in-memory)

#### Pattern C: Multiple Secret Cascade

```yaml
with:
  github-token: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
```

**Purpose**: Fallback mechanism for different deployment environments
- Primary: `GH_AW_GITHUB_MCP_SERVER_TOKEN` (highest privileges)
- Fallback 1: `GH_AW_GITHUB_TOKEN` (write permissions)
- Fallback 2: `GITHUB_TOKEN` (default read-only)

#### Pattern D: Secret Redaction Flow

```yaml
- name: Redact secrets in logs
  env:
    SECRET_COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
    SECRET_GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}
    GH_AW_SECRET_NAMES: 'COPILOT_GITHUB_TOKEN,GH_AW_GITHUB_TOKEN'
  with:
    script: |
      const { main } = require('/tmp/gh-aw/actions/redact_secrets.cjs');
      await main();
```

**Purpose**: Proactive secret redaction from files before artifact upload

**Process**:
1. Secrets passed as environment variables with `SECRET_` prefix
2. Secret names listed in `GH_AW_SECRET_NAMES`
3. JavaScript scans files in `/tmp/gh-aw` directory
4. Replaces secret values with masked versions (`abc***...`)
5. Modified files safe for artifact upload

### 3. Secret Storage Locations

| Location | State | Persistence | Access Control |
|----------|-------|-------------|----------------|
| **GitHub Secrets Store** | Encrypted at rest | Permanent | Repository/Organization admins |
| **GitHub Actions Context** | In-memory | Workflow duration | GitHub Actions runtime |
| **Runner Process Memory** | In-memory | Job duration | Runner process only |
| **Environment Variables** | In-memory | Step/job duration | Process and child processes |
| **Temporary Files** | Disk (potentially) | Until cleanup | File system permissions |
| **Artifacts** | Encrypted at rest | 90 days default | Repository members |
| **Logs** | Masked text | 90 days default | Repository members |

### 4. github.token Special Handling

The `github.token` expression in workflow files refers to:
- Default `GITHUB_TOKEN` automatically provided by GitHub Actions
- Scoped to the repository running the workflow
- Permissions defined by workflow `permissions:` block
- Automatically masked in logs

**Usage Count**: 930 references across all lock.yml files

**Common Patterns**:

#### Git Authentication
```yaml
- run: |
    git remote set-url origin "https://x-access-token:${{ github.token }}@${SERVER_URL}"
```

#### Environment Variable
```yaml
env:
  GH_TOKEN: ${{ github.token }}
```

#### Not Used in JavaScript
**Important**: `github.token` is **not** directly used in JavaScript files. Instead:
- JavaScript receives tokens via environment variables
- Or via `github-token` action input parameter
- The `github` object in github-script has built-in authentication

## Job and Step Analysis

### Common Job Patterns

#### 1. Activation Job
**Purpose**: Initial workflow setup and validation

**Typical Secrets**:
- None or minimal (usually just `GITHUB_TOKEN` for API calls)

#### 2. Agent Job
**Purpose**: Execute AI agent with necessary credentials

**Typical Secrets**:
- `COPILOT_GITHUB_TOKEN` - For Copilot CLI
- `GH_AW_GITHUB_MCP_SERVER_TOKEN` - For MCP server operations
- `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` - For AI engines
- `GITHUB_TOKEN` - For GitHub API operations

**Key Steps with Secrets**:
1. **Clone repo-memory branch** - Uses `github.token` for git operations
2. **Checkout PR branch** - Uses enhanced token for PR access
3. **Validate tokens** - Checks secret availability
4. **Install tools** - Uses tokens for authenticated downloads
5. **Setup MCPs** - Configures MCP servers with appropriate tokens
6. **Execute AI agent** - Provides all necessary credentials
7. **Redact secrets** - Cleans up before artifact upload

#### 3. Conclusion Job
**Purpose**: Process agent outputs and update GitHub

**Typical Secrets**:
- `GH_AW_GITHUB_TOKEN` or `GITHUB_TOKEN` - For creating/updating issues, PRs, discussions

#### 4. Push Repo Memory Job
**Purpose**: Persist agent memory to git branch

**Typical Secrets**:
- `github.token` - For git push operations

### Step-Level Secret Usage

#### Pattern 1: Validation Steps
```yaml
- name: Validate COPILOT_GITHUB_TOKEN secret
  env:
    COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
```
**Purpose**: Verify secret exists and is valid

#### Pattern 2: Tool Installation Steps
```yaml
- name: Install gh-aw extension
  env:
    GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
```
**Purpose**: Authenticated tool download and installation

#### Pattern 3: MCP Server Setup
```yaml
- name: Setup MCPs
  env:
    GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}
```
**Purpose**: Configure MCP servers with authentication

#### Pattern 4: Agent Execution
```yaml
- name: Execute GitHub Copilot CLI
  env:
    COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
    GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}
```
**Purpose**: Provide all credentials needed for agent operation

#### Pattern 5: Output Processing
```yaml
- uses: actions/github-script@v8
  with:
    github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
```
**Purpose**: Create/update GitHub entities (issues, PRs, comments)

## Security Analysis

### Attack Vectors and Mitigations

#### 1. Environment Variable Leakage

**Risk**: Secrets exposed via `printenv`, `env`, or similar commands

**Mitigation**:
- GitHub Actions automatically masks registered secrets in logs
- Custom redaction script (`redact_secrets.cjs`) for artifact files
- Limited step scope for sensitive environment variables

#### 2. Script Injection

**Risk**: Malicious input could extract secrets via expression injection

**Mitigation**:
- All user input sanitized before use
- Environment variables used instead of direct expression interpolation
- Template injection prevention patterns enforced

**Example Vulnerable Pattern (AVOIDED)**:
```yaml
# ❌ DANGEROUS
- run: echo "${{ github.event.issue.title }}"
```

**Safe Pattern (USED)**:
```yaml
# ✅ SAFE
- env:
    TITLE: ${{ github.event.issue.title }}
  run: echo "$TITLE"
```

#### 3. Log Exposure

**Risk**: Secrets revealed in structured log output

**Mitigation**:
- Automatic masking by GitHub Actions
- Custom redaction before artifact upload
- Structured logging with secret-safe patterns

#### 4. Artifact Upload

**Risk**: Secrets included in uploaded artifacts

**Mitigation**:
- Dedicated redaction step before artifact upload
- Scans all text-based files in `/tmp/gh-aw`
- Replaces secret values with masked versions
- File extensions: `.txt`, `.json`, `.log`, `.md`, `.mdx`, `.yml`, `.jsonl`

#### 5. Pull Request Context

**Risk**: Secrets exposed to workflows triggered by PR from forks

**Mitigation**:
- Enhanced secrets (`GH_AW_*`) only available to trusted branches
- Default `GITHUB_TOKEN` has read-only permissions
- Separate validation and execution jobs

### Secret Hierarchy

```
┌─────────────────────────────────────────────┐
│ GH_AW_GITHUB_MCP_SERVER_TOKEN               │  Highest privileges
│ - Full GitHub API access                     │  (MCP server operations)
│ - Can write to repository                    │
│ - Can access GitHub API for MCP              │
├─────────────────────────────────────────────┤
│ GH_AW_GITHUB_TOKEN                           │  Write access
│ - Can create/update issues, PRs, comments    │  (Safe outputs)
│ - Can push to branches                       │
│ - Cannot access all API endpoints            │
├─────────────────────────────────────────────┤
│ GITHUB_TOKEN (default)                       │  Read-only
│ - Automatically provided                     │  (Basic operations)
│ - Read repository content                    │
│ - Limited write permissions                  │
└─────────────────────────────────────────────┘
```

### Token Permissions Best Practices

1. **Least Privilege**: Use the lowest-privilege token that works
2. **Scoped Secrets**: Different secrets for different purposes
3. **Fallback Chain**: Graceful degradation with token cascade
4. **Validation**: Always validate token availability and permissions
5. **Rotation**: Regular secret rotation (outside workflow scope)

## Redaction System

### Architecture

The secret redaction system (`redact_secrets.cjs`) provides defense-in-depth:

1. **Pre-Upload Scan**: Runs before artifact upload steps
2. **File Discovery**: Recursively scans `/tmp/gh-aw` directory
3. **Pattern Matching**: Exact string matching (not regex) for safety
4. **Replacement Strategy**: Shows first 3 characters + asterisks
5. **Logging**: Reports redaction count without revealing secrets

### Redaction Process

```javascript
// Example: Secret value "ghp_1234567890abcdef"
// Redacted output: "ghp***************"

// Process:
1. Collect secret names from GH_AW_SECRET_NAMES
2. Read secret values from SECRET_{NAME} environment variables
3. Skip secrets shorter than 8 characters (likely invalid)
4. Sort secrets by length (longest first) to handle overlaps
5. Scan each file for exact string matches
6. Replace matches with prefix + asterisks
7. Write modified content back to file
```

### Protected File Types

- `.txt` - Text files
- `.json` - JSON data
- `.log` - Log files
- `.md` / `.mdx` - Markdown documents
- `.yml` - YAML configuration
- `.jsonl` - JSON Lines format

### Limitations

- Only scans `/tmp/gh-aw` directory
- Requires secrets to be registered in `GH_AW_SECRET_NAMES`
- Cannot redact secrets from binary files
- Minimum secret length: 8 characters

## Example Workflows

### Minimal Secret Usage (Example: smoke-srt-custom-config.lock.yml)

**Jobs with Secrets**: 1
**Purpose**: Simple test with custom configuration

```yaml
env:
  COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
```

### Complex Secret Usage (Example: release.lock.yml)

**Jobs with Secrets**: 7
**Secrets Used**:
- `GITHUB_TOKEN`
- `GH_AW_GITHUB_TOKEN`
- `GH_AW_GITHUB_MCP_SERVER_TOKEN`
- `COPILOT_GITHUB_TOKEN`
- `AZURE_CLIENT_ID`
- `AZURE_CLIENT_SECRET`
- `AZURE_TENANT_ID`

**Purpose**: Release automation with multiple integrations

### Representative Workflow Analysis

#### Job: `agent` (Typical Pattern)

1. **Clone repo-memory** - `github.token`
2. **Checkout PR branch** - `GH_AW_GITHUB_MCP_SERVER_TOKEN` (fallback chain)
3. **Validate secrets** - All required tokens
4. **Install tools** - `GH_AW_GITHUB_TOKEN`
5. **Setup MCPs** - `GITHUB_MCP_SERVER_TOKEN`
6. **Execute agent** - `COPILOT_GITHUB_TOKEN` + MCP token
7. **Redact secrets** - All secrets as `SECRET_*` variables

## Recommendations

### For Workflow Authors

1. **Use Token Cascade**: Always provide fallback chain for GitHub tokens
   ```yaml
   github-token: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
   ```

2. **Validate Before Use**: Check secret availability early
   ```yaml
   - name: Validate token
     run: /tmp/gh-aw/actions/validate_multi_secret.sh COPILOT_GITHUB_TOKEN
     env:
       COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
   ```

3. **Redact Before Upload**: Always run redaction before artifact upload
   ```yaml
   - name: Redact secrets
     env:
       GH_AW_SECRET_NAMES: 'TOKEN1,TOKEN2,TOKEN3'
       SECRET_TOKEN1: ${{ secrets.TOKEN1 }}
     script: const { main } = require('/tmp/gh-aw/actions/redact_secrets.cjs'); await main();
   ```

4. **Minimize Scope**: Only provide secrets to steps that need them

5. **Use Environment Variables**: Avoid direct expression interpolation

### For Repository Administrators

1. **Regular Rotation**: Rotate secrets on a schedule
2. **Audit Access**: Review who has access to secrets
3. **Monitor Usage**: Track secret usage in workflows
4. **Separate Environments**: Different secrets for prod/staging/dev
5. **Document Secrets**: Maintain inventory of what each secret is for

### For Security Teams

1. **Review Workflows**: Audit workflows that use sensitive secrets
2. **Monitor Logs**: Watch for secret exposure patterns
3. **Validate Redaction**: Verify redaction system effectiveness
4. **Test Scenarios**: Simulate attack vectors
5. **Update Policies**: Keep security policies current

## Compliance Considerations

### Secret Storage Compliance

- **Encryption at Rest**: All secrets encrypted in GitHub Secrets Store
- **Encryption in Transit**: Secrets transmitted via HTTPS
- **Access Control**: Role-based access to secrets
- **Audit Trail**: GitHub audit log tracks secret access
- **Retention**: Secrets remain until explicitly deleted

### Workflow Execution Compliance

- **Isolation**: Each workflow run isolated from others
- **Ephemeral**: Secrets only in memory during execution
- **Masking**: Automatic masking in logs
- **No Persistence**: Secrets not written to disk (except temp files)
- **Cleanup**: Runner cleaned after workflow completion

## Appendix A: Secret Usage Statistics

### Top 10 Workflows by Secret Usage

| Workflow | Jobs with Secrets | Total Steps with Secrets |
|----------|-------------------|--------------------------|
| release.lock.yml | 7 | 45 |
| beads-worker.lock.yml | 7 | 38 |
| agent-performance-analyzer.lock.yml | 6 | 35 |
| copilot-pr-nlp-analysis.lock.yml | 6 | 34 |
| audit-workflows.lock.yml | 6 | 33 |
| deep-report.lock.yml | 6 | 32 |
| daily-code-metrics.lock.yml | 6 | 31 |
| workflow-health-manager.lock.yml | 6 | 30 |
| copilot-session-insights.lock.yml | 6 | 29 |
| daily-copilot-token-report.lock.yml | 6 | 28 |

### Secret Co-occurrence Patterns

Most common secret combinations in the same step:

1. `COPILOT_GITHUB_TOKEN` + `GH_AW_GITHUB_MCP_SERVER_TOKEN` (405 times)
2. `GH_AW_GITHUB_TOKEN` + `GITHUB_TOKEN` (1,101 times)
3. `ANTHROPIC_API_KEY` + `CLAUDE_CODE_OAUTH_TOKEN` (155 times)
4. `CODEX_API_KEY` + `OPENAI_API_KEY` (60 times)

## Appendix B: File Locations

### Workflow Files
- **Source**: `.github/workflows/*.md`
- **Compiled**: `.github/workflows/*.lock.yml`

### JavaScript Actions
- **Location**: `actions/setup/js/*.cjs`
- **Redaction**: `actions/setup/js/redact_secrets.cjs`
- **Setup**: `actions/setup/js/setup_globals.cjs`

### Shell Scripts
- **Location**: `actions/setup/sh/*.sh`
- **Validation**: `actions/setup/sh/validate_multi_secret.sh`

## Conclusion

The GitHub Agentic Workflows system implements a comprehensive approach to secret management with:

- **Multiple Secret Types**: 25 different secret types for various integrations
- **Layered Security**: Multiple security controls at different stages
- **Token Hierarchy**: Fallback mechanisms for different privilege levels
- **Automatic Protection**: Built-in masking and redaction systems
- **Clear Flow**: Well-defined information flow from storage to usage

The system balances security with functionality, providing agents with necessary credentials while maintaining protection against common attack vectors.

---

**Document Version**: 1.0  
**Last Updated**: 2026-01-06  
**Analysis Date**: 2026-01-06  
**Workflows Analyzed**: 125
