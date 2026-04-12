---
safe-outputs:
  upload-artifact:
    max-uploads: 3
    retention-days: 30
    skip-archive: true
---

<!--
# Shared Upload Artifact Safe Output Configuration

This shared workflow enables the `upload_artifact` safe output tool, which lets AI agents
upload files as run-scoped GitHub Actions artifacts.

## How it works

The agent stages files to `$RUNNER_TEMP/gh-aw/safeoutputs/upload-artifacts/` and calls the
`upload_artifact` tool. The `safe_outputs` job picks up the staged files and uploads them
directly via the `@actions/artifact` REST API (no `actions: write` permission needed —
authentication uses `ACTIONS_RUNTIME_TOKEN` which is always available to the runner).

The tool returns a temporary opaque artifact ID (`aw_*`) that can be resolved to
a download URL by an authorised downstream step.

## Usage

Import this shared workflow to enable `upload_artifact` in any workflow:

```yaml
imports:
  - shared/safe-output-upload-artifact.md
```

The agent must stage files before calling the tool:

```bash
# Stage files to the upload-artifacts directory
cp dist/report.json "$RUNNER_TEMP/gh-aw/safeoutputs/upload-artifacts/report.json"
```

Then call the tool:

```json
{ "type": "upload_artifact", "path": "report.json" }
```

## Configuration defaults

- `max-uploads`: 3 uploads per run
- `retention-days`: 30 days (fixed; the agent cannot override this value)
- `skip-archive`: true (single-file uploads skip zip archiving; fixed)

Override any of these by defining `upload-artifact` directly in your workflow's
`safe-outputs` section (the top-level definition takes precedence over the import).
-->
