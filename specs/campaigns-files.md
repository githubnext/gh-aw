# Campaign Files Architecture

This document describes how campaigns are discovered, compiled, and executed in GitHub Agentic Workflows. It covers the complete lifecycle from campaign spec files to running workflows.

## Overview

Campaigns are a first-class feature in gh-aw that enable coordinated, multi-repository initiatives. The campaign system consists of:

1. **Campaign Spec Files** (`.campaign.md`) - Declarative YAML frontmatter defining campaign configuration
2. **Discovery Script** (`campaign_discovery.cjs`) - JavaScript that searches GitHub for campaign items
3. **Orchestrator Generator** - Go code that builds agentic workflows from campaign specs
4. **Compiled Workflows** (`.campaign.lock.yml`) - GitHub Actions workflows that run the campaigns

## File Locations

```
.github/workflows/
├── <campaign-id>.campaign.md          # Campaign spec (source of truth)
├── <campaign-id>.campaign.g.md        # Generated orchestrator (debug artifact, not committed)
└── <campaign-id>.campaign.lock.yml    # Compiled workflow (committed)

actions/setup/js/
└── campaign_discovery.cjs             # Discovery precomputation script

pkg/campaign/
├── spec.go                            # Campaign spec data structures
├── loader.go                          # Campaign discovery and loading
├── orchestrator.go                    # Orchestrator generation
└── validation.go                      # Campaign spec validation
```

## Campaign Discovery Process

### 1. Local Repository Discovery

**Implementation**: `pkg/campaign/loader.go:LoadSpecs()`

The campaign system discovers campaign specs by scanning the local repository:

```go
// Scan .github/workflows/ for *.campaign.md files
workflowsDir := filepath.Join(rootDir, ".github", "workflows")
entries, err := os.ReadDir(workflowsDir)

// For each .campaign.md file:
//   1. Read file contents
//   2. Parse YAML frontmatter using parser.ExtractFrontmatterFromContent()
//   3. Unmarshal to CampaignSpec struct
//   4. Set default ID and Name if not provided
//   5. Store relative path in ConfigPath field
```

**Key features**:
- Only scans `.campaign.md` files (not `.md` or `.g.md`)
- Returns empty slice if `.github/workflows/` doesn't exist (no error)
- Populates `ConfigPath` with repository-relative path
- Auto-generates ID from filename if not specified in frontmatter

### 2. Campaign Spec Structure

**Implementation**: `pkg/campaign/spec.go:CampaignSpec`

Campaign specs use YAML frontmatter with these key fields:

```yaml
---
id: security-q1-2025
name: Security Q1 2025
version: v1
state: active

# Project integration
project-url: https://github.com/orgs/ORG/projects/1
tracker-label: campaign:security-q1-2025

# Associated workflows
workflows:
  - vulnerability-scanner
  - dependency-updater

# Repo-memory configuration
memory-paths:
  - memory/campaigns/security-q1-2025/**
metrics-glob: memory/campaigns/security-q1-2025/metrics/*.json
cursor-glob: memory/campaigns/security-q1-2025/cursor.json

# Governance
governance:
  max-new-items-per-run: 25
  max-discovery-items-per-run: 200
  max-discovery-pages-per-run: 10
  opt-out-labels: [no-campaign, no-bot]
  max-project-updates-per-run: 10
  max-comments-per-run: 10
---
```

## Campaign Compilation Process

### 1. Detection During Compile

**Implementation**: `pkg/cli/compile_workflow_processor.go:processCampaignSpec()`

During `gh aw compile`, the system:

1. Scans `.github/workflows/` for both `.md` and `.campaign.md` files
2. Detects `.campaign.md` suffix to trigger campaign processing
3. Loads and validates the campaign spec
4. Generates an orchestrator workflow if the spec has meaningful details

**Meaningful details check** (`pkg/campaign/orchestrator.go:BuildOrchestrator()`):
- Must have at least one of: workflows, memory paths, metrics glob, cursor glob, project URL, governance, or KPIs
- Returns `nil` if campaign has no actionable configuration
- This prevents empty orchestrators from being generated

### 2. Orchestrator Generation

**Implementation**: `pkg/campaign/orchestrator.go:BuildOrchestrator()`

The orchestrator generator creates a `workflow.WorkflowData` struct containing:

#### A. Discovery Precomputation Steps

**Function**: `buildDiscoverySteps()`

When a campaign has workflows or a tracker label, the generator adds discovery steps:

```yaml
steps:
  - name: Create workspace directory
    run: mkdir -p ./.gh-aw

  - name: Run campaign discovery precomputation
    id: discovery
    uses: actions/github-script@v8.0.0
    env:
      GH_AW_CAMPAIGN_ID: security-q1-2025
      GH_AW_WORKFLOWS: "vulnerability-scanner,dependency-updater"
      GH_AW_TRACKER_LABEL: campaign:security-q1-2025
      GH_AW_PROJECT_URL: https://github.com/orgs/ORG/projects/1
      GH_AW_MAX_DISCOVERY_ITEMS: 200
      GH_AW_MAX_DISCOVERY_PAGES: 10
      GH_AW_CURSOR_PATH: /tmp/gh-aw/repo-memory/campaigns/security-q1-2025/cursor.json
    with:
      github-token: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
      script: |
        const { setupGlobals } = require('/opt/gh-aw/actions/setup_globals.cjs');
        setupGlobals(core, github, context, exec, io);
        const { main } = require('/opt/gh-aw/actions/campaign_discovery.cjs');
        await main();
```

**Discovery script location**: The script is loaded from `/opt/gh-aw/actions/campaign_discovery.cjs`, which is copied during the `actions/setup` step.

#### B. Workflow Metadata

```go
data := &workflow.WorkflowData{
    Name:        spec.Name,
    Description: spec.Description,
    On:          "on:\n  schedule:\n    - cron: \"0 18 * * *\"\n  workflow_dispatch:\n",
    Concurrency: fmt.Sprintf("concurrency:\n  group: \"campaign-%s-orchestrator-${{ github.ref }}\"\n  cancel-in-progress: false", spec.ID),
    RunsOn:      "runs-on: ubuntu-latest",
    Roles:       []string{"admin", "maintainer", "write"},
}
```

#### C. Tools Configuration

```go
Tools: map[string]any{
    "github": map[string]any{
        "toolsets": []any{"default", "actions", "code_security"},
    },
    "repo-memory": []any{
        map[string]any{
            "id":          "campaigns",
            "branch-name": "memory/campaigns",
            "file-glob":   extractFileGlobPatterns(spec),
            "campaign-id": spec.ID,
        },
    },
    "bash": []any{"*"},
    "edit": nil,
}
```

#### D. Safe Outputs Configuration

```go
safeOutputs := &workflow.SafeOutputsConfig{
    CreateIssues:             &workflow.CreateIssuesConfig{Max: 1},
    AddComments:              &workflow.AddCommentsConfig{Max: maxComments},
    UpdateProjects:           &workflow.UpdateProjectConfig{Max: maxProjectUpdates},
    CreateProjectStatusUpdates: &workflow.CreateProjectStatusUpdateConfig{Max: 1},
}
```

Custom GitHub tokens for Projects v2 operations:
- If `spec.ProjectGitHubToken` is set, it's passed to `UpdateProjects` and `CreateProjectStatusUpdates`
- Allows using a different token with appropriate project permissions

#### E. Prompt Section

The orchestrator includes detailed instructions for the AI agent:

```go
markdownBuilder.WriteString("# Campaign Orchestrator\n\n")
// Campaign details: objective, KPIs, workflows, memory paths, etc.

orchestratorInstructions := RenderOrchestratorInstructions(promptData)
projectInstructions := RenderProjectUpdateInstructions(promptData)
closingInstructions := RenderClosingInstructions()
```

### 3. Markdown Generation

**Implementation**: `pkg/cli/compile_orchestrator.go:renderGeneratedCampaignOrchestratorMarkdown()`

The orchestrator is rendered as a markdown file:

```
<campaign-id>.campaign.g.md
```

**Important**: This `.campaign.g.md` file is a **debug artifact**:
- Generated locally during compilation
- Helps users understand the orchestrator structure
- **NOT committed to git** (excluded via `.gitignore`)
- Can be reviewed locally to see generated workflow structure

**Compiled output**: Only the `.campaign.lock.yml` file is committed to version control.

### 4. Lock File Naming

**Implementation**: `pkg/stringutil/identifiers.go:CampaignOrchestratorToLockFile()`

Campaign orchestrators follow a special naming convention:

```
example.campaign.g.md   →   example.campaign.lock.yml
```

**Not**: `example.campaign.g.lock.yml` (the `.g` suffix is removed)

This ensures the lock file name matches the campaign spec name pattern.

## Discovery Script Architecture

### Script Location

**Source**: `actions/setup/js/campaign_discovery.cjs`

**Runtime location**: `/opt/gh-aw/actions/campaign_discovery.cjs`

The discovery script is copied to `/opt/gh-aw/actions/` during the `actions/setup` action, which runs before the agent job.

### Discovery Flow

**Implementation**: `actions/setup/js/campaign_discovery.cjs:main()`

1. **Read configuration from environment variables**:
   - `GH_AW_CAMPAIGN_ID` - Campaign identifier
   - `GH_AW_WORKFLOWS` - Comma-separated list of workflow IDs (tracker-ids)
   - `GH_AW_TRACKER_LABEL` - Optional label for discovery
   - `GH_AW_MAX_DISCOVERY_ITEMS` - Budget for items to discover (default: 100)
   - `GH_AW_MAX_DISCOVERY_PAGES` - Budget for API pages to fetch (default: 10)
   - `GH_AW_CURSOR_PATH` - Path to cursor file for pagination
   - `GH_AW_PROJECT_URL` - Project URL for reference

2. **Load cursor from repo-memory** (if configured):
   ```javascript
   function loadCursor(cursorPath) {
     if (fs.existsSync(cursorPath)) {
       const content = fs.readFileSync(cursorPath, "utf8");
       return JSON.parse(content);
     }
     return null;
   }
   ```

3. **Search for items by tracker-id**:
   ```javascript
   // For each workflow in spec.workflows:
   const searchQuery = `"tracker-id: ${trackerId}" type:issue`;
   const response = await octokit.rest.search.issuesAndPullRequests({
     q: searchQuery,
     per_page: 100,
     page: page,
     sort: "updated",
     order: "asc",  // Stable ordering
   });
   ```

4. **Search for items by tracker label** (if configured):
   ```javascript
   const searchQuery = `label:"${label}"`;
   const response = await octokit.rest.search.issuesAndPullRequests({
     q: searchQuery,
     per_page: 100,
     page: page,
     sort: "updated",
     order: "asc",
   });
   ```

5. **Normalize discovered items**:
   ```javascript
   function normalizeItem(item, contentType) {
     return {
       url: item.html_url || item.url,
       content_type: contentType,  // "issue" or "pull_request"
       number: item.number,
       repo: item.repository?.full_name || "",
       created_at: item.created_at,
       updated_at: item.updated_at,
       state: item.state,
       title: item.title,
       closed_at: item.closed_at,
       merged_at: item.merged_at,
     };
   }
   ```

6. **Deduplicate items** (when using both tracker-id and tracker-label):
   ```javascript
   const existingUrls = new Set(allItems.map(i => i.url));
   for (const item of result.items) {
     if (!existingUrls.has(item.url)) {
       allItems.push(item);
     }
   }
   ```

7. **Sort for stable ordering**:
   ```javascript
   allItems.sort((a, b) => {
     if (a.updated_at !== b.updated_at) {
       return a.updated_at.localeCompare(b.updated_at);
     }
     return a.number - b.number;
   });
   ```

8. **Calculate summary counts**:
   ```javascript
   const needsAddCount = allItems.filter(i => i.state === "open").length;
   const needsUpdateCount = allItems.filter(i => i.state === "closed" || i.merged_at).length;
   ```

9. **Write manifest to `./.gh-aw/campaign.discovery.json`**:
   ```json
   {
     "schema_version": "v1",
     "campaign_id": "security-q1-2025",
     "generated_at": "2025-01-08T12:00:00.000Z",
     "project_url": "https://github.com/orgs/ORG/projects/1",
     "discovery": {
       "total_items": 42,
       "items_scanned": 100,
       "pages_scanned": 2,
       "max_items_budget": 200,
       "max_pages_budget": 10,
       "cursor": { "page": 3, "trackerId": "vulnerability-scanner" }
     },
     "summary": {
       "needs_add_count": 25,
       "needs_update_count": 17,
       "open_count": 25,
       "closed_count": 10,
       "merged_count": 7
     },
     "items": [
       {
         "url": "https://github.com/org/repo/issues/123",
         "content_type": "issue",
         "number": 123,
         "repo": "org/repo",
         "created_at": "2025-01-01T00:00:00Z",
         "updated_at": "2025-01-07T12:00:00Z",
         "state": "open",
         "title": "Upgrade dependency X"
       }
     ]
   }
   ```

10. **Save cursor to repo-memory** (for next run):
    ```javascript
    function saveCursor(cursorPath, cursor) {
      const dir = path.dirname(cursorPath);
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }
      fs.writeFileSync(cursorPath, JSON.stringify(cursor, null, 2));
    }
    ```

### Pagination Budgets

The discovery system enforces strict pagination budgets to prevent unbounded API usage:

- **Max items per run** (`governance.max-discovery-items-per-run`): Default 100, configurable
- **Max pages per run** (`governance.max-discovery-pages-per-run`): Default 10, configurable

When budgets are reached:
```javascript
if (itemsScanned >= maxItems || pagesScanned >= maxPages) {
  core.warning(`Reached discovery budget limits. Stopping discovery.`);
  break;
}
```

### Cursor Persistence

The cursor enables incremental discovery across runs:

**Cursor format**:
```json
{
  "page": 3,
  "trackerId": "vulnerability-scanner"
}
```

**Storage location**: Configured via `spec.CursorGlob`, typically:
```
memory/campaigns/<campaign-id>/cursor.json
```

**How it works**:
1. Discovery loads cursor from repo-memory
2. Continues from saved page number
3. Updates cursor after each workflow/label search
4. Saves updated cursor back to repo-memory
5. Next run picks up where previous run left off

### Campaign Item Protection

Campaign items are protected from being picked up by other workflows to prevent conflicts and ensure proper campaign orchestration.

**Protection Mechanism**:
- Items with `campaign:*` labels are automatically excluded from non-campaign workflows
- The `update-project` safe output automatically applies `campaign:<id>` labels when adding items to campaign projects
- Workflows like `issue-monster` check for campaign labels and skip those issues
- Additional opt-out labels (`no-bot`, `no-campaign`) provide manual protection

**Label Format**:
```
campaign:my-campaign-id
campaign:security-q1-2025
campaign:docs-quality-maintenance-project73
```

**How it's enforced**:

1. **Automatic labeling**: When campaign orchestrators add items to projects, they apply the `campaign:<id>` label
2. **Workflow filtering**: Other workflows (like issue-monster) filter out issues with `campaign:` prefix
3. **Opt-out labels**: Items with `no-bot` or `no-campaign` labels are also excluded

**Example filtering logic** (from issue-monster workflow):
```javascript
// Exclude issues with campaign labels (campaign:*)
// Campaign items are managed by campaign orchestrators
if (issueLabels.some(label => label.startsWith('campaign:'))) {
  core.info(`Skipping #${issue.number}: has campaign label (managed by campaign orchestrator)`);
  return false;
}
```

**Governance Configuration**:
```yaml
governance:
  opt-out-labels: ["no-campaign", "no-bot"]
```

This ensures that campaign items remain under the control of their respective campaign orchestrators and aren't interfered with by other automated workflows.

## For Third-Party Users

### Using gh-aw Compiler Outside This Repository

**Yes, it works!** The campaign system is designed to work in any repository with gh-aw installed.

#### Prerequisites

```bash
# Install gh-aw CLI
gh extension install githubnext/gh-aw

# Or use local binary
./gh-aw --help
```

#### Creating a Campaign

1. **Create campaign spec** in your repository:
   ```bash
   mkdir -p .github/workflows
   gh aw campaign new my-campaign
   ```

2. **Edit the spec** (`.github/workflows/my-campaign.campaign.md`):
   ```yaml
   ---
   id: my-campaign
   name: My Campaign
   version: v1
   project-url: https://github.com/orgs/ORG/projects/1
   tracker-label: campaign:my-campaign
   workflows:
     - my-worker-workflow
   memory-paths:
     - memory/campaigns/my-campaign/**
   ---
   
   # Campaign description goes here
   ```

3. **Compile the campaign**:
   ```bash
   gh aw compile
   ```

   This generates:
   - `.github/workflows/my-campaign.campaign.g.md` (local debug artifact)
   - `.github/workflows/my-campaign.campaign.lock.yml` (committed)

4. **Commit and push**:
   ```bash
   git add .github/workflows/my-campaign.campaign.md
   git add .github/workflows/my-campaign.campaign.lock.yml
   git commit -m "Add my-campaign"
   git push
   ```

5. **Run the orchestrator** from GitHub Actions tab

#### What Gets Executed

When the orchestrator runs:

1. **Setup Actions** - Copies JavaScript files to `/opt/gh-aw/actions/`:
   - Source: `actions/setup/js/campaign_discovery.cjs` (from gh-aw repository)
   - Runtime: `/opt/gh-aw/actions/campaign_discovery.cjs`

2. **Discovery Step** - Executes discovery precomputation:
   - Uses `actions/github-script@v8.0.0`
   - Calls `require('/opt/gh-aw/actions/campaign_discovery.cjs')`
   - Generates `./.gh-aw/campaign.discovery.json`

3. **Agent Job** - AI agent processes the manifest:
   - Reads `./.gh-aw/campaign.discovery.json`
   - Updates GitHub Project board via safe-outputs
   - Uses repo-memory for state persistence

#### Required Files

**In the gh-aw repository** (automatically included):
- `actions/setup/` - Setup action that copies JavaScript files
- `actions/setup/js/campaign_discovery.cjs` - Discovery script
- `actions/setup/js/setup_globals.cjs` - Global utilities

**In your repository** (you create):
- `.github/workflows/<id>.campaign.md` - Campaign spec
- `.github/workflows/<id>.campaign.lock.yml` - Compiled workflow (generated)

#### How the Compiler Finds Scripts

The discovery script is **not** included in the compiled `.lock.yml` file. Instead:

1. The compiled workflow includes an `actions/setup` step
2. `actions/setup` copies files from its repository to `/opt/gh-aw/actions/`
3. The discovery step uses `require('/opt/gh-aw/actions/campaign_discovery.cjs')`
4. This works because the path is available at runtime via the setup action

**Key insight**: The setup action is a composite action that copies JavaScript files to a runtime location. This allows campaigns in any repository to use the discovery script without duplicating it.

## Cross-References

### Related Documentation

**User-facing guides** (in `docs/src/content/docs/`):
- [Getting Started with Campaigns](/docs/src/content/docs/guides/campaigns/getting-started.md) - Quick start guide
- [Campaign Specs](/docs/src/content/docs/guides/campaigns/specs.md) - YAML frontmatter reference
- [Campaign CLI Commands](/docs/src/content/docs/guides/campaigns/cli-commands.md) - CLI usage examples
- [Project Management](/docs/src/content/docs/guides/campaigns/project-management.md) - GitHub Projects integration

**Architecture specs** (in `specs/`):
- [Repo-Memory](./repo-memory.md) - Persistent state storage for campaigns
- [Safe Output Messages](./safe-output-messages.md) - GitHub API operations (create-issue, update-project)
- [Code Organization](./code-organization.md) - Campaign package structure rationale

### Code References

**Campaign package** (`pkg/campaign/`):
- `spec.go` - Data structures (CampaignSpec, CampaignKPI, CampaignGovernancePolicy)
- `loader.go` - Discovery and loading (LoadSpecs, FilterSpecs, CreateSpecSkeleton)
- `orchestrator.go` - Orchestrator generation (BuildOrchestrator, buildDiscoverySteps)
- `validation.go` - Spec validation (ValidateSpec)
- `command.go` - CLI commands (campaign, campaign status, campaign new, campaign validate)

**CLI package** (`pkg/cli/`):
- `compile_workflow_processor.go` - Workflow processing (processCampaignSpec)
- `compile_orchestrator.go` - Orchestrator rendering (renderGeneratedCampaignOrchestratorMarkdown)
- `compile_helpers.go` - Utility functions

**Actions** (`actions/setup/js/`):
- `campaign_discovery.cjs` - Discovery precomputation script
- `setup_globals.cjs` - Global utilities for GitHub Actions scripts

### Key Workflows

**Example campaigns** (in `.github/workflows/`):
- Look for `*.campaign.md` files in the repository root
- Compiled to `*.campaign.lock.yml` files

## Design Decisions

### Why Separate Discovery Step?

**Problem**: AI agents performing GitHub-wide discovery during Phase 1 is:
- Non-deterministic (different results on each run)
- Expensive (many API calls)
- Slow (sequential search)

**Solution**: Precomputation step that runs before the agent:
- Deterministic output (stable manifest)
- Enforced budgets (max items, max pages)
- Fast (parallel search possible)
- Cacheable (manifest can be reused)

### Why `.campaign.g.md` is Not Committed

**Rationale**:
- It's a generated artifact, not source code
- Users edit `.campaign.md`, not `.campaign.g.md`
- The `.lock.yml` file is the authoritative compiled output
- Keeping `.g.md` local aids debugging without cluttering git history

**Benefits**:
- Cleaner git history
- No merge conflicts on generated files
- Users can regenerate anytime with `gh aw compile`
- `.lock.yml` provides reproducible execution

### Why Cursor is in Repo-Memory

**Rationale**:
- Campaigns need durable state across runs
- Git branches provide versioned, auditable history
- Repo-memory integrates with existing GitHub workflows

**Alternatives considered**:
- Environment variables (lost between runs)
- Workflow artifacts (expire after 90 days)
- External database (requires additional infrastructure)

### Why Campaign-Specific Lock File Naming

**Problem**: Standard naming would produce:
```
example.campaign.g.md  →  example.campaign.g.lock.yml
```

This is verbose and inconsistent with the spec file name.

**Solution**: Special handling in `stringutil.CampaignOrchestratorToLockFile()`:
```
example.campaign.g.md  →  example.campaign.lock.yml
```

This keeps lock files aligned with spec files:
```
example.campaign.md        (spec)
example.campaign.lock.yml  (compiled)
```

## Debugging

### Enable Debug Logging

```bash
DEBUG=campaign:*,cli:* gh aw compile
```

### Check Generated Orchestrator

```bash
# Review local debug artifact
cat .github/workflows/<campaign-id>.campaign.g.md

# Review compiled workflow
cat .github/workflows/<campaign-id>.campaign.lock.yml
```

### Inspect Discovery Manifest

After running the orchestrator:

```bash
# Download workflow artifacts
gh run download <run-id>

# Check discovery manifest
cat .gh-aw/campaign.discovery.json
```

### Validate Campaign Spec

```bash
gh aw campaign validate
gh aw campaign validate my-campaign
gh aw campaign validate --json
```

## Future Enhancements

### Planned Improvements

1. **Multi-repository discovery**: Search across organization repositories
2. **Advanced filtering**: Filter items by milestone, assignee, or custom fields
3. **Discovery caching**: Cache discovery results to reduce API calls
4. **Incremental updates**: Only update changed items in project board
5. **Workflow templates**: Pre-built campaign templates for common scenarios

### Extension Points

1. **Custom discovery scripts**: Allow campaigns to provide custom discovery logic
2. **Discovery plugins**: Plugin system for discovery sources (Jira, Linear, etc.)
3. **Campaign hierarchies**: Parent/child campaigns with rollup metrics
4. **Cross-campaign dependencies**: Express dependencies between campaigns

---

**Last Updated**: 2025-01-08

**Related Issues**: #1234 (Campaign Architecture), #5678 (Discovery Optimization)
