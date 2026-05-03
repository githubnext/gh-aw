---
name: Daily Model Inventory Checker
description: Queries model lists from OpenAI, Anthropic, Google, and Copilot APIs daily, then analyzes the combined inventory to propose updates to the builtin model alias mapping
on:
  schedule:
    - cron: daily
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

tracker-id: daily-model-inventory
engine: copilot
strict: true
timeout-minutes: 30

jobs:
  collect_openai_models:
    runs-on: ubuntu-latest
    needs: [activation]
    permissions:
      contents: read
    steps:
      - name: Fetch OpenAI models
        id: fetch
        shell: bash
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          set -euo pipefail
          OUT="/tmp/gh-aw/model-inventory/openai"
          mkdir -p "$OUT"
          if [ -z "${OPENAI_API_KEY:-}" ]; then
            echo '{"provider":"openai","error":"OPENAI_API_KEY not set","models":[]}' > "$OUT/models.json"
            echo "status=skipped" >> "$GITHUB_OUTPUT"
            exit 0
          fi
          HTTP_STATUS=$(curl -sf -o "$OUT/raw.json" -w "%{http_code}" \
            -H "Authorization: Bearer $OPENAI_API_KEY" \
            https://api.openai.com/v1/models) || true
          if [ "${HTTP_STATUS:-0}" = "200" ]; then
            jq '{provider:"openai",models:[.data[].id]|sort}' "$OUT/raw.json" > "$OUT/models.json"
            echo "status=ok" >> "$GITHUB_OUTPUT"
          else
            echo "{\"provider\":\"openai\",\"error\":\"HTTP $HTTP_STATUS\",\"models\":[]}" > "$OUT/models.json"
            echo "status=error" >> "$GITHUB_OUTPUT"
          fi

      - name: Upload OpenAI artifact
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: openai-models
          path: /tmp/gh-aw/model-inventory/openai/models.json
          if-no-files-found: error
          retention-days: 7

  collect_anthropic_models:
    runs-on: ubuntu-latest
    needs: [activation]
    permissions:
      contents: read
    steps:
      - name: Fetch Anthropic models
        id: fetch
        shell: bash
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          set -euo pipefail
          OUT="/tmp/gh-aw/model-inventory/anthropic"
          mkdir -p "$OUT"
          if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
            echo '{"provider":"anthropic","error":"ANTHROPIC_API_KEY not set","models":[]}' > "$OUT/models.json"
            echo "status=skipped" >> "$GITHUB_OUTPUT"
            exit 0
          fi
          HTTP_STATUS=$(curl -sf -o "$OUT/raw.json" -w "%{http_code}" \
            -H "x-api-key: $ANTHROPIC_API_KEY" \
            -H "anthropic-version: 2023-06-01" \
            https://api.anthropic.com/v1/models) || true
          if [ "${HTTP_STATUS:-0}" = "200" ]; then
            jq '{provider:"anthropic",models:[.data[].id]|sort}' "$OUT/raw.json" > "$OUT/models.json"
            echo "status=ok" >> "$GITHUB_OUTPUT"
          else
            echo "{\"provider\":\"anthropic\",\"error\":\"HTTP $HTTP_STATUS\",\"models\":[]}" > "$OUT/models.json"
            echo "status=error" >> "$GITHUB_OUTPUT"
          fi

      - name: Upload Anthropic artifact
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: anthropic-models
          path: /tmp/gh-aw/model-inventory/anthropic/models.json
          if-no-files-found: error
          retention-days: 7

  collect_gemini_models:
    runs-on: ubuntu-latest
    needs: [activation]
    permissions:
      contents: read
    steps:
      - name: Fetch Gemini models
        id: fetch
        shell: bash
        env:
          GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
        run: |
          set -euo pipefail
          OUT="/tmp/gh-aw/model-inventory/gemini"
          mkdir -p "$OUT"
          if [ -z "${GEMINI_API_KEY:-}" ]; then
            echo '{"provider":"gemini","error":"GEMINI_API_KEY not set","models":[]}' > "$OUT/models.json"
            echo "status=skipped" >> "$GITHUB_OUTPUT"
            exit 0
          fi
          HTTP_STATUS=$(curl -sf -o "$OUT/raw.json" -w "%{http_code}" \
            "https://generativelanguage.googleapis.com/v1beta/models?key=${GEMINI_API_KEY}") || true
          if [ "${HTTP_STATUS:-0}" = "200" ]; then
            jq '{provider:"gemini",models:[.models[].name | ltrimstr("models/")]|sort}' "$OUT/raw.json" > "$OUT/models.json"
            echo "status=ok" >> "$GITHUB_OUTPUT"
          else
            echo "{\"provider\":\"gemini\",\"error\":\"HTTP $HTTP_STATUS\",\"models\":[]}" > "$OUT/models.json"
            echo "status=error" >> "$GITHUB_OUTPUT"
          fi

      - name: Upload Gemini artifact
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: gemini-models
          path: /tmp/gh-aw/model-inventory/gemini/models.json
          if-no-files-found: error
          retention-days: 7

  collect_copilot_models:
    runs-on: ubuntu-latest
    needs: [activation]
    permissions:
      contents: read
    steps:
      - name: Fetch Copilot models
        id: fetch
        shell: bash
        env:
          COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
        run: |
          set -euo pipefail
          OUT="/tmp/gh-aw/model-inventory/copilot"
          mkdir -p "$OUT"
          if [ -z "${COPILOT_GITHUB_TOKEN:-}" ]; then
            echo '{"provider":"copilot","error":"COPILOT_GITHUB_TOKEN not set","models":[]}' > "$OUT/models.json"
            echo "status=skipped" >> "$GITHUB_OUTPUT"
            exit 0
          fi
          HTTP_STATUS=$(curl -sf -o "$OUT/raw.json" -w "%{http_code}" \
            -H "Authorization: Bearer $COPILOT_GITHUB_TOKEN" \
            -H "Copilot-Integration-Id: copilot-chat" \
            https://api.githubcopilot.com/models) || true
          if [ "${HTTP_STATUS:-0}" = "200" ]; then
            jq '{provider:"copilot",models:[.data[].id]|sort}' "$OUT/raw.json" > "$OUT/models.json"
            echo "status=ok" >> "$GITHUB_OUTPUT"
          else
            echo "{\"provider\":\"copilot\",\"error\":\"HTTP $HTTP_STATUS\",\"models\":[]}" > "$OUT/models.json"
            echo "status=error" >> "$GITHUB_OUTPUT"
          fi

      - name: Upload Copilot artifact
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: copilot-models
          path: /tmp/gh-aw/model-inventory/copilot/models.json
          if-no-files-found: error
          retention-days: 7

steps:
  - name: Download all model artifacts
    uses: actions/download-artifact@v8.0.1
    with:
      path: /tmp/gh-aw/model-inventory/artifacts

  - name: Merge artifacts into combined inventory
    shell: bash
    run: |
      INVENTORY="/tmp/gh-aw/model-inventory/inventory.json"
      jq -s '.' /tmp/gh-aw/model-inventory/artifacts/*/models.json > "$INVENTORY"
      echo "Combined inventory written to $INVENTORY"
      cat "$INVENTORY"

tools:
  cli-proxy: true
  bash:
    - "cat /tmp/gh-aw/model-inventory/inventory.json"
    - "jq . /tmp/gh-aw/model-inventory/inventory.json"
    - "jq . /tmp/gh-aw/model-inventory/artifacts/*/models.json"
    - "find /tmp/gh-aw/model-inventory -type f"
    - "cat pkg/workflow/model_aliases.go"
  github:
    toolsets: [default]

safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[model-inventory] "
    labels: [automation, models]
    max: 1
    close-older-issues: true

imports:
  - shared/otel.md
---

# Daily Model Inventory Checker

You are an AI model catalog analyst for `${{ github.repository }}`.

Your task is to analyze the current model inventories from all configured AI providers and
determine whether the built-in model alias mapping in `pkg/workflow/model_aliases.go` needs
updating.

## Inputs

The pre-job steps have already fetched model lists from each provider's API and merged them into:

- Combined inventory: `/tmp/gh-aw/model-inventory/inventory.json`
- Individual provider files: `/tmp/gh-aw/model-inventory/artifacts/<provider>-models/models.json`

Each entry in the inventory has the form:
```json
{
  "provider": "openai",
  "models": ["gpt-4o", "gpt-4o-mini", ...]
}
```

If a provider's API key was not configured, the entry will have `"error": "... not set"` and an
empty `models` array. Skip providers with errors or empty model lists.

## Built-in Alias Reference

Read `pkg/workflow/model_aliases.go` to understand the current alias definitions. The current
built-in aliases are:

| Alias | Resolves to |
|-------|-------------|
| `sonnet` | Anthropic Sonnet family |
| `haiku` | Anthropic Haiku family |
| `opus` | Anthropic Opus family |
| `gpt-5` | OpenAI GPT-5 family |
| `gpt-5-mini` | OpenAI GPT-5 mini family |
| `gpt-5-codex` | OpenAI GPT-5 Codex family |
| `gemini-flash` | Google Gemini Flash family |
| `gemini-pro` | Google Gemini Pro family |
| `small` / `mini` | Lightweight/fast models |
| `large` | Full-capability models |
| `auto` | Convenience alias for `large` |

The alias pattern syntax is:
- `"vendor/model*id"` — wildcard glob (e.g. `"copilot/*sonnet*"`)
- `"alias"` — recursive reference to another alias

## Task

### Step 1: Load and Validate the Inventory

Read the combined inventory from `/tmp/gh-aw/model-inventory/inventory.json`. List the
providers that returned data and the count of models available from each.

### Step 2: Identify New or Updated Model Families

Compare the live model list against the current aliases in `pkg/workflow/model_aliases.go`.
Look for:

1. **New model generations** — e.g. a new `claude-sonnet-5` or `gpt-6` that is not covered by
   any existing alias glob pattern.
2. **New model families** — entirely new families (e.g. a new reasoning or multimodal line)
   that have no corresponding alias.
3. **Stale aliases** — patterns that no longer match any live model.
4. **Task-oriented alias gaps** — useful semantic aliases that are missing, such as:
   - `summarization-model` → a fast, cost-effective model good at summarization
   - `coding-model` → a model optimized for code generation
   - `reasoning-model` → a model with extended reasoning/thinking capability
   - `vision-model` → a model that supports image input

### Step 3: Propose Alias Mapping Updates

For each finding, produce a concrete YAML snippet showing the proposed new or updated alias entry
in the `models:` frontmatter format. Use the alias pattern syntax:

```yaml
models:
  new-alias:
    - "copilot/vendor-model-id*"
    - "vendor/vendor-model-id*"
```

Focus on aliases that provide genuine value to workflow authors. Prioritize:
- Adding patterns to existing aliases to cover new model generations
- Adding new semantic task-oriented aliases
- Updating patterns that are stale

### Step 4: Create Issue

If you found any meaningful updates to propose, create a GitHub issue using `create_issue`.

**Issue title format**: `Model alias inventory update - YYYY-MM-DD`

**Issue body structure** (use h3 `###` or lower — never h1/h2):

```markdown
### Summary

Brief description of what was found.

- Providers queried: OpenAI, Anthropic, Gemini, Copilot
- Total models found: <count>
- Proposed changes: <count>

### Provider Model Counts

| Provider | Models Available | Status |
|----------|-----------------|--------|
| openai   | 42              | ✅ ok  |
| anthropic | 15             | ✅ ok  |
| gemini   | 28              | ✅ ok  |
| copilot  | 35              | ✅ ok  |

### Proposed Alias Updates

For each change, explain:
1. **What**: The alias name and new/updated patterns
2. **Why**: Which live model(s) prompted this change
3. **Syntax**: YAML snippet ready to copy into `pkg/workflow/model_aliases.go`

<details>
<summary><b>Full Model Lists by Provider</b></summary>

List the complete sorted model IDs for each provider.

</details>

### Notes

Any caveats, stale patterns removed, or aliases that are already well-covered.
```

If no updates are needed (all live models are already covered by existing aliases and no new
task-oriented aliases are warranted), create an issue with title
`Model alias inventory - no changes needed - YYYY-MM-DD` and a brief summary confirming
coverage is up to date.

{{#runtime-import shared/noop-reminder.md}}
