---
name: Documentation Unbloat
description: Reviews and simplifies documentation by reducing verbosity while maintaining clarity and completeness
on:
  # Daily (scattered execution time)
  schedule: daily
  
  # Command trigger for /unbloat in PR comments
  slash_command:
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
    args: ["--viewport-size", "1920x1080"]
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

      # Pre-select the best candidate: largest uncleaned file (by line count)
      SELECTED=""
      while IFS= read -r f; do
        [ -f "$f" ] || continue
        # Skip if recently cleaned (match against full path to avoid basename collisions)
        if awk -v cutoff="$RECENT_CUTOFF" -v fpath="$f" \
          'NF>0 && $1>=cutoff && index($0,fpath)>0{found=1}END{exit !found}' \
          /tmp/gh-aw/cache-memory/cleaned-files.txt 2>/dev/null; then
          continue
        fi
        LINES=$(wc -l < "$f" 2>/dev/null || echo 0)
        printf '%06d %s\n' "$LINES" "$f"
      done < /tmp/gh-aw/agent/candidate-files.txt 2>/dev/null \
        | sort -rn | head -1 | awk '{$1=""; print substr($0,2)}' \
        > /tmp/gh-aw/agent/selected-file.txt || true

      SELECTED=$(cat /tmp/gh-aw/agent/selected-file.txt 2>/dev/null | xargs)
      if [ -z "$SELECTED" ] || [ ! -f "$SELECTED" ]; then
        # Fallback: first file in candidate list
        head -1 /tmp/gh-aw/agent/candidate-files.txt > /tmp/gh-aw/agent/selected-file.txt || true
        SELECTED=$(cat /tmp/gh-aw/agent/selected-file.txt 2>/dev/null | xargs)
      fi

      printf '{"pass":true,"reason":"All pre-flight checks passed. %d uncleaned candidates available.","uncleaned":%d,"total":%d,"selected_file":"%s"}\n' \
        "$UNCLEANED" "$UNCLEANED" "$TOTAL" "$SELECTED" \
        > /tmp/gh-aw/agent/preflight.json

      echo "Pre-flight passed: $UNCLEANED uncleaned candidates out of $TOTAL eligible files"
      echo "Candidate files written to /tmp/gh-aw/agent/candidate-files.txt"
      echo "Pre-selected file: $SELECTED ($(wc -l < "$SELECTED" 2>/dev/null) lines)"

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

Read `/tmp/gh-aw/agent/preflight.json`. If `"pass"` is `false`, call `noop` with the `"reason"` value and stop.
Only proceed if `"pass"` is `true`.

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

### 1. Read Your Pre-selected File

The pre-flight step has already selected the best candidate file for you. Read the file path:

```bash
cat /tmp/gh-aw/agent/selected-file.txt
```

Then read the file itself:

```bash
cat <file_path_from_above>
```

{{#if ${{ github.event.pull_request.number }}}}
**Pull Request Context**: This workflow was triggered by PR #${{ github.event.pull_request.number }}. Use the GitHub MCP `pull_request_read` tool to get changed files. If any `docs/` markdown files appear in the PR's changed files list, use that file instead of the pre-selected file. Otherwise proceed with the pre-selected file.
{{/if}}

Scan the file for bloat: count bullet points per section, identify repeated "What it does" / "Why it's valuable" patterns, and note sections with 5+ consecutive bullets that could become prose.

### 2. Remove Bloat

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

### 3. Preserve Essential Content

**DO NOT REMOVE**:
- Technical accuracy or specific details
- Links to external resources
- Code examples (though you can consolidate duplicates)
- Critical warnings or notes
- Frontmatter metadata

### 4. Create a Branch for Your Changes

Before making changes, create a new branch with a descriptive name:
```bash
git checkout -b docs/unbloat-<filename-without-extension>
```

For example, if you're cleaning `validation-timing.md`, create branch `docs/unbloat-validation-timing`.

**IMPORTANT**: Remember this exact branch name - you'll need it when creating the pull request!

### 5. Update Cache Memory

After improving the file, update the cache memory to track the cleanup (use the full file path so future runs can match it correctly):
```bash
echo "$(date -u +%Y-%m-%d) - Cleaned: <full_file_path>" >> /tmp/gh-aw/cache-memory/cleaned-files.txt
```

### 6. Take Screenshots of Modified Documentation

After making changes to a documentation file, take screenshots of the rendered page in the Astro Starlight website:

#### Build and Start Documentation Server

Follow the shared **Documentation Server Lifecycle Management** instructions:
1. Start the preview server (section "Starting the Documentation Preview Server")
2. Wait for readiness (section "Waiting for Server Readiness")
3. Optionally verify accessibility (section "Verifying Server Accessibility")

#### Take Screenshots with Playwright

For the modified documentation file(s):

1. Determine the URL path for the modified file (e.g., if you modified `docs/src/content/docs/guides/getting-started.md`, the URL would be `http://localhost:4321/gh-aw/guides/getting-started/`)
2. Use Playwright to navigate to the documentation page URL
3. Wait for the page to fully load (including all CSS, fonts, and images)
4. Take a full-page HD screenshot of the documentation page (1920x1080 viewport is configured)
5. The screenshot will be saved in `/tmp/gh-aw/mcp-logs/playwright/` by Playwright (e.g., `/tmp/gh-aw/mcp-logs/playwright/getting-started.png`)

#### Verify Screenshots Were Saved

**IMPORTANT**: Before uploading, verify that Playwright successfully saved the screenshots:

```bash
# List files in the output directory to confirm screenshots were saved
ls -lh /tmp/gh-aw/mcp-logs/playwright/
```

**If no screenshot files are found:**
- Report this in the PR description under an "Issues" section
- Include the error message or reason why screenshots couldn't be captured
- Do not proceed with upload-asset if no files exist

#### Upload Screenshots

1. Call the `upload_asset` safe-output tool for each screenshot using absolute paths (for example `/tmp/gh-aw/mcp-logs/playwright/<screenshot>.png`)
2. Record the returned asset URL for each screenshot to include in the PR description

#### Report Blocked Domains

While taking screenshots, monitor the browser console for any blocked network requests:
- Look for CSS files that failed to load
- Look for font files that failed to load
- Look for any other resources that were blocked by network policies

If you encounter any blocked domains:
1. Note the domain names and resource types (CSS, fonts, images, etc.)
2. Include this information in the PR description under a "Blocked Domains" section
3. Example format: "Blocked: fonts.googleapis.com (fonts), cdn.example.com (CSS)"

#### Cleanup Server

After taking screenshots, follow the shared **Documentation Server Lifecycle Management** instructions for cleanup (section "Stopping the Documentation Server").

### 7. Create Pull Request

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

Begin by reading your pre-selected file and improving it!

{{#runtime-import shared/noop-reminder.md}}
