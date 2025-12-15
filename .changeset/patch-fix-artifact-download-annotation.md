---
"gh-aw": patch
---

Prevent false-positive download annotations by gating the patch download step
behind `needs.agent.outputs.has_patch == 'true'`. When a run uses only
safe-outputs and no code patch is produced, the artifact `aw.patch` will not
exist; the conditional avoids attempting to download it and thus removes
erroneous GitHub Actions error annotations.

Files changed:
- `pkg/workflow/artifacts.go` - added conditional support for artifact steps
- `pkg/workflow/threat_detection.go` - download step now has an `if` check
- `pkg/workflow/threat_detection_test.go` - added test for conditional step

Fixes: false-positive `Unable to download artifact(s): Artifact not found`
issues when no patch is present.

