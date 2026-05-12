---
name: awf-release-integrator
description: Upgrade gh-aw to latest gh-aw-firewall release and identify follow-up spec tasks.
---

# AWF Release Integrator

Use this skill when updating `github/gh-aw` to a newer `github/gh-aw-firewall` release.

## Goal

Land the version bump cleanly, rebuild the generated artifacts, and review upstream release/spec changes for any follow-up work that should accompany the bump.

## Required sources

Consult these sources before editing anything:

1. The latest `github/gh-aw-firewall` release metadata and body.
2. The current gh-aw version pins in `pkg/constants/version_constants.go`.
3. The canonical AWF config sources spec in `specs/awf-config-sources-spec.md`.
4. The embedded AWF schema in `pkg/workflow/schemas/awf-config.schema.json`.
5. AWF config integration code in:
   - `pkg/workflow/awf_config.go`
   - `pkg/workflow/awf_helpers.go`
   - related AWF tests under `pkg/workflow/`

For upstream spec review, compare these files from the target `github/gh-aw-firewall` release or tag:

- `docs/awf-config-spec.md`
- `docs/awf-config.schema.json`
- `src/awf-config-schema.json`
- any release assets such as `awf-config.schema.json`

## Update procedure

1. Read `pkg/constants/version_constants.go` and record:
   - `DefaultFirewallVersion`
   - every `AWF*MinVersion` constant
2. Look up the latest `github/gh-aw-firewall` release.
3. If the latest release tag matches `DefaultFirewallVersion`, report that no version bump is needed and only continue with spec/release-note review if explicitly requested.
4. If a newer release exists, update the gh-aw pins:
   - bump `DefaultFirewallVersion`
   - update any `AWF*MinVersion` constants that must move because the new release introduces or changes gated flags/features
5. Review release notes for:
   - new flags
   - removed or deprecated flags
   - schema/config additions
   - security fixes
   - behavioral changes that could require new tests, docs, or ADR/spec updates
6. Review the upstream AWF specification and schema changes against:
   - `pkg/workflow/schemas/awf-config.schema.json`
   - `specs/awf-config-sources-spec.md`
   - local AWF config generation and validation code
7. Update any directly related gh-aw files needed for a complete integration, such as:
   - embedded schema copies
   - version-gated helpers/tests
   - specs or ADRs documenting newly surfaced AWF behavior
8. Add or update a patch changeset when the bump changes shipped behavior.

## Required validation

After editing, run the full AWF rebuild flow exactly in this order. The second
`make recompile` is required to refresh image SHA pins resolved during the first pass.

```bash
make build
make recompile
make recompile
```

Then run focused validation for any touched Go code or schema logic, especially AWF-related tests.

## Expected output

Summarize:

- current gh-aw AWF version → target release
- updated constants
- release-note highlights
- specification/schema differences reviewed
- additional recommended follow-up updates that are not yet implemented

## Review heuristics

When deciding whether more than a version bump is needed, specifically check for:

- new AWF schema properties not represented in gh-aw
- new CLI flags that need `AWF*MinVersion` gates
- config fields present in schema but absent from gh-aw generation/validation
- drift that should update `specs/awf-config-sources-spec.md`
- tests whose expected pinned AWF version or schema URLs need refresh
