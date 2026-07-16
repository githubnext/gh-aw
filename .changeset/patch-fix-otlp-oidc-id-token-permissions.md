---
"gh-aw": patch
---

Fix missing `id-token: write` permission on activation, conclusion, and safe_outputs jobs when using OTLP OIDC audience-only authentication (`observability.otlp.github-app` with only an `audience`, no `app-id`/`private-key`). These jobs receive the OTLP OIDC mint step (`core.getIDToken`) at compile time but previously lacked the permission required to call the GitHub OIDC token API, causing `Unable to get ACTIONS_ID_TOKEN_REQUEST_URL env variable` at runtime.

Also adds `audience` as a valid field in the `observability.otlp.github-app` JSON schema so that credential-less OIDC mode can be configured without schema validation errors.
