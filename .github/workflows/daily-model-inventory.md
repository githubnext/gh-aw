---
private: true
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
engine:
  id: copilot
  copilot-sdk: true
  copilot-sdk-driver: .github/drivers/copilot_sdk_driver_sample_node.cjs
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
          OUT="/tmp/gh-aw/agent/model-inventory/openai"
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
            /tmp/gh-aw/agent/model-inventory/openai/models.json
            /tmp/gh-aw/agent/model-inventory/openai/raw.json
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
          OUT="/tmp/gh-aw/agent/model-inventory/anthropic"
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
            /tmp/gh-aw/agent/model-inventory/anthropic/models.json
            /tmp/gh-aw/agent/model-inventory/anthropic/raw.json
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
          OUT="/tmp/gh-aw/agent/model-inventory/gemini"
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
            /tmp/gh-aw/agent/model-inventory/gemini/models.json
            /tmp/gh-aw/agent/model-inventory/gemini/raw.json
          if-no-files-found: error
          retention-days: 7

  collect_copilot_billing_models:
    runs-on: ubuntu-latest
    needs: [activation]
    permissions:
      contents: read
    steps:
      - name: Fetch Copilot models and pricing
        id: fetch
        shell: bash
        run: |
          set -euo pipefail
          OUT="/tmp/gh-aw/agent/model-inventory/copilot-billing"
          mkdir -p "$OUT"
          python3 - <<'PYEOF'
          import json, sys, urllib.request, html.parser

          # NOTE: If GitHub's documentation URL structure changes, this URL must be updated manually.
          URL = "https://docs.github.com/en/copilot/reference/copilot-billing/models-and-pricing"
          EXCLUDED_MODELS = {"gpt-4o-mini", "gpt-4.1", "gpt-4o", "gpt-5.4-nano"}

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
              with open("/tmp/gh-aw/agent/model-inventory/copilot-billing/models.json", "w") as f:
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
                      model_id = entry.get("Model", "").strip()
                      if model_id in EXCLUDED_MODELS:
                          continue
                      models.append(entry)

          result = {"source": URL, "excluded_models": sorted(EXCLUDED_MODELS), "headers": parser.headers, "models": models}
          out_path = "/tmp/gh-aw/agent/model-inventory/copilot-billing/models.json"
          with open(out_path, "w") as f:
              json.dump(result, f, indent=2)
          print(f"Extracted {len(models)} model pricing entries", file=sys.stderr)
          PYEOF
          echo "status=ok" >> "$GITHUB_OUTPUT"

      - name: Upload Copilot billing models artifact
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: copilot-billing-models
          path: /tmp/gh-aw/agent/model-inventory/copilot-billing/models.json
          if-no-files-found: error
          retention-days: 7

steps:
  - name: Download all model artifacts
    uses: actions/download-artifact@v8.0.1
    with:
      path: /tmp/gh-aw/agent/model-inventory/artifacts

  - name: Predownload models.dev API index
    shell: bash
    run: |
      set -euo pipefail
      OUT="/tmp/gh-aw/agent/model-inventory/models-dev"
      mkdir -p "$OUT"
      curl -fsS https://models.dev/api.json -o "$OUT/api.json"
      echo "Downloaded models.dev API index to $OUT/api.json"

  - name: Merge artifacts into combined inventory
    shell: bash
    run: |
      INVENTORY="/tmp/gh-aw/agent/model-inventory/inventory.json"
      jq -s '.' /tmp/gh-aw/agent/model-inventory/artifacts/*/models.json > "$INVENTORY"
      echo "Combined inventory written to $INVENTORY"
      cat "$INVENTORY"

tools:
  cli-proxy: true
  playwright:
    mode: cli
  bash:
    - "*"
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
  - shared/otlp.md
---

# Daily Model Inventory Checker

You are an AI model catalog analyst for `${{ github.repository }}`.

Your task is to analyze the current model inventories from all configured AI providers and
determine whether the built-in model alias mapping in `pkg/workflow/data/model_aliases.json` needs
updating.

## Inputs

The pre-job steps have already fetched model lists from OpenAI, Anthropic, and Gemini, then merged
them into:

- Combined inventory: `/tmp/gh-aw/agent/model-inventory/inventory.json`
- Individual provider files: `/tmp/gh-aw/agent/model-inventory/artifacts/<provider>-models/models.json`
- Raw provider responses: `/tmp/gh-aw/agent/model-inventory/artifacts/<provider>-models/raw.json`
- Predownloaded models.dev API index: `/tmp/gh-aw/agent/model-inventory/models-dev/api.json`
- Copilot live provider metadata: `/tmp/gh-aw/agent/model-inventory/reflect.json` (generated in
  Step 0 below; filter `.endpoints[] | select(.provider == "copilot") | .models`). If the
  file contains an `error` field, treat Copilot data as unavailable for this run and
  continue with the remaining providers.

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
Note: Copilot model data is fetched during agent execution in Step 0 below because the AWF
`api-proxy` hostname is only reachable from within the agent Docker network. The fetch
enriches null models via `models_url` where possible (see `.github/aw/llms.md`). Copilot
serves models from multiple vendors (Anthropic, OpenAI, Google), and those models may include
`vendor` metadata.

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

### Step 0: Fetch Copilot Models from API Proxy

Before loading the inventory, fetch Copilot model metadata from the AWF `api-proxy` `/reflect`
endpoint from within this agent execution context and write it to:
`/tmp/gh-aw/agent/model-inventory/reflect.json`.

Run:

```bash
set -euo pipefail
OUT="/tmp/gh-aw/agent/model-inventory/reflect.json"
mkdir -p "$(dirname "$OUT")"
if ! curl -fsS http://api-proxy:10000/reflect > "$OUT"; then
  printf '%s' '{"endpoints":[],"error":"reflect endpoint unavailable"}' > "$OUT"
fi
# For configured endpoints where /reflect returned null models, fetch directly from
# models_url (the api-proxy injects auth headers). Mirrors enrichReflectModels() in
# awf_reflect.cjs — see .github/aw/llms.md for endpoint port/URL reference.
while IFS= read -r entry; do
  provider=$(printf '%s' "$entry" | jq -r '.provider')
  models_url=$(printf '%s' "$entry" | jq -r '.models_url')
  echo "Fetching models for $provider from $models_url"
  if raw=$(curl -fsS "$models_url" 2>&1); then
    ids=$(printf '%s' "$raw" | jq -c '[.data[].id] // empty' 2>&1) || {
      echo "Warning: failed to parse models response for $provider: $ids"
      ids=""
    }
    if [ -n "$ids" ]; then
      jq --arg p "$provider" --argjson m "$ids" \
        '(.endpoints[] | select(.provider == $p) | .models) |= $m' \
        "$OUT" > "${OUT}.tmp" && mv "${OUT}.tmp" "$OUT"
      echo "Enriched $provider with $(printf '%s' "$ids" | jq 'length') model(s)"
    else
      echo "Warning: no model IDs extracted for $provider"
    fi
  else
    echo "Warning: failed to fetch models_url for $provider ($models_url): $raw"
  fi
done < <(jq -c '.endpoints[]? | select(.configured == true and .models == null and .models_url != null)' "$OUT" 2>/dev/null || true)
echo "Copilot reflect metadata written to $OUT"
```

### Step 1: Load and Validate the Inventory

Read the combined inventory from `/tmp/gh-aw/agent/model-inventory/inventory.json`. Then read
the `/tmp/gh-aw/agent/model-inventory/reflect.json` file from Step 0 and extract the configured
`copilot` endpoint (`.endpoints[] | select(.provider == "copilot" and .configured)`).
Also read `/tmp/gh-aw/agent/model-inventory/models-dev/api.json` as a secondary cross-provider
catalog snapshot.

List the providers that returned data and the count of models available from each, including
Copilot from the reflect file.

If the reflect file has an `error` field, or contains no `copilot` endpoint, note Copilot as
unavailable and continue.

### Step 2: Explore Raw API Fields

For each provider that returned data, examine the raw response to identify all available fields:

- OpenAI / Anthropic / Gemini: `/tmp/gh-aw/agent/model-inventory/artifacts/<provider>-models/raw.json`
- Copilot: `/tmp/gh-aw/agent/model-inventory/reflect.json` filtered to the `copilot` endpoint object

Specifically look for:

- **Context window metadata**: input/output token limits (e.g. `inputTokenLimit`, `outputTokenLimit`,
  `capabilities.limits.max_context_window_tokens`, `capabilities.limits.max_output_tokens`)
- **Capability flags**: supported generation methods, vision support, tool use, streaming
  (e.g. `supportedGenerationMethods`, `capabilities.supports.vision`, `capabilities.type`)
- **Billing/pricing fields**: any field that conveys relative cost, a multiplier, a tier name,
  or a premium indicator (e.g. `billing.multiplier`, `policy`, `tier`, `premium`, `cost_multiplier`)
- **Model metadata**: `display_name`, `vendor`, `version`, `created_at`/`created`

For `models.dev/api.json`, focus on normalized provider/model IDs and any capability or pricing-like
metadata that can improve alias coverage checks when provider APIs are partial.

Summarize which fields are present and which carry useful data worth including in future cached
inventories.

### Step 3: Validate models.json pricing data

Read the current built-in pricing payloads from:

- `pkg/cli/data/models.json`
- `actions/setup/js/models.json`

Treat these two files as a mirrored pair: proposed updates must keep them identical.

The pre-job step has also fetched the **official GitHub Copilot models/pricing table** from the
documentation page and stored it as:

- `/tmp/gh-aw/agent/model-inventory/artifacts/copilot-billing-models/models.json`

This file is extracted from:
`https://docs.github.com/en/copilot/reference/copilot-billing/models-and-pricing`.

Use the Copilot reflect endpoint (`billing.multiplier`) and the docs pricing table as validation
sources for `models.json` pricing fields. Prefer reflect data when available for Copilot model
multiplier validation, and use docs table values as a secondary cross-check.

Treat `gpt-4o-mini`, `gpt-4.1`, `gpt-4o`, and `gpt-5.4-nano` as intentionally deprecated
Copilot-facing model IDs. Keep ignoring them even if they appear in the reflect data, docs table,
`models.dev`, or live provider inventories: do not propose adding or restoring them in
`pkg/cli/data/models.json` (or `actions/setup/js/models.json`), and exclude them from
missing/discrepancy tables.

Use `models.dev/api.json` as the refresh source for the baseline `models.json` payload.
When pricing updates are required, first run:

```bash
make refresh-models-json
```

Then validate/adjust pricing entries against reflect and docs-derived data.

For each provider's enriched data, validate pricing/model coverage for each model:

1. **Copilot reflect data** — use the `copilot` endpoint's `models` list from
   `/tmp/gh-aw/agent/model-inventory/reflect.json` as the primary source. For each model, use
   the `billing.multiplier` field as the authoritative ET multiplier value. Compare against the
   matching `github/<model-id>` entry in `models.json`, and list discrepancies or missing models.
   Cross-reference against the docs table as a secondary validation source.

2. **Gemini API** — use `inputTokenLimit` / `outputTokenLimit` as an approximate proxy for model
   complexity (this is an inference heuristic, not a definitive billing mapping).
   Large-context, high-output-limit models typically correspond to higher-priced tiers; smaller
   Flash models to lower-priced tiers. Flag any models whose limits suggest a pricing-tier change
   versus what is currently in `models.json`.

3. **OpenAI API** — use `owned_by` and model-ID naming conventions (e.g. `-mini`, `-nano`, `o1`,
   `o3`) to cross-check current pricing tiers. Flag missing models or likely mismatches.

4. **Anthropic API** — use `display_name` family grouping (haiku/sonnet/opus) to validate
   current pricing tiers. Flag any new model IDs not yet in `models.json`.

Produce a consolidated pricing gap table listing:
- Models present in the live inventory but **missing** from `models.json` — include
  the provider name for each model (e.g. "openai", "anthropic", "gemini", "copilot")
- Models in `models.json` that are **not currently returned** by live APIs; keep these
  in the payload as historical entries (do not propose automatic removals)
- Models where the **inferred pricing tier or multiplier signal** differs from the stored pricing

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
- Pricing gaps found: <count>

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

### models.json Pricing Analysis

#### Missing from models.json

| Model ID | Provider | Inferred Pricing | Basis |
|----------|----------|-----------------:|-------|
| ...      | ...      | ...              | ...   |

#### Historical entries not currently returned

List model IDs that appear in `models.json` but are absent from all live inventories.
Treat these as historical records that should remain in the payload unless a human explicitly
decides to delete them.

#### Inferred vs stored pricing discrepancies

| Model ID | Stored Pricing | Inferred Pricing | Inferred From |
|----------|---------------:|-----------------:|---------------|
| ...      | ...            | ...              | ...           |

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

Any caveats, historical entries retained, or aliases that are already well-covered.
```

If no updates are needed (all live models are already covered by existing aliases, all
`models.json` pricing entries are up to date, and no new task-oriented aliases are warranted), create an issue with
title `Model alias inventory - no changes needed - YYYY-MM-DD` and a brief summary confirming
coverage is up to date.

{{#runtime-import shared/noop-reminder.md}}
