# Investigation: Campaign Worker Workflow Labels

**Date**: 2026-01-23  
**Author**: GitHub Copilot  
**Purpose**: Document how campaigns currently set worker workflow labels

---

## Executive Summary

This investigation examines how GitHub Agentic Workflows campaigns manage labeling for worker-created outputs (issues and pull requests). The system uses two distinct labels:

1. **`agentic-campaign`** - Generic marker for all campaign-related work
2. **`z_campaign_<campaign-id>`** - Campaign-specific identifier for discovery

Both labels are automatically applied to worker outputs through the **safe-outputs** configuration in worker workflow frontmatter.

---

## Label Types and Purpose

### 1. Generic Campaign Label: `agentic-campaign`

**Purpose**: Mark content as part of ANY campaign

**Format**: Fixed string literal `"agentic-campaign"`

**Benefits**:
- Prevents other workflows from processing campaign items
- Enables campaign-wide queries and filters
- Provides a consistent marker across all campaigns

**Example Usage**:
```yaml
safe-outputs:
  create-pull-request:
    labels: [security, automated-fix, agentic-campaign, z_campaign_security-alert-burndown]
```

### 2. Campaign-Specific Label: `z_campaign_<campaign-id>`

**Purpose**: Enable precise discovery of items belonging to a specific campaign

**Format**: `z_campaign_<campaign-id>` where:
- Prefix `z_campaign_` is constant (defined in `pkg/constants/constants.go`)
- Campaign ID is lowercase, hyphen-separated
- Spaces and underscores are converted to hyphens

**Label Generation Examples**:
```
Campaign ID                    → Campaign Label
----------------------------- → ---------------------------------
security-q1-2025              → z_campaign_security-q1-2025
Security Q1 2025              → z_campaign_security-q1-2025
dependency_updates            → z_campaign_dependency-updates
```

**Implementation Locations**:
- Go: `pkg/stringutil/identifiers.go:FormatCampaignLabel()`
- JavaScript: `actions/setup/js/campaign_discovery.cjs` (line 329)
- JavaScript: `actions/setup/js/safe_output_handler_manager.cjs:formatCampaignLabel()`

**Why the `z_` prefix?**
The prefix ensures campaign labels sort last in label lists, improving visibility when viewing issues/PRs with multiple labels.

---

## Label Application Mechanisms

### Mechanism 1: Explicit Configuration in Worker Frontmatter

**Primary Method**: Workers explicitly configure labels in their safe-outputs configuration.

**Example: Code Scanning Fixer**
```yaml
# File: .github/workflows/code-scanning-fixer.md
safe-outputs:
  create-pull-request:
    title-prefix: "[code-scanning-fix] "
    labels: [security, automated-fix, agentic-campaign, z_campaign_security-alert-burndown]
    reviewers: [copilot]
```

**Example: Secret Scanning Triage**
```yaml
# File: .github/workflows/secret-scanning-triage.md
safe-outputs:
  create-issue:
    title-prefix: "[secret-triage] "
    labels: [security, secret-scanning, triage, agentic-campaign, z_campaign_security-alert-burndown]
    max: 1
  create-pull-request:
    title-prefix: "[secret-removal] "
    labels: [security, secret-scanning, automated-fix, agentic-campaign, z_campaign_security-alert-burndown]
    reviewers: [copilot]
```

### Mechanism 2: Safe Outputs Handler Manager

**Location**: `actions/setup/js/safe_output_handler_manager.cjs`

**Function**: `formatCampaignLabel(campaignId)`

The safe outputs handler automatically applies campaign labels when:
- A worker workflow has `repo-memory` tool configured with `campaign-id` property
- Campaign labels are added to the labels array before creating issues/PRs

**Code Implementation**:
```javascript
/**
 * Normalize campaign IDs to the same label format used by campaign discovery.
 * Mirrors actions/setup/js/campaign_discovery.cjs.
 * @param {string} campaignId
 * @returns {string}
 */
function formatCampaignLabel(campaignId) {
  return `z_campaign_${String(campaignId)
    .toLowerCase()
    .replace(/[_\s]+/g, "-")}`;
}
```

### Mechanism 3: Discovery Precomputation

**Location**: `actions/setup/js/campaign_discovery.cjs`

**Discovery Strategy**: The discovery script searches for worker outputs using the campaign-specific label as the primary discovery mechanism.

**Implementation**:
```javascript
// Generate campaign-specific label
const campaignLabel = `z_campaign_${campaignId.toLowerCase().replace(/[_\s]+/g, "-")}`;

// Primary discovery: Search by campaign-specific label (most reliable)
core.info(`Primary discovery: Searching by campaign-specific label: ${campaignLabel}`);
const labelResult = await searchByLabel(octokit, campaignLabel, repos, orgs, maxDiscoveryItems, maxDiscoveryPages, cursor);
```

**Search Query**:
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

---

## Campaign Orchestrator Label Requirements

Campaign orchestrators document the labeling contract in their generated instructions.

**Location**: `.github/aw/orchestrate-agentic-campaign.md` (lines 80-106)

**Required Labels for Worker Outputs**:

1. **`agentic-campaign`** - Generic label marking content as part of ANY campaign
   - Prevents other workflows from processing campaign items
   - Enables campaign-wide queries and filters

2. **`z_campaign_{{.CampaignID}}`** - Campaign-specific label
   - Enables precise discovery of items belonging to THIS campaign
   - Format: `z_campaign_<campaign-id>` (lowercase, hyphen-separated)
   - Example: `z_campaign_security-q1-2025`

**Worker Responsibilities**:
- Workers creating issues/PRs as campaign output MUST add both labels
- Workers SHOULD use `create-issue` or `create-pr` safe outputs with labels configuration
- If workers cannot add labels automatically, campaign orchestrator will attempt to add them during discovery

**Non-Campaign Workflow Responsibilities**:
- Workflows triggered by issues/PRs SHOULD skip items with `agentic-campaign` label
- Use `skip-if-match` configuration to filter out campaign items

---

## Data Flow Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Campaign Spec (.campaign.md)                │
│  - Defines campaign ID                                           │
│  - Associates worker workflows                                   │
│  - Configures tracker-label (optional)                           │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          │ gh aw compile
                          ↓
┌─────────────────────────────────────────────────────────────────┐
│           Orchestrator Workflow (.campaign.lock.yml)            │
│  - Includes discovery precomputation step                        │
│  - Searches by z_campaign_<id> label                            │
│  - Generates manifest of discovered items                        │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          │ workflow_dispatch
                          ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Worker Workflow (.md)                         │
│  - Configured with safe-outputs labels                           │
│  - Creates issues/PRs with:                                      │
│    • agentic-campaign                                           │
│    • z_campaign_<campaign-id>                                   │
│    • Worker-specific labels (security, automated-fix, etc.)     │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          │ creates issue/PR
                          ↓
┌─────────────────────────────────────────────────────────────────┐
│                   GitHub Issue/Pull Request                      │
│  Labels: [agentic-campaign, z_campaign_security-alert-burndown, │
│           security, automated-fix]                               │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          │ next orchestrator run
                          ↓
┌─────────────────────────────────────────────────────────────────┐
│              Discovery Script (campaign_discovery.cjs)           │
│  - Searches GitHub for label:z_campaign_<id>                    │
│  - Returns manifest of discovered items                          │
│  - Updates Project board                                         │
└─────────────────────────────────────────────────────────────────┘
```

---

## Configuration Examples

### Worker Configuration (Frontmatter)

**Basic Worker with Campaign Labels**:
```yaml
---
name: Security Alert Fixer
description: Fix security alerts automatically
on:
  workflow_dispatch:
engine: copilot
tools:
  github:
    toolsets: [default, code_security]
  repo-memory:
    - id: campaigns
      branch-name: memory/campaigns
      file-glob: [security-alert-burndown/**]
      campaign-id: security-alert-burndown
safe-outputs:
  create-pull-request:
    title-prefix: "[security-fix] "
    labels: [security, automated-fix, agentic-campaign, z_campaign_security-alert-burndown]
    reviewers: [copilot]
    max: 1
---
```

### Campaign Spec Configuration

**Campaign Definition**:
```yaml
---
id: security-alert-burndown
name: Security Alert Burndown Q1 2025
version: v1
state: active

project-url: https://github.com/orgs/githubnext/projects/123
tracker-label: z_campaign_security-alert-burndown

workflows:
  - code-scanning-fixer
  - secret-scanning-triage
  - dependabot-bundler

governance:
  max-new-items-per-run: 25
  max-discovery-items-per-run: 200
  max-discovery-pages-per-run: 10
  opt-out-labels: [no-campaign, no-bot]
---
```

---

## Safe Outputs Configuration Types

### CreateIssuesConfig

**File**: `pkg/workflow/create_issue.go`

**Label Fields**:
```go
type CreateIssuesConfig struct {
    BaseSafeOutputConfig `yaml:",inline"`
    TitlePrefix          string   `yaml:"title-prefix,omitempty"`
    Labels               []string `yaml:"labels,omitempty"`            // Labels to apply to created issues
    AllowedLabels        []string `yaml:"allowed-labels,omitempty"`    // Optional whitelist of allowed labels
    Assignees            []string `yaml:"assignees,omitempty"`
    // ... other fields
}
```

**Environment Variables**:
- `GH_AW_ISSUE_LABELS` - JSON array of labels to apply
- `GH_AW_ISSUE_ALLOWED_LABELS` - JSON array of allowed labels (optional whitelist)

### CreatePullRequestsConfig

**File**: `pkg/workflow/create_pull_request.go`

**Label Fields**:
```go
type CreatePullRequestsConfig struct {
    BaseSafeOutputConfig `yaml:",inline"`
    TitlePrefix          string   `yaml:"title-prefix,omitempty"`
    Labels               []string `yaml:"labels,omitempty"`            // Labels to apply to created PRs
    AllowedLabels        []string `yaml:"allowed-labels,omitempty"`    // Optional whitelist of allowed labels
    Reviewers            []string `yaml:"reviewers,omitempty"`
    // ... other fields
}
```

**Environment Variables**:
- `GH_AW_PR_LABELS` - JSON array of labels to apply
- `GH_AW_PR_ALLOWED_LABELS` - JSON array of allowed labels (optional whitelist)

---

## Label Discovery Strategy

### Primary Discovery: Campaign-Specific Label

**Method**: Search GitHub for `label:z_campaign_<campaign-id>`

**Advantages**:
- Most reliable discovery mechanism
- Works across repositories when `discovery-repos` or `discovery-orgs` is configured
- Handles multi-repo campaigns

**Example Search Query**:
```
label:"z_campaign_security-alert-burndown"
```

### Secondary Discovery: Tracker ID (Deprecated)

**Method**: Search for `tracker-id: <workflow-id>` in issue/PR body

**Status**: Legacy approach, less reliable than label-based discovery

**Reason for Deprecation**: 
- Requires parsing issue/PR body text
- Fragile to formatting changes
- Not indexed by GitHub search

---

## Protection from Other Workflows

Campaign items are protected from being picked up by other workflows to prevent conflicts.

### Protection Mechanism

**Label-Based Filtering**:
```yaml
on:
  issues:
    types: [opened, labeled]
    skip-if-match:
      query: "label:agentic-campaign"
      max: 0  # Skip if ANY campaign items match
```

**Example from issue-monster**:
```javascript
// Exclude issues with campaign labels (campaign:*)
// Campaign items are managed by campaign orchestrators
if (issueLabels.some(label => label.startsWith('campaign:'))) {
  core.info(`Skipping #${issue.number}: has campaign label (managed by campaign orchestrator)`);
  return false;
}
```

### Opt-Out Labels

Campaign governance can specify additional opt-out labels:

```yaml
governance:
  opt-out-labels: ["no-campaign", "no-bot"]
```

---

## Code References

### Go Code

**Label Format Constant**:
```go
// File: pkg/constants/constants.go
const CampaignLabelPrefix = "z_campaign_"
```

**Label Formatting Function**:
```go
// File: pkg/stringutil/identifiers.go
func FormatCampaignLabel(campaignID string) string {
    sanitized := strings.ToLower(campaignID)
    sanitized = strings.ReplaceAll(sanitized, " ", "-")
    sanitized = strings.ReplaceAll(sanitized, "_", "-")
    return constants.CampaignLabelPrefix + sanitized
}
```

**Environment Variable Builder**:
```go
// File: pkg/workflow/safe_outputs_env.go
func buildLabelsEnvVar(envVarName string, labels []string) []string {
    // Converts labels array to JSON and sets environment variable
}
```

### JavaScript Code

**Discovery Script**:
```javascript
// File: actions/setup/js/campaign_discovery.cjs
const campaignLabel = `z_campaign_${campaignId.toLowerCase().replace(/[_\s]+/g, "-")}`;
```

**Safe Outputs Handler**:
```javascript
// File: actions/setup/js/safe_output_handler_manager.cjs
function formatCampaignLabel(campaignId) {
  return `z_campaign_${String(campaignId)
    .toLowerCase()
    .replace(/[_\s]+/g, "-")}`;
}
```

---

## Testing

### Unit Tests

**Go Tests**:
```go
// File: pkg/stringutil/identifiers_test.go
func TestFormatCampaignLabel(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"security-q1-2025", "z_campaign_security-q1-2025"},
        {"Security Q1 2025", "z_campaign_security-q1-2025"},
        {"dependency_updates", "z_campaign_dependency-updates"},
    }
    // ...
}
```

**JavaScript Tests**:
```javascript
// File: actions/setup/js/safe_output_handler_manager.test.cjs
expect(message.labels).toContain("z_campaign_security-alert-burndown");
```

---

## Best Practices

### For Worker Workflows

1. **Always Include Both Labels**:
   ```yaml
   labels: [agentic-campaign, z_campaign_<campaign-id>, <worker-specific-labels>]
   ```

2. **Use Consistent Label Format**:
   - Campaign ID in lowercase
   - Spaces/underscores → hyphens
   - Prefix: `z_campaign_`

3. **Configure Repo-Memory with Campaign ID**:
   ```yaml
   tools:
     repo-memory:
       - id: campaigns
         campaign-id: <campaign-id>
   ```

4. **Use Safe Outputs Configuration**:
   - Prefer explicit label configuration in frontmatter
   - Don't rely on manual label application in workflow logic

### For Campaign Orchestrators

1. **Specify Tracker Label in Spec**:
   ```yaml
   tracker-label: z_campaign_<campaign-id>
   ```

2. **Configure Discovery Settings**:
   ```yaml
   governance:
     max-discovery-items-per-run: 200
     max-discovery-pages-per-run: 10
   ```

3. **Document Worker Requirements**:
   - Include labeling requirements in orchestrator instructions
   - Specify both `agentic-campaign` and campaign-specific labels

### For Non-Campaign Workflows

1. **Filter Out Campaign Items**:
   ```yaml
   on:
     issues:
       skip-if-match: "label:agentic-campaign"
   ```

2. **Check Labels Before Processing**:
   ```javascript
   if (labels.some(l => l.startsWith('campaign:'))) {
     // Skip campaign items
     return;
   }
   ```

---

## Common Pitfalls

### 1. Missing Campaign-Specific Label

**Problem**: Worker only applies `agentic-campaign` label, not `z_campaign_<id>`

**Impact**: Campaign orchestrator cannot discover worker outputs

**Solution**: Always include both labels in safe-outputs configuration

### 2. Inconsistent Label Format

**Problem**: Manual label application with incorrect format (e.g., `campaign_security-q1-2025`)

**Impact**: Discovery fails to find items

**Solution**: Use `FormatCampaignLabel()` helper or `formatCampaignLabel()` JavaScript function

### 3. Hardcoded Campaign ID

**Problem**: Worker hardcodes campaign label instead of using dynamic configuration

**Impact**: Worker cannot be reused across campaigns

**Solution**: Configure campaign ID via `repo-memory` tool or workflow inputs

### 4. Missing Label Validation

**Problem**: Worker doesn't validate that labels were applied successfully

**Impact**: Silent failures in label application

**Solution**: Verify labels after issue/PR creation (safe outputs handler does this automatically)

---

## Future Enhancements

### Potential Improvements

1. **Automatic Label Injection**:
   - Compiler could automatically inject campaign labels based on campaign spec
   - Reduces manual configuration in worker workflows

2. **Label Validation at Compile Time**:
   - Validate that worker workflows include required campaign labels
   - Fail compilation if labels are missing

3. **Dynamic Label Configuration**:
   - Support environment variable expansion in label arrays
   - Enable campaign-id to be passed as workflow input

4. **Cross-Campaign Labeling**:
   - Support workers participating in multiple campaigns
   - Apply multiple campaign-specific labels

5. **Label Namespacing**:
   - Introduce org-level label prefixes (e.g., `z_campaign_org_<org-name>_<campaign-id>`)
   - Enable cross-org campaign tracking

---

## Conclusion

The campaign labeling system uses a two-label approach:

1. **`agentic-campaign`** - Generic marker for all campaign work
2. **`z_campaign_<campaign-id>`** - Campaign-specific identifier

Labels are applied through safe-outputs configuration in worker frontmatter, ensuring consistent and discoverable outputs. The discovery system relies primarily on the campaign-specific label to find and track worker outputs across repositories.

**Key Takeaways**:
- Workers MUST apply both labels to all campaign outputs
- Labels follow a strict format: lowercase, hyphen-separated, `z_campaign_` prefix
- Discovery uses label-based search as the primary mechanism
- Non-campaign workflows should filter out `agentic-campaign` labeled items

---

## Related Documentation

- [specs/campaigns-files.md](/home/runner/work/gh-aw/gh-aw/specs/campaigns-files.md) - Campaign architecture and file locations
- [docs/campaign-workers.md](/home/runner/work/gh-aw/gh-aw/docs/campaign-workers.md) - Worker workflow design patterns
- [pkg/campaign/spec.go](/home/runner/work/gh-aw/gh-aw/pkg/campaign/spec.go) - Campaign spec data structures
- [actions/setup/js/campaign_discovery.cjs](/home/runner/work/gh-aw/gh-aw/actions/setup/js/campaign_discovery.cjs) - Discovery implementation

---

**Last Updated**: 2026-01-23  
**Reviewed By**: GitHub Copilot  
**Status**: Investigation Complete
