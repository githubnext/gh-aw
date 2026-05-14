---
emoji: "📦"
name: Daily Model Inventory Checker
description: Queries model lists from OpenAI, Anthropic, and Google APIs daily, uses AWF /reflect for Copilot models, then analyzes the combined inventory to propose updates to the builtin model alias mapping
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
            echo '{"provider":"openai","error":"OPENAI_API_KEY not set"}' > "$OUT/raw.json"
            echo "status=skipped" >> "$GITHUB_OUTPUT"
            exit 0
          fi
          HTTP_STATUS=$(curl -sf -o "$OUT/raw.json" -w "%{http_code}" \
            -H "Authorization: Bearer $OPENAI_API_KEY" \
            https://api.openai.com/v1/models) || true
          if [ "${HTTP_STATUS:-0}" = "200" ]; then
            jq '{
              provider: "openai",
              models: [
                .data[] | {
                  id,
                  owned_by,
                  created
                }
              ] | sort_by(.id)
            }' "$OUT/raw.json" > "$OUT/models.json"
            echo "status=ok" >> "$GITHUB_OUTPUT"
          else
            echo "{\"provider\":\"openai\",\"error\":\"HTTP $HTTP_STATUS\",\"models\":[]}" > "$OUT/models.json"
            echo "status=error" >> "$GITHUB_OUTPUT"
          fi

      - name: Upload OpenAI artifacts
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: openai-models
          path: |
            /tmp/gh-aw/model-inventory/openai/models.json
            /tmp/gh-aw/model-inventory/openai/raw.json
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
            echo '{"provider":"anthropic","error":"ANTHROPIC_API_KEY not set"}' > "$OUT/raw.json"
            echo "status=skipped" >> "$GITHUB_OUTPUT"
            exit 0
          fi
          HTTP_STATUS=$(curl -sf -o "$OUT/raw.json" -w "%{http_code}" \
            -H "x-api-key: $ANTHROPIC_API_KEY" \
            -H "anthropic-version: 2023-06-01" \
            https://api.anthropic.com/v1/models) || true
          if [ "${HTTP_STATUS:-0}" = "200" ]; then
            jq '{
              provider: "anthropic",
              models: [
                .data[] | {
                  id,
                  display_name,
                  created_at,
                  type
                }
              ] | sort_by(.id)
            }' "$OUT/raw.json" > "$OUT/models.json"
            echo "status=ok" >> "$GITHUB_OUTPUT"
          else
            echo "{\"provider\":\"anthropic\",\"error\":\"HTTP $HTTP_STATUS\",\"models\":[]}" > "$OUT/models.json"
            echo "status=error" >> "$GITHUB_OUTPUT"
          fi

      - name: Upload Anthropic artifacts
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: anthropic-models
          path: |
            /tmp/gh-aw/model-inventory/anthropic/models.json
            /tmp/gh-aw/model-inventory/anthropic/raw.json
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
            echo '{"provider":"gemini","error":"GEMINI_API_KEY not set"}' > "$OUT/raw.json"
            echo "status=skipped" >> "$GITHUB_OUTPUT"
            exit 0
          fi
          HTTP_STATUS=$(curl -sf -o "$OUT/raw.json" -w "%{http_code}" \
            "https://generativelanguage.googleapis.com/v1beta/models?key=${GEMINI_API_KEY}") || true
          if [ "${HTTP_STATUS:-0}" = "200" ]; then
            jq '{
              provider: "gemini",
              models: [
                .models[] | {
                  id: (.name | ltrimstr("models/")),
                  display_name: .displayName,
                  description: .description,
                  input_token_limit: .inputTokenLimit,
                  output_token_limit: .outputTokenLimit,
                  supported_generation_methods: .supportedGenerationMethods,
                  version: .version
                }
              ] | sort_by(.id)
            }' "$OUT/raw.json" > "$OUT/models.json"
            echo "status=ok" >> "$GITHUB_OUTPUT"
          else
            echo "{\"provider\":\"gemini\",\"error\":\"HTTP $HTTP_STATUS\",\"models\":[]}" > "$OUT/models.json"
            echo "status=error" >> "$GITHUB_OUTPUT"
          fi

      - name: Upload Gemini artifacts
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: gemini-models
          path: |
            /tmp/gh-aw/model-inventory/gemini/models.json
            /tmp/gh-aw/model-inventory/gemini/raw.json
          if-no-files-found: error
          retention-days: 7

  collect_copilot_billing_multipliers:
    runs-on: ubuntu-latest
    needs: [activation]
    permissions:
      contents: read
    steps:
      - name: Fetch Copilot billing multipliers
        id: fetch
        shell: bash
        run: |
          set -euo pipefail
          OUT="/tmp/gh-aw/model-inventory/copilot-billing"
          mkdir -p "$OUT"
          python3 - <<'PYEOF'
          import json, sys, urllib.request, html.parser

          # NOTE: If GitHub's documentation URL structure changes, this URL must be updated manually.
          URL = "https://docs.github.com/en/copilot/reference/copilot-billing/model-multipliers-for-annual-plans"

          class TableParser(html.parser.HTMLParser):
              def __init__(self):
                  super().__init__()
                  self.in_table = False
                  self.headers = []
                  self.rows = []
                  self.current_row = []
                  self.current_cell = None
                  self.cell_text = []

              def handle_starttag(self, tag, attrs):
                  if tag == "table":
                      self.in_table = True
                  elif self.in_table and tag in ("th", "td"):
                      self.current_cell = tag
                      self.cell_text = []
                  elif self.in_table and tag == "tr":
                      self.current_row = []

              def handle_endtag(self, tag):
                  if tag == "table":
                      self.in_table = False
                  elif self.in_table and tag in ("th", "td"):
                      text = "".join(self.cell_text).strip()
                      if self.current_cell == "th":
                          self.headers.append(text)
                      else:
                          self.current_row.append(text)
                      self.current_cell = None
                  elif self.in_table and tag == "tr":
                      if self.current_row:
                          self.rows.append(self.current_row)

              def handle_data(self, data):
                  if self.current_cell is not None:
                      self.cell_text.append(data)

          req = urllib.request.Request(URL, headers={"User-Agent": "Mozilla/5.0"})
          try:
              with urllib.request.urlopen(req, timeout=30) as resp:
                  html_content = resp.read().decode("utf-8", errors="replace")
          except Exception as e:
              result = {"source": URL, "error": str(e), "headers": [], "models": []}
              with open("/tmp/gh-aw/model-inventory/copilot-billing/multipliers.json", "w") as f:
                  json.dump(result, f, indent=2)
              print(f"Error fetching page: {e}", file=sys.stderr)
              sys.exit(0)

          parser = TableParser()
          parser.feed(html_content)

          models = []
          if parser.headers and parser.rows:
              for row in parser.rows:
                  if len(row) == len(parser.headers):
                      entry = {parser.headers[i]: row[i] for i in range(len(parser.headers))}
                      models.append(entry)

          result = {"source": URL, "headers": parser.headers, "models": models}
          out_path = "/tmp/gh-aw/model-inventory/copilot-billing/multipliers.json"
          with open(out_path, "w") as f:
              json.dump(result, f, indent=2)
          print(f"Extracted {len(models)} model multiplier entries", file=sys.stderr)
          PYEOF
          echo "status=ok" >> "$GITHUB_OUTPUT"

      - name: Upload Copilot billing multipliers artifact
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: copilot-billing-multipliers
          path: /tmp/gh-aw/model-inventory/copilot-billing/multipliers.json
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
  playwright:
    mode: cli
  bash:
    - "cat /tmp/gh-aw/model-inventory/inventory.json"
    - "jq . /tmp/gh-aw/model-inventory/inventory.json"
    - "jq . /tmp/gh-aw/model-inventory/artifacts/*/models.json"
    - "jq . /tmp/gh-aw/model-inventory/artifacts/*/raw.json"
    - "jq '[.data[] | keys] | add | unique' /tmp/gh-aw/model-inventory/artifacts/openai-models/raw.json"
    - "jq '[.data[] | keys] | add | unique' /tmp/gh-aw/model-inventory/artifacts/anthropic-models/raw.json"
    - "jq '[.models[] | keys] | add | unique' /tmp/gh-aw/model-inventory/artifacts/gemini-models/raw.json"
    - "mkdir -p /tmp/gh-aw/model-inventory && (curl -fsS http://api-proxy:10000/reflect > /tmp/gh-aw/model-inventory/reflect.json || printf \"%s\" \"{\\\"endpoints\\\":[],\\\"error\\\":\\\"reflect endpoint unavailable\\\"}\" > /tmp/gh-aw/model-inventory/reflect.json)"
    - "jq . /tmp/gh-aw/model-inventory/reflect.json"
    - "jq \".endpoints[] | select(.provider == \\\"copilot\\\") | .models\" /tmp/gh-aw/model-inventory/reflect.json"
    - "cat /tmp/gh-aw/model-inventory/artifacts/copilot-billing-multipliers/multipliers.json"
    - "jq . /tmp/gh-aw/model-inventory/artifacts/copilot-billing-multipliers/multipliers.json"
    - "jq '.models[]' /tmp/gh-aw/model-inventory/artifacts/copilot-billing-multipliers/multipliers.json"
    - "find /tmp/gh-aw/model-inventory -type f"
    - "cat pkg/workflow/data/model_aliases.json"
    - "cat pkg/cli/data/model_multipliers.json"
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

  - shared/observability-otlp.md
---

# Daily Model Inventory Checker

You are an AI model catalog analyst for `${{ github.repository }}`.

Your task is to analyze the current model inventories from all configured AI providers and
determine whether the built-in model alias mapping in `pkg/workflow/data/model_aliases.json` needs
updating.

## Inputs

The pre-job steps have already fetched model lists from OpenAI, Anthropic, and Gemini, then merged
them into:

- Combined inventory: `/tmp/gh-aw/model-inventory/inventory.json`
- Individual provider files: `/tmp/gh-aw/model-inventory/artifacts/<provider>-models/models.json`
- Raw provider responses: `/tmp/gh-aw/model-inventory/artifacts/<provider>-models/raw.json`
- Copilot live provider metadata: available via `http://api-proxy:10000/reflect` (filter
  `.endpoints[] | select(.provider == "copilot")`). If `/reflect` is unavailable, treat Copilot
  data as unavailable for this run and continue with the remaining providers.

Each enriched `models.json` entry has the form (fields vary by provider):
```json
{
  "provider": "copilot",
  "models": [
    {
      "id": "claude-sonnet-4-5",
      "name": "Claude Sonnet 4.5",
      "vendor": "anthropic",
      "capabilities": { "limits": { "max_context_window_tokens": 200000 } },
      "billing": { "multiplier": 1.0 }
    }
  ]
}
```
Note: Copilot model data must be read from the AWF `/reflect` endpoint. Copilot serves models from
multiple vendors (Anthropic, OpenAI, Google), and those models may include `vendor` metadata.

If a provider's API key was not configured, the entry will have `"error": "... not set"` and an
empty `models` array. Skip providers with errors or empty model lists.

## Built-in Alias Reference

Read `pkg/workflow/data/model_aliases.json` to understand the current alias definitions. The current
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

Read the combined inventory from `/tmp/gh-aw/model-inventory/inventory.json`. Then query
`http://api-proxy:10000/reflect` and extract the configured `copilot` endpoint.

List the providers that returned data and the count of models available from each, including
Copilot from `/reflect`.

If `/reflect` returns no `copilot` endpoint (or reports an error), note Copilot as unavailable and
continue.

### Step 2: Explore Raw API Fields

For each provider that returned data, examine the raw response to identify all available fields:

- OpenAI / Anthropic / Gemini: `/tmp/gh-aw/model-inventory/artifacts/<provider>-models/raw.json`
- Copilot: `http://api-proxy:10000/reflect` filtered to the `copilot` endpoint object

Specifically look for:

- **Context window metadata**: input/output token limits (e.g. `inputTokenLimit`, `outputTokenLimit`,
  `capabilities.limits.max_context_window_tokens`, `capabilities.limits.max_output_tokens`)
- **Capability flags**: supported generation methods, vision support, tool use, streaming
  (e.g. `supportedGenerationMethods`, `capabilities.supports.vision`, `capabilities.type`)
- **Billing/pricing fields**: any field that conveys relative cost, a multiplier, a tier name,
  or a premium indicator (e.g. `billing.multiplier`, `policy`, `tier`, `premium`, `cost_multiplier`)
- **Model metadata**: `display_name`, `vendor`, `version`, `created_at`/`created`

Summarize which fields are present and which carry useful data worth including in future cached
inventories.

### Step 3: Infer Token Multipliers

Read the current built-in multiplier table from `pkg/cli/data/model_multipliers.json`.

The pre-job step has also fetched the **official GitHub Copilot billing multipliers** from the
documentation page and stored them as:

- `/tmp/gh-aw/model-inventory/artifacts/copilot-billing-multipliers/multipliers.json`

This file contains the authoritative ET multipliers per model extracted from
`https://docs.github.com/en/copilot/reference/copilot-billing/model-multipliers-for-annual-plans`,
with columns `Model`, `Current multiplier`, and `New multiplier`.

**Use the docs table as the primary (authoritative) source of ET multipliers.** Prefer the
**New multiplier** column for upcoming billing schedule comparisons. If the docs table fetch
failed or returned an empty model list, fall back to the heuristics below.

For each provider's enriched data, attempt to infer or validate the ET multiplier for each model:

1. **Copilot `/reflect` endpoint** — use the `copilot` endpoint's `models` list as the live model
   source, then match model names/IDs against the official docs table first. If a match is found,
   use the `New multiplier` as the authoritative value. Compare against the matching entry in
   `model_multipliers.json`, and list discrepancies or missing models.

2. **Gemini API** — use `inputTokenLimit` / `outputTokenLimit` as an approximate proxy for model
   complexity (this is an inference heuristic, not a definitive billing mapping).
   Large-context, high-output-limit models typically correspond to Pro-tier multipliers (~1.0);
   smaller Flash models to lower multipliers (~0.1–0.2). Flag any models whose limits suggest a
   tier change versus what is currently in `model_multipliers.json`.

3. **OpenAI API** — use `owned_by` and model-ID naming conventions (e.g. `-mini`, `-nano`, `o1`,
   `o3`) to cross-check current multipliers. Flag missing models or likely mismatches.

4. **Anthropic API** — use `display_name` family grouping (haiku/sonnet/opus) to validate
   current multipliers. Flag any new model IDs not yet in `model_multipliers.json`.

Produce a consolidated multiplier gap table listing:
- Models present in the live inventory but **missing** from `model_multipliers.json` — include
  the provider name for each model (e.g. "openai", "anthropic", "gemini", "copilot")
- Models in `model_multipliers.json` that are **no longer returned** by any API (stale)
- Models where the **inferred multiplier** differs from the stored one

### Step 4: Identify New or Updated Model Families

Compare the live model list against the current aliases in `pkg/workflow/data/model_aliases.json`.
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

### Step 5: Propose Alias Mapping Updates

For each finding from Step 4, produce a concrete JSON snippet showing the proposed new or updated
alias entry in the `aliases` object in `pkg/workflow/data/model_aliases.json`. Use the alias pattern syntax:

```json
{
  "new-alias": [
    "copilot/vendor-model-id*",
    "vendor/vendor-model-id*"
  ]
}
```

Focus on aliases that provide genuine value to workflow authors. Prioritize:
- Adding patterns to existing aliases to cover new model generations
- Adding new semantic task-oriented aliases
- Updating patterns that are stale

### Step 6: Create Issue

If you found any meaningful updates to propose, create a GitHub issue using `create_issue`.

**Issue title format**: `Model alias inventory update - YYYY-MM-DD`

**Issue body structure** (use h3 `###` or lower — never h1/h2):

```markdown
### Summary

Brief description of what was found.

- Providers queried: OpenAI, Anthropic, Gemini, Copilot
- Total models found: <count>
- Proposed alias changes: <count>
- Multiplier gaps found: <count>

### Provider Model Counts

| Provider | Models Available | Status |
|----------|-----------------|--------|
| openai   | 42              | ✅ ok  |
| anthropic | 15             | ✅ ok  |
| gemini   | 28              | ✅ ok  |
| copilot  | 35              | ✅ ok  |

### Raw API Fields Discovered

For each provider, list noteworthy fields found in the raw response that are now captured
in the enriched `models.json` artifact (context limits, capabilities, billing fields, etc.).

### Token Multiplier Analysis

#### Missing from model_multipliers.json

| Model ID | Provider | Inferred Multiplier | Basis |
|----------|----------|--------------------:|-------|
| ...      | ...      | ...                 | ...   |

#### Stale entries (no longer returned by any API)

List model IDs that appear in `model_multipliers.json` but are absent from all live inventories.

#### Inferred vs stored discrepancies

| Model ID | Stored Multiplier | Inferred Multiplier | Inferred From |
|----------|------------------:|--------------------:|---------------|
| ...      | ...               | ...                 | ...           |

### Proposed Alias Updates

For each change, explain:
1. **What**: The alias name and new/updated patterns
2. **Why**: Which live model(s) prompted this change
3. **Syntax**: JSON snippet showing the new or updated entry for the `aliases` object in `pkg/workflow/data/model_aliases.json`

<details>
<summary><b>Full Model Lists by Provider</b></summary>

List the complete sorted model IDs for each provider.

</details>

### Notes

Any caveats, stale patterns removed, or aliases that are already well-covered.
```

If no updates are needed (all live models are already covered by existing aliases, all
multipliers are up to date, and no new task-oriented aliases are warranted), create an issue with
title `Model alias inventory - no changes needed - YYYY-MM-DD` and a brief summary confirming
coverage is up to date.

{{#runtime-import shared/noop-reminder.md}}
