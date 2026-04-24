---
mcp-servers:
  kreuzberg:
    container: "ghcr.io/kreuzberg-dev/kreuzberg"
    version: "latest"
    entrypointArgs:
      - "mcp"
    mounts:
      - ${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}:ro
    allowed:
      # Document extraction tools (read-only)
      - "extract_file"
      - "extract_bytes"
      - "batch_extract_files"
      # Format discovery tools (read-only)
      - "detect_mime_type"
      - "list_formats"
      - "get_version"
      # Text processing tools (read-only)
      - "embed_text"
      - "chunk_text"
      # Cache inspection tools (read-only)
      - "cache_stats"
      - "cache_manifest"
      # Excluded write/mutating operations:
      # - "cache_clear"   # Evicts all cached results
      # - "cache_warm"    # Pre-downloads embedding models
      # Excluded feature-flag-gated operations:
      # - "extract_structured"  # Requires liter-llm feature flag at build time
---
<!--
## Kreuzberg MCP Server

Kreuzberg is a polyglot document intelligence engine. The MCP server exposes its
full extraction engine as 13 discoverable tools, communicating over stdin/stdout
with JSON-RPC 2.0. It supports 97+ file formats including PDF, DOCX, PPTX,
images (with Tesseract OCR), and legacy Office formats (with LibreOffice in the
full image).

Documentation: https://docs.kreuzberg.dev/guides/docker/
MCP integration guide: https://docs.kreuzberg.dev/guides/mcp-integration/
GitHub: https://github.com/kreuzberg-dev/kreuzberg

### Container images

Two images are available (both on `ghcr.io/kreuzberg-dev/kreuzberg`):
- **Core** (~1.0–1.3 GB): Modern formats, Tesseract OCR (12 languages)
- **Full** (~1.5–2.1 GB): Adds LibreOffice for legacy `.doc`/`.ppt` files
  Use tag `full` or `latest-full` to select the full image.

### Required secrets

None — no API token is required.

### Available tools (read-only)

| Tool | Params | Description |
|---|---|---|
| `extract_file` | `path` | Extract text and metadata from a local file |
| `extract_bytes` | `data` (base64) | Extract from base64-encoded file content |
| `batch_extract_files` | `paths` | Extract multiple files in one call |
| `detect_mime_type` | `path` | Identify a file's MIME type |
| `list_formats` | — | List all supported file formats |
| `get_version` | — | Return the library version string |
| `embed_text` | `texts` | Generate embedding vectors for text chunks |
| `chunk_text` | `text` | Split text into overlapping chunks |
| `cache_stats` | — | Report how much content is cached |
| `cache_manifest` | — | Return model checksums |

### Excluded tools

- `cache_clear` — Evicts all cached results (write operation)
- `cache_warm` — Pre-downloads embedding models (write operation)
- `extract_structured` — Requires the `liter-llm` build-time feature flag

### Workspace access

The workspace is mounted read-only at the same path it exists on the host,
so `extract_file` and `batch_extract_files` can reference files using their
absolute workspace paths (e.g. `${{ github.workspace }}/document.pdf`).

### Usage in workflows

```yaml
imports:
  - shared/mcp/kreuzberg.md
```
-->
