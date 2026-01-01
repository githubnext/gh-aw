---
"gh-aw": patch
---

Normalize artifact names to comply with upload-artifact@v5 and fix download path resolution.

Artifact names no longer include file extensions and use consistent delimiters (e.g., `prompt.txt` → `prompt`, `safe_output.jsonl` → `safe-output`). Updated download path logic accounts for `actions/download-artifact` extracting into `{download-path}/{artifact-name}/` subdirectories. Backward-compatible flattening preserves CLI behavior for older runs.

