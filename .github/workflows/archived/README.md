# Archived Smoke Test Workflows

This directory contains smoke test workflows that have been consolidated into a unified test matrix.

## Consolidation Overview

To reduce maintenance burden and improve test clarity, the original 10 smoke test workflows have been consolidated into 5 focused workflows:

### New Consolidated Workflows

1. **smoke-core.md** - Core functionality testing across all engines (Copilot, Claude, Codex)
2. **smoke-security.md** - Security variant testing (firewall, no-firewall, safe-inputs)
3. **smoke-integrations.md** - Integration testing (Playwright, MCP servers)
4. **smoke-srt.md** - Sandbox Runtime (SRT) testing (kept separate - specialized)
5. **smoke-detector.md** - Failure investigation workflow (kept separate - meta-workflow)

### Archived Workflows

The following workflows have been archived and their functionality moved to the consolidated workflows:

| Archived Workflow | Consolidated Into | Reason |
|-------------------|-------------------|---------|
| `smoke-copilot.md` | `smoke-core.md` | Core engine testing |
| `smoke-claude.md` | `smoke-core.md` | Core engine testing |
| `smoke-codex.md` | `smoke-core.md` | Core engine testing |
| `smoke-copilot-no-firewall.md` | `smoke-security.md` | Security variant (no firewall) |
| `smoke-copilot-playwright.md` | `smoke-integrations.md` | Playwright integration |
| `smoke-copilot-safe-inputs.md` | `smoke-security.md` | Security variant (safe-inputs) |
| `smoke-codex-firewall.md` | `smoke-security.md` | Security variant (firewall) |

## Benefits of Consolidation

1. **Reduced Maintenance:** 7 workflows → 3 workflows (excluding specialized SRT and detector workflows)
2. **Clear Test Strategy:** Organized by purpose (core, security, integrations)
3. **Better Coverage Visibility:** Test matrix documented in one place
4. **Easier to Extend:** Add new engines/variants in one location
5. **Consistent Patterns:** Same test structure across all engines

## Migration Path

The archived workflows are preserved here for reference. Their `.lock.yml` files remain in the main workflows directory temporarily to avoid breaking existing integrations, but they are no longer actively maintained.

### If You Need to Reference Old Tests

The archived workflows contain valuable test patterns and configurations. When adding new tests, you can:

1. Review the archived workflow for the test scenario
2. Extract the relevant test requirements
3. Add them to the appropriate consolidated workflow

## Documentation

See `docs/src/content/docs/smoke-testing-strategy.md` for complete documentation on:
- Current smoke testing strategy
- Test coverage matrix
- How to add new test scenarios
- Debugging failed smoke tests
- Best practices

## Date of Consolidation

Consolidated: December 2024

## Related Issues

See issue: "Consolidate Smoke Test Workflows into Unified Test Matrix"
