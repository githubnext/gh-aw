---
title: Smoke Testing Strategy
description: Documentation of the smoke testing strategy for GitHub Agentic Workflows
---

# Smoke Testing Strategy

This document describes the smoke testing strategy for GitHub Agentic Workflows (gh-aw). Smoke tests are automated workflows that validate core functionality and ensure the system is working correctly across different engines and configurations.

## Overview

The smoke testing infrastructure has been consolidated into a focused set of workflows that provide comprehensive coverage while minimizing maintenance burden.

### Test Coverage Matrix

| Engine | Core | Security | Integrations | SRT |
|--------|------|----------|--------------|-----|
| **Copilot** | ✅ | ✅ | ✅ | ✅ |
| **Claude** | ✅ | ⚪ | ⚪ | ⚪ |
| **Codex** | ✅ | ⚪ | ⚪ | ⚪ |

**Legend:**
- ✅ Fully supported
- ⚪ Not currently tested (can be added as needed)

## Workflow Structure

### Core Smoke Tests (`smoke-core.md`)

**Purpose:** Validate essential functionality that should work consistently across all AI engines.

**Schedule:** Every 6 hours

**Engines Tested:** Copilot, Claude, Codex (selectable via workflow_dispatch)

**Test Scenarios:**
- GitHub MCP server integration
- File writing and reading operations
- Bash tool execution
- Cache memory functionality
- Playwright browser automation
- GitHub MCP toolset validation

**Key Features:**
- Strict mode enabled
- Firewall (AWF) enabled by default
- Multi-engine testing support via input parameter
- Comprehensive validation of core features

**Workflow File:** `.github/workflows/smoke-core.md`

### Security Smoke Tests (`smoke-security.md`)

**Purpose:** Validate security-related configuration variants including firewall settings, strict mode, and safe-inputs.

**Schedule:** Twice daily (3am, 3pm UTC)

**Engine:** Copilot

**Test Variants:**
- **Firewall:** Tests with AWF firewall enabled and strict mode
- **No-Firewall:** Tests without firewall to validate broader access
- **Safe-Inputs:** Tests safe-inputs functionality with GitHub MCP disabled

**Test Scenarios:**
- Firewall blocking validation (domains not in allow list should be blocked)
- GitHub MCP access through firewall
- Strict mode constraint enforcement
- Network sandboxing verification
- Safe-inputs as GitHub MCP alternative

**Key Features:**
- Dynamic firewall configuration based on variant
- Validates security constraints are properly enforced
- Tests alternative access methods (safe-inputs)

**Workflow File:** `.github/workflows/smoke-security.md`

### Integration Smoke Tests (`smoke-integrations.md`)

**Purpose:** Validate Playwright, MCP servers, and external tool integrations.

**Schedule:** Twice daily (9am, 9pm UTC)

**Engine:** Copilot (with debug logging enabled)

**Test Integrations:**
- **Playwright:** Browser automation, Docker container health, trace capture
- **MCP Servers:** GitHub MCP communication, toolset availability, tool execution

**Test Scenarios:**
- Playwright browser navigation and page verification
- Docker container startup and health checks
- MCP server communication through Docker
- GitHub CLI integration via safe-inputs
- Trace capture and artifact collection

**Key Features:**
- Pre-flight Docker container tests
- Post-execution log collection
- Artifact upload for debugging
- Chrome services domain access validation

**Workflow File:** `.github/workflows/smoke-integrations.md`

### Specialized Tests

#### Sandbox Runtime (SRT) Tests

**Purpose:** Validate Sandbox Runtime (SRT) integration with custom configurations.

**Workflows:**
- `smoke-srt.md` - Basic SRT validation
- `smoke-srt-custom-config.md` - Custom SRT configuration testing
- `smoke-isolated-srt.yml` - Isolated SRT environment testing

**Why Separate:** SRT requires specialized configuration and setup that differs significantly from other test scenarios. These workflows test experimental sandboxing features.

**Workflow Files:** 
- `.github/workflows/smoke-srt.md`
- `.github/workflows/smoke-srt-custom-config.md`
- `.github/workflows/smoke-isolated-srt.yml`

#### Smoke Detector (`smoke-detector.md`)

**Purpose:** Reusable workflow that investigates and diagnoses failed smoke tests.

**Type:** `workflow_call` (reusable workflow)

**Function:** When a smoke test fails, this workflow:
- Analyzes failure logs using `gh-aw_audit` and `gh-aw_logs` tools
- Identifies root causes and patterns
- Searches for similar historical failures
- Creates investigation reports
- Posts findings to associated PR or creates an issue

**Why Separate:** This is a meta-workflow that operates on other workflows, not a test itself.

**Workflow File:** `.github/workflows/smoke-detector.md`

## Test Schedule

| Workflow | Schedule | Frequency |
|----------|----------|-----------|
| smoke-core | `0 0,6,12,18 * * *` | Every 6 hours |
| smoke-security | `0 3,15 * * *` | Twice daily |
| smoke-integrations | `0 9,21 * * *` | Twice daily |
| smoke-srt | On-demand | workflow_dispatch |
| smoke-srt-custom-config | On-demand | workflow_dispatch |

## Triggering Smoke Tests

### Scheduled Execution

Smoke tests run automatically on their configured schedules.

### Manual Execution

All smoke tests support manual triggering via `workflow_dispatch`:

```bash
# Trigger via GitHub CLI
gh workflow run smoke-core.md
gh workflow run smoke-security.md -f variant=firewall
gh workflow run smoke-integrations.md -f integration=playwright
```

### Pull Request Labels

Smoke tests can be triggered on pull requests by adding the `smoke` label:

```bash
# Label a PR to trigger smoke tests
gh pr edit <PR_NUMBER> --add-label smoke
```

## Adding New Test Scenarios

### To Core Tests

Edit `.github/workflows/smoke-core.md` and add new test requirements in the markdown body. The test should be:
- Engine-agnostic (works on Copilot, Claude, and Codex)
- Essential functionality (not optional features)
- Quick to execute (< 2 minutes)

### To Security Tests

Edit `.github/workflows/smoke-security.md` and add new security variant tests. Consider:
- Security implications
- Firewall/sandbox configuration requirements
- Strict mode compatibility

### To Integration Tests

Edit `.github/workflows/smoke-integrations.md` and add new integration tests. Ensure:
- External dependencies are documented
- Pre-flight checks are included
- Post-execution cleanup is handled

### New Workflow Category

If a new category of tests is needed:
1. Create a new `smoke-[category].md` file
2. Follow the existing workflow structure
3. Document the purpose and schedule
4. Update this strategy document

## Debugging Failed Smoke Tests

### Automated Investigation

When a smoke test fails, the `smoke-detector.md` workflow automatically:
1. Downloads and analyzes logs
2. Identifies error patterns
3. Searches for similar past failures
4. Posts investigation results

### Manual Investigation

To manually investigate a failed smoke test:

```bash
# Audit a specific workflow run
gh-aw audit <RUN_ID>

# Download and analyze logs
gh-aw logs <RUN_ID>

# List recent workflow runs
gh run list --workflow=smoke-core.md --limit 10
```

### Common Failure Patterns

| Pattern | Likely Cause | Resolution |
|---------|--------------|------------|
| GitHub MCP timeout | API rate limiting | Wait for rate limit reset |
| Playwright container failed | Docker image unavailable | Check Docker registry status |
| File write permission denied | Filesystem permissions | Check sandbox configuration |
| Firewall blocking expected domain | Network configuration | Verify allowed domains list |

## Maintenance

### Updating Smoke Tests

1. Edit the markdown workflow file (`.md`)
2. Test changes locally if possible
3. Commit and push changes
4. The compiled `.lock.yml` files will be regenerated automatically

### Recompiling Workflows

After making changes to smoke test workflows:

```bash
make recompile
```

This ensures the `.lock.yml` files are up-to-date with your changes.

### Monitoring Test Health

- Review GitHub Actions workflow runs regularly
- Check for patterns in failures
- Review smoke-detector investigation reports
- Update tests when new features are added

## Archived Workflows

Previous smoke test workflows have been consolidated. The old workflows are:

- `smoke-copilot.md` → Consolidated into `smoke-core.md`
- `smoke-claude.md` → Consolidated into `smoke-core.md`
- `smoke-codex.md` → Consolidated into `smoke-core.md`
- `smoke-copilot-no-firewall.md` → Consolidated into `smoke-security.md`
- `smoke-copilot-playwright.md` → Consolidated into `smoke-integrations.md`
- `smoke-copilot-safe-inputs.md` → Consolidated into `smoke-security.md`
- `smoke-codex-firewall.md` → Consolidated into `smoke-security.md`

These workflows remain available in `.github/workflows/archived/` for reference.

## Best Practices

### Writing Smoke Tests

- **Keep tests concise:** Smoke tests should complete quickly (< 10 minutes)
- **Focus on critical paths:** Test essential functionality, not edge cases
- **Use brief output:** Follow the "extremely short and concise" guideline
- **Include cleanup:** Ensure tests don't leave artifacts that affect subsequent runs
- **Document expectations:** Clearly specify what "passing" means

### Labeling and Reporting

- Use consistent label names (`smoke-core`, `smoke-security`, etc.)
- Report results in a structured table format
- Include run ID for traceability
- Keep PR comments concise (max 10-15 lines)

### Security Considerations

- Always test with firewall enabled in at least one variant
- Validate that blocked domains are actually blocked
- Ensure strict mode constraints are enforced
- Never expose secrets in test output

## Future Enhancements

Potential improvements to the smoke testing infrastructure:

- Add performance benchmarking to smoke tests
- Implement smoke test result trends/dashboards
- Add smoke tests for new engines as they're added
- Expand Claude and Codex coverage to security/integration tests
- Create smoke tests for new features automatically

## Related Documentation

- [GitHub Actions Workflows](../guides/github-actions)
- [MCP Server Configuration](../reference/mcp-servers)
- [Troubleshooting Guide](../troubleshooting)
- [CI/CD Documentation](../guides/cicd)
