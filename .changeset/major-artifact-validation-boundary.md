---
"gh-aw": major
---

Add an isolated artifact-validation boundary for repository publication.

Repository-publishing workflows can declare required policy checks in `safe-outputs.artifact-validation.steps`. Validation runs without publication credentials, and the publisher consumes only the approved artifact ID after the checks succeed.

Custom `safe-outputs.steps` are no longer accepted with repository-publishing handlers. Move artifact validators to `safe-outputs.artifact-validation.steps`. Unsupported protected checkout or publisher customizations fail explicitly instead of weakening the validation boundary.
