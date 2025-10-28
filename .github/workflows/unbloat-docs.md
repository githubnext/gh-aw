---
name: Documentation Unbloat
on:
  # Daily at 2am PST (10am UTC)
  schedule:
    - cron: "0 22 * * *"  # Daily at 10 PM UTC
  
  # Command trigger for /unbloat in PR comments
  command:
    name: unbloat
    events: [pull_request_comment]
  
  # Manual trigger for testing
  workflow_dispatch:

# Minimal permissions - safe-outputs handles write operations
permissions:
  contents: read
  actions: read
  pull-requests: read

# AI engine configuration
engine:
  id: claude
  max-turns: 90  # Reduce from avg 115 turns

# Network access for documentation best practices research
network:
  allowed:
    - defaults
    - github

# Tools configuration
tools:
  cache-memory: true
  github:
    allowed:
      - get_repository
      - get_file_contents
      - list_commits
      - get_pull_request
      - search_pull_requests
  edit:
  playwright:
    args: ["--viewport-size", "1920x1080"]
  bash:
    - "find docs/src/content/docs -name '*.md'"
    - "wc -l *"
    - "grep -n *"
    - "cat *"
    - "head *"
    - "tail *"
    - "cd *"
    - "node *"
    - "curl *"
    - "ps *"
    - "kill *"
    - "sleep *"
    - "mkdir *"
    - "cp *"
    - "mv *"

# Safe outputs configuration
safe-outputs:
  create-pull-request:
    title-prefix: "[docs] "
    labels: [documentation, automation]
    draft: true
  add-comment:
    max: 1
  upload-assets:

# Timeout (based on avg 6.8min runtime + buffer)
timeout_minutes: 12

# Build steps for documentation
steps:
  - name: Checkout repository
    uses: actions/checkout@v5

  - name: Setup Node.js
    uses: actions/setup-node@v5
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

## Context

- **Repository**: ${{ github.repository }}

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
ls -la /tmp/gh-aw/cache-memory/
cat /tmp/gh-aw/cache-memory/cleaned-files.txt 2>/dev/null || echo "No previous cleanups found"
```

This will help you avoid re-cleaning files that were recently processed.

### 2. Find Documentation Files

Scan the `docs/` directory for markdown files, excluding code-generated files:
```bash
find docs/src/content/docs -name '*.md' -type f ! -name 'frontmatter-full.md'
```

**IMPORTANT**: Exclude `frontmatter-full.md` as it is automatically generated from the JSON schema by `scripts/generate-schema-docs.js` and should not be manually edited.

Focus on files that were recently modified or are in the `docs/src/content/docs/samples/` directory.

{{#if ${{ github.event.pull_request.number }}}}
**Pull Request Context**: Since this workflow is running in the context of PR #${{ github.event.pull_request.number }}, prioritize reviewing the documentation files that were modified in this pull request. Use the GitHub API to get the list of changed files:

```bash
# Get PR file changes using the get_pull_request tool
```

Focus on markdown files in the `docs/` directory that appear in the PR's changed files list.
{{/if}}

### 3. Select ONE File to Improve

**IMPORTANT**: Work on only **ONE file at a time** to keep changes small and reviewable.

**NEVER select these code-generated files**:
- `docs/src/content/docs/reference/frontmatter-full.md` - Auto-generated from JSON schema

Choose the file most in need of improvement based on:
- Recent modification date
- File size (larger files may have more bloat)
- Number of bullet points or repetitive patterns
- **Files NOT in the cleaned-files.txt cache** (avoid duplicating recent work)
- **Files NOT in the exclusion list above** (avoid editing generated files)

### 4. Analyze the File

Read the selected file and identify bloat:
- Count bullet points - are there excessive lists?
- Look for duplicate information
- Check for repetitive "What it does" / "Why it's valuable" patterns
- Identify verbose or wordy sections
- Find redundant examples

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

### 7. Update Cache Memory

After improving the file, update the cache memory to track the cleanup:
```bash
echo "$(date -u +%Y-%m-%d) - Cleaned: <filename>" >> /tmp/gh-aw/cache-memory/cleaned-files.txt
```

This helps future runs avoid re-cleaning the same files.

### 8. Take Screenshots of Modified Documentation

After making changes to a documentation file, take screenshots of the rendered page in the Astro Starlight website:

#### Build and Start Documentation Server

1. Go to the `docs` directory (this was already done in the build steps)
2. Start the documentation development server using `npm run dev`
3. Wait for the server to fully start (it should be accessible on `http://localhost:4321/gh-aw/`)
4. Verify the server is running by making a curl request to test accessibility

#### Take Screenshots with Playwright

For the modified documentation file(s):

1. Determine the URL path for the modified file (e.g., if you modified `docs/src/content/docs/guides/getting-started.md`, the URL would be `http://localhost:4321/gh-aw/guides/getting-started/`)
2. Use Playwright to navigate to the documentation page URL
3. Wait for the page to fully load (including all CSS, fonts, and images)
4. Take a full-page HD screenshot of the documentation page (1920x1080 viewport is configured)
5. The screenshot will be saved in `/tmp/gh-aw/mcp-logs/playwright/` by Playwright (e.g., `/tmp/gh-aw/mcp-logs/playwright/getting-started.png`)

#### Upload Screenshots

1. Use the `upload asset` tool from safe-outputs to upload each screenshot file
2. The tool will return a URL for each uploaded screenshot
3. Keep track of these URLs to include in the PR description

#### Report Blocked Domains

While taking screenshots, monitor the browser console for any blocked network requests:
- Look for CSS files that failed to load
- Look for font files that failed to load
- Look for any other resources that were blocked by network policies

If you encounter any blocked domains:
1. Note the domain names and resource types (CSS, fonts, images, etc.)
2. Include this information in the PR description under a "Blocked Domains" section
3. Example format: "Blocked: fonts.googleapis.com (fonts), cdn.example.com (CSS)"

### 9. Create Pull Request

After improving ONE file:
1. Verify your changes preserve all essential information
2. Update cache memory with the cleaned file
3. Take HD screenshots (1920x1080 viewport) of the modified documentation page(s)
4. Upload the screenshots and collect the URLs
5. Create a pull request with your improvements
6. Include in the PR description:
   - Which file you improved
   - What types of bloat you removed
   - Estimated word count or line reduction
   - Summary of changes made
   - **Screenshot URLs**: Links to the uploaded screenshots showing the modified documentation pages
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
