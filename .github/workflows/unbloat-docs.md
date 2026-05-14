---
emoji: "📝"
name: Documentation Unbloat
description: Reviews and simplifies documentation by reducing verbosity while maintaining clarity and completeness
on:
  # Daily (scattered execution time)
  schedule: daily
  
  # Command trigger for /unbloat in PR comments
  slash_command:
    strategy: centralized
    name: unbloat
    events: [pull_request_comment]
  
  # Manual trigger for testing
  workflow_dispatch:
  
  # Skip if there is already an open draft PR from this workflow to avoid duplicate work
  skip-if-match: 'is:pr is:open is:draft label:doc-unbloat'

# Minimal permissions - safe-outputs handles write operations
permissions:
  contents: read
  pull-requests: read
  issues: read

strict: true

runtimes:
  node:
    version: "22"

# AI engine configuration
engine:
  id: claude
  max-turns: 90  # Reduce from avg 115 turns

# Shared instructions
imports:
  - uses: shared/daily-pr-base.md
    with:
      title-prefix: "[docs] "
      expires: "2d"
      labels: [documentation, automation, doc-unbloat]
      reviewers: [copilot]
  - shared/docs-server-lifecycle.md

# Network access for documentation best practices research
  - shared/observability-otlp.md
network:
  allowed:
    - defaults
    - github

# Sandbox configuration - AWF is enabled by default but making it explicit for clarity
sandbox:
  agent: awf

# Tools configuration
tools:
  cli-proxy: true
  cache-memory: true
  github:
    mode: gh-proxy
    toolsets: [default]
  edit:
  playwright:
    mode: cli
  bash:
    - "find docs/src/content/docs *"
    - "find /tmp/gh-aw/cache-memory *"
    - "wc -l *"
    - "wc"
    - "grep -n *"
    - "grep -rL *"
    - "grep *"
    - "xargs *"
    - "date *"
    - "date"
    - "awk *"
    - "git"
    - "cat *"
    - "head *"
    - "tail *"
    - "cd *"
    - "node *"
    - "npm *"
    - "curl *"
    - "ps *"
    - "kill *"
    - "sleep *"
    - "echo *"
    - "mkdir *"
    - "cp *"
    - "mv *"

# Safe outputs configuration
safe-outputs:
  create-pull-request:
    expires: 2d
    title-prefix: "[docs] "
    labels: [documentation, automation, doc-unbloat]
    reviewers: [copilot]
    draft: true
    auto-merge: true
    fallback-as-issue: false
  add-comment:
    max: 1
  upload-asset:
    max: 10
    allowed-exts: [.png, .jpg, .jpeg, .svg]
  messages:
    footer: "> 🗜️ *Compressed by [{workflow_name}]({run_url})*{effective_tokens_suffix}{history_link}"
    run-started: "📦 Time to slim down! [{workflow_name}]({run_url}) is trimming the excess from this {event_type}..."
    run-success: "🗜️ Docs on a diet! [{workflow_name}]({run_url}) has removed the bloat. Lean and mean! 💪"
    run-failure: "📦 Unbloating paused! [{workflow_name}]({run_url}) {status}. The docs remain... fluffy."

# Timeout (increased from 12min after timeout issues; aligns with similar doc workflows)
timeout-minutes: 30

# Pre-agent steps: deterministic precomputation before the AI engine starts
pre-agent-steps:
  - name: Pre-flight checks
    run: |
      mkdir -p /tmp/gh-aw/agent

      # Check 1: verify docs directory structure exists
      DIR_COUNT=$(find docs/src/content/docs -maxdepth 1 -type d 2>/dev/null | wc -l)
      if [ "$DIR_COUNT" -eq 0 ]; then
        echo '{"pass":false,"reason":"Pre-flight failed: docs/src/content/docs directory not found — documentation structure is missing or repository is not set up correctly."}' \
          > /tmp/gh-aw/agent/preflight.json
        exit 0
      fi

      # Check 2: count editable markdown files
      TOTAL=$(find docs/src/content/docs -path '*/blog*' -prune \
        -o -name '*.md' -type f ! -name 'frontmatter-full.md' -print \
        | xargs grep -rL 'disable-agentic-editing: true' 2>/dev/null \
        | wc -l)
      if [ "$TOTAL" -eq 0 ]; then
        echo '{"pass":false,"reason":"Pre-flight failed: no editable markdown files found in docs/src/content/docs (all files may be protected or excluded)."}' \
          > /tmp/gh-aw/agent/preflight.json
        exit 0
      fi

      # Check 3: count uncleaned candidates (not cleaned in the past 7 days)
      RECENT_CUTOFF=$(date -d '7 days ago' '+%Y-%m-%d' 2>/dev/null \
        || date -v-7d '+%Y-%m-%d' 2>/dev/null \
        || echo "0000-00-00")
      CLEANED=$(awk -v cutoff="$RECENT_CUTOFF" \
        'NF>0 && $1>=cutoff{count++} END{print count+0}' \
        /tmp/gh-aw/cache-memory/cleaned-files.txt 2>/dev/null || echo "0")
      UNCLEANED=$(( TOTAL - CLEANED ))
      if [ "$UNCLEANED" -le 0 ]; then
        echo '{"pass":false,"reason":"Pre-flight check: all eligible documentation files were cleaned recently — nothing to do this run."}' \
          > /tmp/gh-aw/agent/preflight.json
        exit 0
      fi

      # All checks passed — write candidate file list and preflight result
      find docs/src/content/docs -path '*/blog*' -prune \
        -o -name '*.md' -type f ! -name 'frontmatter-full.md' -print \
        | xargs grep -rL 'disable-agentic-editing: true' 2>/dev/null \
        > /tmp/gh-aw/agent/candidate-files.txt
      printf '{"pass":true,"reason":"All pre-flight checks passed. %d uncleaned candidates available.","uncleaned":%d,"total":%d}\n' \
        "$UNCLEANED" "$UNCLEANED" "$TOTAL" \
        > /tmp/gh-aw/agent/preflight.json

      echo "Pre-flight passed: $UNCLEANED uncleaned candidates out of $TOTAL eligible files"
      echo "Candidate files written to /tmp/gh-aw/agent/candidate-files.txt"

  - name: Start documentation dev server
    run: |
      # Skip if pre-flight check failed — no need to start the server
      if [ "$(jq -r '.pass' /tmp/gh-aw/agent/preflight.json 2>/dev/null)" != "true" ]; then
        echo "Pre-flight check failed, skipping server startup"
        exit 0
      fi
      mkdir -p /tmp/gh-aw
      cd docs
      nohup npm run dev -- --host 0.0.0.0 --port 4321 > /tmp/gh-aw/preview.log 2>&1 &
      PID=$!
      echo $PID > /tmp/gh-aw/server.pid
      echo "Dev server started (PID: $PID)"

  - name: Wait for documentation server readiness
    run: |
      # Skip if pre-flight check failed — server was not started
      if [ "$(jq -r '.pass' /tmp/gh-aw/agent/preflight.json 2>/dev/null)" != "true" ]; then
        echo "Pre-flight check failed, skipping server readiness check"
        exit 0
      fi
      URL="http://localhost:4321/gh-aw/"
      STATUS=""
      echo "Readiness check target: $URL"
      echo "Preview log: /tmp/gh-aw/preview.log"
      for i in $(seq 1 45); do
        STATUS=$(curl -sS -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 5 "$URL" || true)
        [ "$STATUS" = "200" ] && echo "Server ready at $URL" && break
        if [ -z "$STATUS" ]; then STATUS="curl_error"; fi
        echo "Waiting for server... ($i/45) (status: $STATUS)"
        sleep 3
      done
      if [ "$STATUS" != "200" ]; then
        echo "Dev server failed to start after 135 seconds:"
        echo "Final readiness status: $STATUS"
        cat /tmp/gh-aw/preview.log || true
        exit 1
      fi

  - name: Write Playwright base URL
    run: |
      # Skip if pre-flight check failed — server was not started
      if [ "$(jq -r '.pass' /tmp/gh-aw/agent/preflight.json 2>/dev/null)" != "true" ]; then
        echo "Pre-flight check failed, skipping Playwright base URL setup"
        exit 0
      fi
      mkdir -p /tmp/gh-aw/agent
      echo "http://localhost:4321/gh-aw/" > /tmp/gh-aw/agent/playwright-base-url.txt
      echo "Playwright base URL: http://localhost:4321/gh-aw/"

# Build steps for documentation
steps:
  - name: Checkout repository
    uses: actions/checkout@v6.0.2
    with:
      persist-credentials: false

  - name: Setup Node.js
    uses: actions/setup-node@v6.4.0
    with:
      node-version: '24'
      cache: 'npm'
      cache-dependency-path: 'docs/package-lock.json'

  - name: Install dependencies
    working-directory: ./docs
    run: npm ci

  - name: Build documentation
    working-directory: ./docs
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: npm run build

---

# Documentation Unbloat Workflow

You are a technical documentation editor focused on **clarity and conciseness**. Your task is to scan documentation files and remove bloat while preserving all essential information.

## 0. Pre-flight Validation

Read `/tmp/gh-aw/agent/preflight.json`. If `"pass"` is `false`, **immediately** call `noop` with the `"reason"` value and stop — do not read any other files beyond `preflight.json`, do not proceed with any further steps. This is mandatory: failing to call `noop` when preflight fails causes a safe-output compliance error.
Only proceed if `"pass"` is `true`.

The list of candidate files is already available at `/tmp/gh-aw/agent/candidate-files.txt` (one path per line).

---

## Context

- **Repository**: ${{ github.repository }}
- **Triggered by**: ${{ github.actor }}

## What is Documentation Bloat?

Documentation bloat includes:

1. **Duplicate content**: Same information repeated in different sections
2. **Excessive bullet points**: Long lists that could be condensed into prose or tables
3. **Redundant examples**: Multiple examples showing the same concept
4. **Verbose descriptions**: Overly wordy explanations that could be more concise
5. **Repetitive structure**: The same "What it does" / "Why it's valuable" pattern overused

## Your Task

Analyze documentation files in the `docs/` directory and make targeted improvements:

### 1. Check Cache Memory for Previous Cleanups

First, check the cache folder for notes about previous cleanups:
```bash
find /tmp/gh-aw/cache-memory/ -maxdepth 1 -ls
cat /tmp/gh-aw/cache-memory/cleaned-files.txt 2>/dev/null || echo "No previous cleanups found"
```

This will help you avoid re-cleaning files that were recently processed.

### 2. Find Documentation Files

Use `search` to semantically search for documentation files that may contain bloat (verbose descriptions, repetitive patterns, excessive bullet points). This is faster and more targeted than listing all files:

- Query for areas known to accumulate bloat: `search("verbose documentation long examples repeated patterns")`
- Query for specific topics recently added: `search("recently added feature documentation")`
- Read the returned file paths to assess their content

Then scan the `docs/` directory for all markdown files, excluding code-generated files and blog posts:
```bash
find docs/src/content/docs -path 'docs/src/content/docs/blog' -prune -o -name '*.md' -type f ! -name 'frontmatter-full.md' -print
```

**IMPORTANT**: Exclude these directories and files:
- `docs/src/content/docs/blog/` - Blog posts have a different writing style and purpose
- `frontmatter-full.md` - Automatically generated from the JSON schema by `scripts/generate-schema-docs.js` and should not be manually edited
- **Files with `disable-agentic-editing: true` in frontmatter** - These files are protected from automated editing

Focus on files that were recently modified or are in the `docs/src/content/docs/` directory (excluding blog).

{{#if ${{ github.event.pull_request.number }}}}
**Pull Request Context**: Since this workflow is running in the context of PR #${{ github.event.pull_request.number }}, prioritize reviewing the documentation files that were modified in this pull request. Use the GitHub API to get the list of changed files:

```bash
# Get PR file changes using the pull_request_read tool
```

Focus on markdown files in the `docs/` directory that appear in the PR's changed files list.
{{/if}}

### 3. Select ONE File to Improve

**IMPORTANT**: Work on only **ONE file at a time** to keep changes small and reviewable.

**NEVER select these directories or code-generated files**:
- `docs/src/content/docs/blog/` - Blog posts have a different writing style and should not be unbloated
- `docs/src/content/docs/reference/frontmatter-full.md` - Auto-generated from JSON schema
- **Files with `disable-agentic-editing: true` in frontmatter** - These files are explicitly protected from automated editing

Before selecting a file, check its frontmatter to ensure it doesn't have `disable-agentic-editing: true`:
```bash
# Check if a file has disable-agentic-editing set to true
head -20 <filename> | grep -A1 "^---" | grep "disable-agentic-editing: true"
# If this returns a match, SKIP this file - it's protected
```

Choose the file most in need of improvement based on:
- Recent modification date
- File size (larger files may have more bloat)
- Number of bullet points or repetitive patterns
- **Files NOT in the cleaned-files.txt cache** (avoid duplicating recent work)
- **Files NOT in the exclusion list above** (avoid editing generated files)
- **Files WITHOUT `disable-agentic-editing: true` in frontmatter** (respect protection flag)

### 4. Analyze the File

Use the `file-bloat-analyzer` agent, passing the selected file path as the input, to get a structured bloat inventory.
Review the returned JSON to plan targeted edits: focus on `heavy_bullet_sections`,
`duplicate_headings`, and high `repetitive_pattern_count`.

### 5. Remove Bloat

Make targeted edits to improve clarity:

**Consolidate bullet points**: 
- Convert long bullet lists into concise prose or tables
- Remove redundant points that say the same thing differently

**Eliminate duplicates**:
- Remove repeated information
- Consolidate similar sections

**Condense verbose text**:
- Make descriptions more direct and concise
- Remove filler words and phrases
- Keep technical accuracy while reducing word count

**Standardize structure**:
- Reduce repetitive "What it does" / "Why it's valuable" patterns
- Use varied, natural language

**Simplify code samples**:
- Remove unnecessary complexity from code examples
- Focus on demonstrating the core concept clearly
- Eliminate boilerplate or setup code unless essential for understanding
- Keep examples minimal yet complete
- Use realistic but simple scenarios

### 6. Preserve Essential Content

**DO NOT REMOVE**:
- Technical accuracy or specific details
- Links to external resources
- Code examples (though you can consolidate duplicates)
- Critical warnings or notes
- Frontmatter metadata

### 7. Create a Branch for Your Changes

Before making changes, create a new branch with a descriptive name:
```bash
git checkout -b docs/unbloat-<filename-without-extension>
```

For example, if you're cleaning `validation-timing.md`, create branch `docs/unbloat-validation-timing`.

**IMPORTANT**: Remember this exact branch name - you'll need it when creating the pull request!

### 8. Update Cache Memory

After improving the file, update the cache memory to track the cleanup:
```bash
echo "$(date -u +%Y-%m-%d) - Cleaned: <filename>" >> /tmp/gh-aw/cache-memory/cleaned-files.txt
```

This helps future runs avoid re-cleaning the same files.

### 9. Take Screenshots of Modified Documentation

After making changes to a documentation file, take a screenshot of the rendered page using the `doc-page-screenshotter` sub-agent. The documentation server is already running — no setup is needed.

#### Determine the Page URL

Read the Playwright base URL written by the pre-agent setup step:
```bash
cat /tmp/gh-aw/agent/playwright-base-url.txt
```

Convert the modified file path to a page URL path:
- Strip the `docs/src/content/docs/` prefix and the `.md` suffix, then append `/`
- Example: `docs/src/content/docs/guides/ephemerals.md` → `guides/ephemerals/`

Append that page path to the base URL (e.g., `http://localhost:4321/gh-aw/guides/ephemerals/`).

#### Capture Screenshots via Sub-Agent

Use the `doc-page-screenshotter` agent, passing the full page URL as input. The sub-agent handles all Playwright interactions and returns a structured JSON result:

```json
{
  "success": true,
  "screenshots": ["/tmp/gh-aw/screenshots/doc-screenshot.png"],
  "blocked_domains": [],
  "error": null
}
```

#### Verify Screenshots Were Saved

Check the `screenshots` array returned by the `doc-page-screenshotter` sub-agent:

**If `screenshots` is empty or `success` is `false`:**
- Report this in the PR description under an "Issues" section
- Include the `error` value from the sub-agent result
- Do not proceed with upload-asset

#### Upload Screenshots

1. Call the `upload_asset` safe-output tool for each screenshot using absolute paths (for example `/tmp/gh-aw/screenshots/<screenshot>.png`)
2. Record the returned asset URL for each screenshot to include in the PR description

#### Report Blocked Domains

The `doc-page-screenshotter` sub-agent returns `blocked_domains` in its result. If any domains are listed:
1. Include this information in the PR description under a "Blocked Domains" section
2. Example format: "Blocked: fonts.googleapis.com (fonts), cdn.example.com (CSS)"

#### Cleanup Server

After taking screenshots, follow the shared **Documentation Server Lifecycle Management** instructions for cleanup (section "Stopping the Documentation Server").

### 10. Create Pull Request

After improving ONE file:
1. Verify your changes preserve all essential information
2. Update cache memory with the cleaned file
3. Take HD screenshots (1920x1080 viewport) of the modified documentation page(s)
4. Upload the screenshots as assets (see "Upload Screenshots" section above) and collect the returned asset URLs
5. Create a pull request with your improvements
   - **IMPORTANT**: When calling the create_pull_request tool, do NOT pass a "branch" parameter - let it auto-detect the current branch you created
   - Or if you must specify the branch, use the exact branch name you created earlier (NOT "main")
6. Include in the PR description:
   - Which file you improved
   - What types of bloat you removed
   - Estimated word count or line reduction
   - Summary of changes made
   - **Screenshots**: List the uploaded asset URLs for the before/after screenshots
   - **Blocked Domains (if any)**: List any CSS/font/resource domains that were blocked during screenshot capture

## Example Improvements

### Before (Bloated):
```markdown
### Tool Name
Description of the tool.

- **What it does**: This tool does X, Y, and Z
- **Why it's valuable**: It's valuable because A, B, and C
- **How to use**: You use it by doing steps 1, 2, 3, 4, 5
- **When to use**: Use it when you need X
- **Benefits**: Gets you benefit A, benefit B, benefit C
- **Learn more**: [Link](url)
```

### After (Concise):
```markdown
### Tool Name
Description of the tool that does X, Y, and Z to achieve A, B, and C.

Use it when you need X by following steps 1-5. [Learn more](url)
```

## Guidelines

1. **One file per run**: Focus on making one file significantly better
2. **Preserve meaning**: Never lose important information
3. **Be surgical**: Make precise edits, don't rewrite everything
4. **Maintain tone**: Keep the neutral, technical tone
5. **Test locally**: If possible, verify links and formatting are still correct
6. **Document changes**: Clearly explain what you improved in the PR

## Success Criteria

A successful run:
- ✅ Improves exactly **ONE** documentation file
- ✅ Reduces bloat by at least 20% (lines, words, or bullet points)
- ✅ Preserves all essential information
- ✅ Creates a clear, reviewable pull request
- ✅ Explains the improvements made
- ✅ Includes HD screenshots (1920x1080) of the modified documentation page(s) in the Astro Starlight website
- ✅ Reports any blocked domains for CSS/fonts (if encountered)

Begin by scanning the docs directory and selecting the best candidate for improvement!

{{#runtime-import shared/noop-reminder.md}}

## agent: `file-bloat-analyzer`
---
model: small
description: Reads a single documentation file and returns a structured inventory of bloat indicators
---
You are a documentation bloat analysis agent. The file path to analyze is provided as the first line of your input (or as the argument you are invoked with). Read that file using the `bash` tool (`cat <file_path>`) and return a structured JSON inventory of bloat indicators.

Analyze the file for:
- **bullet_count**: Total number of bullet/list items in the file
- **heavy_bullet_sections**: Array of section headings that contain 5 or more consecutive bullet points (5+ is the threshold for sections likely to benefit from prose consolidation)
- **duplicate_headings**: Array of heading texts that appear more than once
- **repetitive_pattern_count**: Count of occurrences of repetitive "What it does" / "Why it's valuable" / "How to use" patterns
- **estimated_line_count**: Total number of lines in the file
- **bloat_score**: A score from 0–10 estimating overall bloat severity (0 = clean, 10 = extremely bloated)
- **top_bloat_reason**: One-sentence summary of the primary bloat issue found

Return a JSON object only — no prose, no extra text:

```json
{
  "file": "<file path>",
  "bullet_count": 42,
  "heavy_bullet_sections": ["### Tool Configuration", "## Features"],
  "duplicate_headings": ["## Overview"],
  "repetitive_pattern_count": 7,
  "estimated_line_count": 320,
  "bloat_score": 7,
  "top_bloat_reason": "Excessive bullet lists in Tool Configuration and Features sections with repetitive What/Why/How patterns."
}
```

## agent: `doc-page-screenshotter`
---
model: small
description: Navigates to a documentation page URL using Playwright and captures a full-page screenshot, returning a structured JSON result with screenshot paths and any blocked domains
---
You are a documentation screenshot agent. Your input is a full page URL to screenshot (e.g., `http://localhost:4321/gh-aw/guides/ephemerals/`).

1. Navigate to the URL using `playwright-cli`. Use `browser_navigate` with the URL. If navigation times out (Vite dev server can be slow with the default `load` wait), fall back to `browser_run_code_unsafe` with `waitUntil: 'domcontentloaded'`:
   ```bash
   # Primary: direct navigation
   playwright-cli browser_navigate --url "<URL>"
   ```
   If the above times out, use this fallback instead:
   ```bash
   playwright-cli browser_run_code_unsafe --code "async (page) => { await page.goto('<URL>', { waitUntil: 'domcontentloaded', timeout: 30000 }); return { url: page.url(), title: await page.title() }; }"
   ```

2. Set viewport to HD (1920×1080) and take a full-page screenshot:
   ```bash
   mkdir -p /tmp/gh-aw/screenshots
   playwright-cli browser_resize --width 1920 --height 1080
   playwright-cli browser_take_screenshot --filename /tmp/gh-aw/screenshots/doc-screenshot.png --full-page true
   ```

3. Check the browser console for blocked network requests:
   ```bash
   playwright-cli browser_console_messages
   ```
   Look for errors mentioning blocked CSS, font, or image domains.

4. Verify the screenshot was saved:
   ```bash
   ls -lh /tmp/gh-aw/screenshots/
   ```

Return a JSON object only — no prose, no extra text:

```json
{
  "success": true,
  "screenshots": ["/tmp/gh-aw/screenshots/doc-screenshot.png"],
  "blocked_domains": [],
  "error": null
}
```

If navigation or screenshot fails (timeout, connection error, file not written), set `"success": false`, `"screenshots": []`, and describe the failure in `"error"`.
