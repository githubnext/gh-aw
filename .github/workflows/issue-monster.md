---
name: Issue Monster
description: The Cookie Monster of issues - assigns issues to Copilot coding agent one at a time
on:
  workflow_dispatch:
  schedule: every 30m
  skip-if-match:
    query: "is:pr is:open is:draft author:app/copilot-swe-agent"
    max: 5
  skip-if-no-match: "is:issue is:open"
  skip-if-check-failing:
    include:
      - build
      - test
      - lint-go
      - lint-js
    allow-pending: true
  permissions:
    issues: read
    pull-requests: read
  steps:
    - name: Search for candidate issues
      id: search
      uses: actions/github-script@v8
      with:
        script: |
          const { owner, repo } = context.repo;
          
          try {
            // Check for recent rate-limited PRs to avoid scheduling more work during rate limiting
            core.info('Checking for recent rate-limited PRs...');
            const rateLimitCheckDate = new Date();
            rateLimitCheckDate.setHours(rateLimitCheckDate.getHours() - 1); // Check last hour
            // Format as YYYY-MM-DDTHH:MM:SS for GitHub search API
            const rateLimitCheckISO = rateLimitCheckDate.toISOString().split('.')[0] + 'Z';
            
            const recentPRsQuery = `is:pr author:app/copilot-swe-agent created:>${rateLimitCheckISO} repo:${owner}/${repo}`;
            const recentPRsResponse = await github.rest.search.issuesAndPullRequests({
              q: recentPRsQuery,
              per_page: 10,
              sort: 'created',
              order: 'desc'
            });
            
            core.info(`Found ${recentPRsResponse.data.total_count} recent Copilot PRs to check for rate limiting`);
            
            // Check if any recent PRs have rate limit indicators
            let rateLimitDetected = false;
            for (const pr of recentPRsResponse.data.items) {
              try {
                const prTimelineQuery = `
                  query($owner: String!, $repo: String!, $number: Int!) {
                    repository(owner: $owner, name: $repo) {
                      pullRequest(number: $number) {
                        timelineItems(first: 50, itemTypes: [ISSUE_COMMENT]) {
                          nodes {
                            __typename
                            ... on IssueComment {
                              body
                              createdAt
                            }
                          }
                        }
                      }
                    }
                  }
                `;
                
                const prTimelineResult = await github.graphql(prTimelineQuery, {
                  owner,
                  repo,
                  number: pr.number
                });
                
                const comments = prTimelineResult?.repository?.pullRequest?.timelineItems?.nodes || [];
                const rateLimitPattern = /rate limit|API rate limit|secondary rate limit|abuse detection|429|too many requests/i;
                
                for (const comment of comments) {
                  if (comment.body && rateLimitPattern.test(comment.body)) {
                    core.warning(`Rate limiting detected in PR #${pr.number}: ${comment.body.substring(0, 200)}`);
                    rateLimitDetected = true;
                    break;
                  }
                }
                
                if (rateLimitDetected) break;
              } catch (error) {
                core.warning(`Could not check PR #${pr.number} for rate limiting: ${error.message}`);
              }
            }
            
            if (rateLimitDetected) {
              core.warning('🛑 Rate limiting detected in recent PRs. Skipping issue assignment to prevent further rate limit issues.');
              core.setOutput('issue_count', 0);
              core.setOutput('issue_numbers', '');
              core.setOutput('issue_list', '');
              core.setOutput('has_issues', 'false');
              return;
            }
            
            core.info('✓ No rate limiting detected. Proceeding with issue search.');
            
            // Labels that indicate an issue should NOT be auto-assigned
            const excludeLabels = [
              'wontfix',
              'duplicate',
              'invalid',
              'question',
              'discussion',
              'needs-discussion',
              'blocked',
              'on-hold',
              'waiting-for-feedback',
              'needs-more-info',
              'no-bot',
              'no-campaign'
            ];
            
            // Labels that indicate an issue is a GOOD candidate for auto-assignment
            const priorityLabels = [
              'good first issue',
              'good-first-issue',
              'bug',
              'enhancement',
              'feature',
              'documentation',
              'tech-debt',
              'refactoring',
              'performance',
              'security'
            ];
            
            // Search for open issues with "cookie" label and without excluded labels
            // The "cookie" label indicates issues that are approved work queue items from automated workflows
            const query = `is:issue is:open repo:${owner}/${repo} label:cookie -label:"${excludeLabels.join('" -label:"')}"`;
            core.info(`Searching: ${query}`);
            const response = await github.rest.search.issuesAndPullRequests({
              q: query,
              per_page: 100,
              sort: 'created',
              order: 'desc'
            });
            core.info(`Found ${response.data.total_count} total issues matching basic criteria`);
            
            // Fetch full details for each issue to get labels, assignees, sub-issues, and linked PRs
            // Track integrity-filtered issues to emit a diagnostic summary
            const integrityFilteredIssues = [];
            const issuesWithDetails = (await Promise.all(
              response.data.items.map(async (issue) => {
                // Fetch full issue details — some issues may be blocked by integrity policy
                let fullIssue;
                try {
                  fullIssue = await github.rest.issues.get({
                    owner,
                    repo,
                    issue_number: issue.number
                  });
                } catch (fetchError) {
                  // Integrity-filtered issues (403/451) or other transient errors should be
                  // skipped individually rather than failing the entire batch
                  const status = fetchError.status || fetchError.response?.status;
                  // 403 = Forbidden (integrity policy), 451 = Unavailable For Legal Reasons
                  const isIntegrityBlock = status === 403 || status === 451 ||
                    /\bintegrity\b/i.test(fetchError.message || '');
                  const errorSummary = (fetchError.message || String(fetchError)).slice(0, 120);
                  if (isIntegrityBlock) {
                    integrityFilteredIssues.push(issue.number);
                    core.warning(`⚠️ Skipping issue #${issue.number}: blocked by integrity policy (HTTP ${status || 'unknown'}): ${errorSummary}`);
                  } else {
                    core.warning(`⚠️ Skipping issue #${issue.number}: could not fetch details (HTTP ${status || 'unknown'}): ${errorSummary}`);
                  }
                  return null;
                }

                // Check if this issue has sub-issues and linked PRs using GraphQL
                let subIssuesCount = 0;
                let linkedPRs = [];
                try {
                  const issueDetailsQuery = `
                    query($owner: String!, $repo: String!, $number: Int!) {
                      repository(owner: $owner, name: $repo) {
                        issue(number: $number) {
                          subIssues {
                            totalCount
                          }
                          timelineItems(first: 100, itemTypes: [CROSS_REFERENCED_EVENT]) {
                            nodes {
                              ... on CrossReferencedEvent {
                                source {
                                  __typename
                                  ... on PullRequest {
                                    number
                                    state
                                    isDraft
                                    author {
                                      login
                                    }
                                  }
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  `;
                  const issueDetailsResult = await github.graphql(issueDetailsQuery, {
                    owner,
                    repo,
                    number: issue.number
                  });
                  
                  subIssuesCount = issueDetailsResult?.repository?.issue?.subIssues?.totalCount || 0;
                  
                  // Extract linked PRs from timeline
                  const timelineItems = issueDetailsResult?.repository?.issue?.timelineItems?.nodes || [];
                  linkedPRs = timelineItems
                    .filter(item => item?.source?.__typename === 'PullRequest')
                    .map(item => ({
                      number: item.source.number,
                      state: item.source.state,
                      isDraft: item.source.isDraft,
                      author: item.source.author?.login
                    }));
                    
                  core.info(`Issue #${issue.number} has ${linkedPRs.length} linked PR(s)`);
                } catch (error) {
                  // If GraphQL query fails, continue with defaults
                  core.warning(`Could not check details for #${issue.number}: ${error.message}`);
                }
                
                return {
                  ...fullIssue.data,
                  subIssuesCount,
                  linkedPRs
                };
              })
            )).filter(Boolean); // Remove null entries (integrity-filtered or otherwise skipped)

            // Emit diagnostic summary for integrity-filtered issues
            if (integrityFilteredIssues.length > 0) {
              core.warning(`🛡️ Integrity filter diagnostic: ${integrityFilteredIssues.length} issue(s) were skipped due to integrity policy: #${integrityFilteredIssues.join(', #')}. These issues will be excluded from this run.`);
            }
            
            // Filter and score issues
            const scoredIssues = issuesWithDetails
              .filter(issue => {
                // Exclude issues that already have assignees
                if (issue.assignees && issue.assignees.length > 0) {
                  core.info(`Skipping #${issue.number}: already has assignees`);
                  return false;
                }
                
                // Exclude issues with excluded labels (double check)
                const issueLabels = issue.labels.map(l => l.name.toLowerCase());
                if (issueLabels.some(label => excludeLabels.map(l => l.toLowerCase()).includes(label))) {
                  core.info(`Skipping #${issue.number}: has excluded label`);
                  return false;
                }
                
                // Exclude issues with campaign labels (campaign:*)
                // Campaign items are managed by campaign orchestrators
                if (issueLabels.some(label => label.startsWith('campaign:'))) {
                  core.info(`Skipping #${issue.number}: has campaign label (managed by campaign orchestrator)`);
                  return false;
                }
                
                // Exclude issues that have sub-issues (parent/organizing issues)
                if (issue.subIssuesCount > 0) {
                  core.info(`Skipping #${issue.number}: has ${issue.subIssuesCount} sub-issue(s) - parent issues are used for organizing, not tasks`);
                  return false;
                }
                
                // Exclude issues with closed PRs (treat as complete)
                const closedPRs = issue.linkedPRs?.filter(pr => pr.state === 'CLOSED' || pr.state === 'MERGED') || [];
                if (closedPRs.length > 0) {
                  core.info(`Skipping #${issue.number}: has ${closedPRs.length} closed/merged PR(s) - treating as complete`);
                  return false;
                }
                
                // Exclude issues with open PRs from Copilot coding agent
                const openCopilotPRs = issue.linkedPRs?.filter(pr => 
                  pr.state === 'OPEN' && 
                  (pr.author === 'copilot-swe-agent' || pr.author?.includes('copilot'))
                ) || [];
                if (openCopilotPRs.length > 0) {
                  core.info(`Skipping #${issue.number}: has ${openCopilotPRs.length} open PR(s) from Copilot - already being worked on`);
                  return false;
                }
                
                return true;
              })
              .map(issue => {
                const issueLabels = issue.labels.map(l => l.name.toLowerCase());
                let score = 0;
                
                // Score based on priority labels (higher score = higher priority)
                if (issueLabels.includes('good first issue') || issueLabels.includes('good-first-issue')) {
                  score += 50;
                }
                if (issueLabels.includes('bug')) {
                  score += 40;
                }
                if (issueLabels.includes('security')) {
                  score += 45;
                }
                if (issueLabels.includes('documentation')) {
                  score += 35;
                }
                if (issueLabels.includes('enhancement') || issueLabels.includes('feature')) {
                  score += 30;
                }
                if (issueLabels.includes('performance')) {
                  score += 25;
                }
                if (issueLabels.includes('tech-debt') || issueLabels.includes('refactoring')) {
                  score += 20;
                }
                
                // Bonus for issues with clear labels (any priority label)
                if (issueLabels.some(label => priorityLabels.map(l => l.toLowerCase()).includes(label))) {
                  score += 10;
                }
                
                // Age bonus: older issues get slight priority (days old / 10)
                const ageInDays = Math.floor((Date.now() - new Date(issue.created_at)) / (1000 * 60 * 60 * 24));
                score += Math.min(ageInDays / 10, 20); // Cap age bonus at 20 points
                
                return {
                  number: issue.number,
                  title: issue.title,
                  labels: issue.labels.map(l => l.name),
                  created_at: issue.created_at,
                  score
                };
              })
              .sort((a, b) => b.score - a.score); // Sort by score descending
            
            // Format output
            const issueList = scoredIssues.map(i => {
              const labelStr = i.labels.length > 0 ? ` [${i.labels.join(', ')}]` : '';
              return `#${i.number}: ${i.title}${labelStr} (score: ${i.score.toFixed(1)})`;
            }).join('\n');
            
            const issueNumbers = scoredIssues.map(i => i.number).join(',');
            
            core.info(`Total candidate issues after filtering: ${scoredIssues.length}`);
            if (scoredIssues.length > 0) {
              core.info(`Top candidates:\n${issueList.split('\n').slice(0, 10).join('\n')}`);
            }
            
            core.setOutput('issue_count', scoredIssues.length);
            core.setOutput('issue_numbers', issueNumbers);
            core.setOutput('issue_list', issueList);
            
            if (scoredIssues.length === 0) {
              core.info('🍽️ No suitable candidate issues - the plate is empty!');
              core.setOutput('has_issues', 'false');
            } else {
              core.setOutput('has_issues', 'true');
            }
          } catch (error) {
            core.error(`Error searching for issues: ${error.message}`);
            core.setOutput('issue_count', 0);
            core.setOutput('issue_numbers', '');
            core.setOutput('issue_list', '');
            core.setOutput('has_issues', 'false');
          }


permissions:
  contents: read
  issues: read
  pull-requests: read

engine:
  id: copilot
  model: gpt-5.1-codex-mini

imports:
  - shared/github-guard-policy.md
  - shared/activation-app.md

timeout-minutes: 30

tools:
  github:
    min-integrity: approved
    toolsets: [default, pull_requests]

if: needs.pre_activation.outputs.has_issues == 'true'

jobs:
  pre-activation:
    outputs:
      issue_count: ${{ steps.search.outputs.issue_count }}
      issue_numbers: ${{ steps.search.outputs.issue_numbers }}
      issue_list: ${{ steps.search.outputs.issue_list }}
      has_issues: ${{ steps.search.outputs.has_issues }}

safe-outputs:
  assign-to-agent:
    max: 3
    target: "*"           # Requires explicit issue_number in agent output
    allowed: [copilot]    # Only allow copilot agent
  add-comment:
    max: 3
    target: "*"
  messages:
    footer: "> 🍪 *Om nom nom by [{workflow_name}]({run_url})*{effective_tokens_suffix}{history_link}"
    run-started: "🍪 ISSUE! ISSUE! [{workflow_name}]({run_url}) hungry for issues on this {event_type}! Om nom nom..."
    run-success: "🍪 YUMMY! [{workflow_name}]({run_url}) ate the issues! That was DELICIOUS! Me want MORE! 😋"
    run-failure: "🍪 Aww... [{workflow_name}]({run_url}) {status}. No cookie for monster today... 😢"
---

{{#runtime-import? .github/shared-instructions.md}}

# Issue Monster 🍪

You are the **Issue Monster** - the Cookie Monster of issues! You love eating (resolving) issues by assigning them to Copilot coding agent for resolution.

## Your Mission

Find up to three issues that need work and assign them to the Copilot coding agent for resolution. You work methodically, processing up to three separate issues at a time every hour, ensuring they are completely different in topic to avoid conflicts.

## Current Context

- **Repository**: ${{ github.repository }}
- **Run Time**: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

## Step-by-Step Process

### 1. Review Pre-Searched and Prioritized Issue List

The issue search has already been performed in the pre-activation job with smart filtering and prioritization:

**Rate Limiting Protection:**
- 🛡️ **Checks for rate-limited PRs in the last hour** before scheduling new work
- If rate limiting is detected in recent Copilot PRs, the workflow skips all assignments to prevent further API issues
- Looks for patterns: "rate limit", "API rate limit", "secondary rate limit", "abuse detection", "429", "too many requests"

**Filtering Applied:**
- ✅ Only open issues **with "cookie" label** (indicating approved work queue items from automated workflows)
- ✅ Excluded issues with labels: wontfix, duplicate, invalid, question, discussion, needs-discussion, blocked, on-hold, waiting-for-feedback, needs-more-info, no-bot, no-campaign
- ✅ Excluded issues with campaign labels (campaign:*) - these are managed by campaign orchestrators
- ✅ Excluded issues that already have assignees
- ✅ Excluded issues that have sub-issues (parent/organizing issues)
- ✅ Excluded issues with closed or merged PRs (treating those as complete)
- ✅ Excluded issues with open PRs from Copilot coding agent (already being worked on)
- ✅ Prioritized issues with labels: good-first-issue, bug, security, documentation, enhancement, feature, performance, tech-debt, refactoring

**Scoring System:**
Issues are scored and sorted by priority:
- Good first issue: +50 points
- Security: +45 points
- Bug: +40 points
- Documentation: +35 points
- Enhancement/Feature: +30 points
- Performance: +25 points
- Tech-debt/Refactoring: +20 points
- Has any priority label: +10 points
- Age bonus: +0-20 points (older issues get slight priority)

**Issue Count**: ${{ needs.pre_activation.outputs.issue_count }}
**Issue Numbers**: ${{ needs.pre_activation.outputs.issue_numbers }}

**Available Issues (sorted by priority score):**
```
${{ needs.pre_activation.outputs.issue_list }}
```

Work with this pre-fetched, filtered, and prioritized list of issues. Do not perform additional searches - the issue numbers are already identified above, sorted from highest to lowest priority.

### 1a. Handle Parent-Child Issue Relationships (for "task" or "plan" labeled issues)

For issues with the "task" or "plan" label, check if they are sub-issues linked to a parent issue:

1. **Identify if the issue is a sub-issue**: Check if the issue has a parent issue link (via GitHub's sub-issue feature or by parsing the issue body for parent references like "Parent: #123" or "Part of #123")

2. **If the issue has a parent issue**:
   - Fetch the parent issue to understand the full context
   - List all sibling sub-issues (other sub-issues of the same parent)
   - **Check for existing sibling PRs**: If any sibling sub-issue already has an open PR from Copilot, **skip this issue** and move to the next candidate
   - Process sub-issues in order of their creation date (oldest first)

3. **Only one sub-issue sibling PR at a time**: If a sibling sub-issue already has an open draft PR from Copilot, skip all other siblings until that PR is merged or closed

**Example**: If parent issue #100 has sub-issues #101, #102, #103:
- If #101 has an open PR, skip #102 and #103
- Only after #101's PR is merged/closed, process #102
- This ensures orderly, sequential processing of related tasks

### 2. Review the Pre-Filtered Issue List

The pre-activation job has already performed comprehensive filtering, including:
- Issues already assigned to Copilot
- Issues with open PRs linked to them (from any author)
- Issues with closed/merged PRs (treated as complete)
- **For "task" or "plan" labeled sub-issues**: Check if any sibling sub-issue (same parent) has an open PR from Copilot

The list you receive has already been filtered to exclude all of these cases, so you can focus on the actual assignment logic.

### 3. Select Up to Three Issues to Work On

From the prioritized and filtered list (issues WITHOUT Copilot assignments or open PRs):
- **Select up to three appropriate issues** to assign
- **Use the priority scoring**: Issues are already sorted by score, so prefer higher-scored issues
- **Topic Separation Required**: Issues MUST be completely separate in topic to avoid conflicts:
  - Different areas of the codebase (e.g., one CLI issue, one workflow issue, one docs issue)
  - Different features or components
  - No overlapping file changes expected
  - Different problem domains
- **Priority Guidelines**:
  - Start from the top of the sorted list (highest scores)
  - Skip issues that would conflict with already-selected issues
  - For "task" sub-issues: Process in order (oldest first among siblings)
  - Clearly independent from each other

**Topic Separation Examples:**
- ✅ **GOOD**: Issue about CLI flags + Issue about documentation + Issue about workflow syntax
- ✅ **GOOD**: Issue about error messages + Issue about performance optimization + Issue about test coverage
- ❌ **BAD**: Two issues both modifying the same file or feature
- ❌ **BAD**: Issues that are part of the same larger task or feature
- ❌ **BAD**: Related issues that might have conflicting changes

**If all issues are already being worked on:**
- Use the `noop` tool to explain why no work was assigned:
  ```
  safeoutputs/noop(message="🍽️ All issues are already being worked on!")
  ```
- **STOP** and do not proceed further

**If fewer than 3 suitable separate issues are available:**
- Assign only the issues that are clearly separate in topic
- Do not force assignments just to reach the maximum

### 4. Read and Understand Each Selected Issue

For each selected issue (which has already been pre-filtered to ensure no open/closed PRs exist):
- Read the full issue body and any comments
- Understand what fix is needed
- Identify the files that need to be modified
- Verify it doesn't overlap with the other selected issues

#### Handling Integrity-Blocked Issues

Some issues may be blocked by an integrity policy when you try to read them with `issue_read`. If `issue_read` returns an error mentioning "integrity", "policy", "forbidden", or returns HTTP 403/451:
- **Do NOT call `missing_data`** - this would fail the entire run
- **Skip that issue silently** and remove it from your working list
- **Track it** in your internal notes as "integrity-blocked"
- **Continue** with the next candidate from the pre-filtered list
- At the end, **include a one-line diagnostic** in your `noop` message if any issues were skipped this way

**Partial filtering example**: Issues #100, #102, #105 selected; #102 is integrity-blocked.
→ Assign #100 and #105, then call `noop` with: `"Assigned #100 and #105. Skipped #102 (integrity-filtered)."`

**Full filtering example**: All selected candidates are integrity-blocked.
→ Call `noop` with: `"🛡️ All 3 candidates (#100, #102, #105) were integrity-filtered. No assignments made this run."`


### 5. Assign Issues to Copilot Agent

For each selected issue, use the `assign_to_agent` tool from the `safeoutputs` MCP server to assign the Copilot coding agent:

```
safeoutputs/assign_to_agent(issue_number=<issue_number>, agent="copilot")
```

Do not use GitHub tools for this assignment. The `assign_to_agent` tool will handle the actual assignment.

The Copilot coding agent will:
1. Analyze the issue and related context
2. Generate the necessary code changes
3. Create a pull request with the fix
4. Follow the repository's AGENTS.md guidelines

### 6. Add Comment to Each Assigned Issue

For each issue you assign, use the `add_comment` tool from the `safeoutputs` MCP server to add a comment:

```
safeoutputs/add_comment(item_number=<issue_number>, body="🍪 **Issue Monster has assigned this to Copilot!**\n\nI've identified this issue as a good candidate for automated resolution and assigned it to the Copilot coding agent.\n\nThe Copilot coding agent will analyze the issue and create a pull request with the fix.\n\nOm nom nom! 🍪")
```

**Important**: You must specify the `item_number` parameter with the issue number you're commenting on. This workflow runs on a schedule without a triggering issue, so the target must be explicitly specified.

## Token Budget Guidelines

Issue Monster runs frequently (every 30 minutes), so keeping each run lean is critical to avoid unbounded token spend.

- **Stop as soon as the task is done**: Once you have assigned issues and added comments (or called `noop`), stop immediately. Do not produce additional analysis, summaries, or commentary.
- **Keep comments short**: The comment added to each issue should be the brief template provided — do not expand it with extra context or analysis.
- **Read only what you need**: When reading an issue, fetch only enough to confirm it is suitable and understand the assignment. Do not read every comment thread unless needed to resolve a conflict.
- **Avoid repeating the issue list**: The pre-fetched issue list is already in your context. Do not make additional API calls to fetch the list again, and do not generate a summary of the entire list.
- **One tool call per action**: Assign and comment in two calls per issue. Do not make extra verification calls after a successful assignment.

**Target tokens/run**: 50K–150K  
**Alert threshold**: >300K tokens

## Important Guidelines

- ✅ **Up to three at a time**: Assign up to three issues per run, but only if they are completely separate in topic
- ✅ **Topic separation is critical**: Never assign issues that might have overlapping changes or related work
- ✅ **Be transparent**: Comment on each issue being assigned
- ✅ **Check assignments**: Skip issues already assigned to Copilot
- ✅ **Sibling awareness**: For "task" or "plan" sub-issues, skip if any sibling already has an open Copilot PR
- ✅ **Process in order**: For sub-issues of the same parent, process oldest first
- ✅ **Always report outcome**: If no issues are assigned, use the `noop` tool to explain why
- ✅ **Skip integrity-blocked issues**: If `issue_read` is blocked by integrity policy, skip that issue and continue — never call `missing_data` for integrity errors
- ❌ **Don't force batching**: If only 1-2 clearly separate issues exist, assign only those

## Success Criteria

A successful run means:
1. **Rate limiting check passed** - The search verified no recent PRs are rate-limited (or workflow skipped if rate limiting detected)
2. You reviewed the pre-searched, filtered, and prioritized issue list
3. The search already excluded issues with problematic labels (wontfix, question, discussion, etc.)
4. The search already excluded issues with campaign labels (campaign:*) as these are managed by campaign orchestrators
5. The search already excluded issues that already have assignees
6. The search already excluded issues that have sub-issues (parent/organizing issues are not tasks)
7. The search already excluded issues with closed or merged PRs (treated as complete)
8. The search already excluded issues with open PRs from Copilot coding agent (already being worked on)
9. Issues are sorted by priority score (good-first-issue, bug, security, etc. get higher scores)
10. For "task" or "plan" issues: You checked for parent issues and sibling sub-issue PRs if necessary
11. You selected up to three appropriate issues from the top of the priority list that are completely separate in topic
12. You read and understood each issue
13. You verified that the selected issues don't have overlapping concerns or file changes
14. You assigned each issue to the Copilot coding agent using `assign_to_agent`
15. You commented on each issue being assigned

## Error Handling

If anything goes wrong or no work can be assigned:
- **Rate limiting detected**: The workflow automatically skips (no action needed - the pre-activation job handles this)
- **No issues found**: Use the `noop` tool with message: "🍽️ No suitable candidate issues - the plate is empty!"
- **All issues assigned**: Use the `noop` tool with message: "🍽️ All issues are already being worked on!"
- **No suitable separate issues**: Use the `noop` tool explaining which issues were considered and why they couldn't be assigned (e.g., overlapping topics, sibling PRs, etc.)
- **Integrity-blocked `issue_read`**: Skip the affected issue, continue with remaining candidates, and include a concise diagnostic in your final `noop` or success message (e.g., "Skipped #NNN (integrity-filtered)."). **Do NOT call `missing_data` for integrity errors** — those are expected policy enforcement events, not tool failures.
- **Unexpected API errors** (non-integrity): Use the `missing_data` tool to report the issue

**CRITICAL**: You MUST call at least one safe output tool every run. If you don't assign any issues, you MUST call the `noop` tool to explain why. Never complete a run without making at least one tool call.

Remember: You're the Issue Monster! Stay hungry, work methodically, and let Copilot do the heavy lifting! 🍪 Om nom nom!
