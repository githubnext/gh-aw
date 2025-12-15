# Recommended Fixes for Update Project Errors

## Summary

This document provides actionable fixes for the GraphQL error encountered in workflow run [#20234787361](https://github.com/githubnext/gh-aw/actions/runs/20234787361/job/58086697021).

**Error**: `Project not found or not accessible: https://github.com/orgs/githubnext/projects/60`

## Fix Option 1: Hybrid Approach (Recommended)

Combine the efficiency of direct queries with the resilience of list-based search.

### Implementation

```javascript
/**
 * Try to fetch project directly first, fall back to list search if needed
 * @param {object} github - GitHub client
 * @param {string} login - Organization or user login
 * @param {string} scope - "orgs" or "users"
 * @param {number} projectNumber - Project number to find
 * @returns {Promise<{id: string, number: number, title: string} | null>}
 */
async function findProject(github, login, scope, projectNumber) {
  // Attempt 1: Direct query (fast, but may return null for permission issues)
  try {
    const directQuery = scope === "orgs"
      ? `query($login: String!, $number: Int!) {
          organization(login: $login) {
            projectV2(number: $number) {
              id
              number
              title
              url
            }
          }
        }`
      : `query($login: String!, $number: Int!) {
          user(login: $login) {
            projectV2(number: $number) {
              id
              number
              title
              url
            }
          }
        }`;
    
    const result = await github.graphql(directQuery, { login, number: projectNumber });
    const project = scope === "orgs" ? result.organization?.projectV2 : result.user?.projectV2;
    
    if (project) {
      core.info(`✓ Found project #${projectNumber} via direct query`);
      return project;
    }
  } catch (error) {
    core.warning(`Direct query failed: ${error.message}. Trying list-based search...`);
  }
  
  // Attempt 2: List all projects and search (slower, but works with limited permissions)
  try {
    const listQuery = scope === "orgs"
      ? `query($login: String!) {
          organization(login: $login) {
            projectsV2(first: 100) {
              nodes {
                id
                number
                title
                url
              }
            }
          }
        }`
      : `query($login: String!) {
          user(login: $login) {
            projectsV2(first: 100) {
              nodes {
                id
                number
                title
                url
              }
            }
          }
        }`;
    
    const result = await github.graphql(listQuery, { login });
    const projects = scope === "orgs" 
      ? result.organization?.projectsV2?.nodes || []
      : result.user?.projectsV2?.nodes || [];
    
    const project = projects.find(p => p.number === projectNumber);
    
    if (project) {
      core.info(`✓ Found project #${projectNumber} via list search`);
      return project;
    }
    
    // Provide helpful error with available projects
    if (projects.length > 0) {
      const available = projects.map(p => `#${p.number} (${p.title})`).slice(0, 5).join(', ');
      throw new Error(
        `Project #${projectNumber} not found. ` +
        `Available projects include: ${available}${projects.length > 5 ? ', ...' : ''}`
      );
    } else {
      throw new Error(
        `No projects accessible in ${login}. ` +
        `The token may lack 'read:project' permission or no projects exist.`
      );
    }
  } catch (error) {
    throw new Error(
      `Failed to find project #${projectNumber}: ${error.message}`
    );
  }
}
```

### Update `updateProject` function

```javascript
async function updateProject(output) {
  const { owner, repo } = context.repo;
  const projectInfo = parseProjectUrl(output.project);
  const projectNumberFromUrl = parseInt(projectInfo.projectNumber, 10);
  const campaignId = output.campaign_id || generateCampaignId(output.project, projectInfo.projectNumber);

  try {
    core.info(`Looking up project #${projectNumberFromUrl} from URL: ${output.project}`);
    core.info("[1/5] Fetching repository information...");
    
    const repoResult = await github.graphql(
      `query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          id
          owner {
            id
            __typename
          }
        }
      }`,
      { owner, repo }
    );
    
    const repositoryId = repoResult.repository.id;
    const ownerType = repoResult.repository.owner.__typename;
    core.info(`✓ Repository: ${owner}/${repo} (${ownerType})`);

    // Use hybrid approach to find project
    core.info(`[2/5] Resolving project from URL (scope=${projectInfo.scope}, login=${projectInfo.ownerLogin}, number=${projectNumberFromUrl})...`);
    
    const project = await findProject(
      github,
      projectInfo.ownerLogin,
      projectInfo.scope,
      projectNumberFromUrl
    );
    
    if (!project) {
      throw new Error(
        `Project #${projectNumberFromUrl} not found or not accessible at ${output.project}`
      );
    }
    
    const projectId = project.id;
    const projectNumber = project.number;
    core.info(`✓ Resolved project #${projectNumber}: ${project.title}`);
    
    // Rest of the function continues...
    core.info("[3/5] Linking project to repository...");
    // ... rest of implementation
  } catch (error) {
    // Enhanced error logging
    core.error(`Failed to manage project: ${error.message}`);
    throw error;
  }
}
```

## Fix Option 2: Better Error Messages

If keeping the direct query approach, enhance error messages:

```javascript
async function resolveProject(github, projectInfo, projectNumberFromUrl) {
  try {
    const projectNumberInt = parseInt(projectNumberFromUrl, 10);
    if (!Number.isFinite(projectNumberInt)) {
      throw new Error(`Invalid project number parsed from URL: ${projectNumberFromUrl}`);
    }

    const query = projectInfo.scope === "orgs"
      ? `query($login: String!, $number: Int!) {
          organization(login: $login) {
            projectV2(number: $number) {
              id
              number
              title
              url
            }
          }
        }`
      : `query($login: String!, $number: Int!) {
          user(login: $login) {
            projectV2(number: $number) {
              id
              number
              title
              url
            }
          }
        }`;
    
    const result = await github.graphql(query, {
      login: projectInfo.ownerLogin,
      number: projectNumberInt
    });
    
    const project = projectInfo.scope === "orgs"
      ? result.organization?.projectV2
      : result.user?.projectV2;
    
    if (!project) {
      // Try to provide more context
      const checkUrl = `https://github.com/${projectInfo.scope}/${projectInfo.ownerLogin}/projects/${projectNumberFromUrl}`;
      throw new Error(
        `Project #${projectNumberFromUrl} not found or not accessible.\n\n` +
        `Troubleshooting:\n` +
        `  1. Verify the project exists: ${checkUrl}\n` +
        `  2. Check if it's a Projects v2 board (not classic Projects)\n` +
        `  3. Ensure GH_AW_PROJECT_GITHUB_TOKEN has 'read:project' scope\n` +
        `  4. Verify the token has access to the ${projectInfo.ownerLogin} organization projects\n\n` +
        `If the project doesn't exist, create it manually or use a different project number.`
      );
    }
    
    return project;
  } catch (error) {
    if (error.message.includes('not found or not accessible')) {
      throw error; // Re-throw our enhanced error
    }
    // Handle GraphQL errors
    logGraphQLError(error, "Resolving project from URL");
    throw error;
  }
}
```

## Fix Option 3: Add Pre-Validation Step

Add a validation step in the workflow before the agent runs:

```yaml
jobs:
  validate-project:
    runs-on: ubuntu-latest
    outputs:
      project-exists: ${{ steps.check.outputs.exists }}
      project-title: ${{ steps.check.outputs.title }}
    steps:
      - name: Check Project Exists
        id: check
        uses: actions/github-script@v7
        env:
          GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
        with:
          github-token: ${{ env.GH_AW_PROJECT_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
          script: |
            const projectNumber = 60; // Or parse from input
            const orgLogin = context.repo.owner;
            
            try {
              const result = await github.graphql(`
                query($login: String!, $number: Int!) {
                  organization(login: $login) {
                    projectV2(number: $number) {
                      id
                      title
                      number
                    }
                  }
                }
              `, { login: orgLogin, number: projectNumber });
              
              if (result.organization.projectV2) {
                core.setOutput('exists', 'true');
                core.setOutput('title', result.organization.projectV2.title);
                core.info(`✅ Project #${projectNumber} validated: ${result.organization.projectV2.title}`);
              } else {
                core.setOutput('exists', 'false');
                core.setFailed(
                  `❌ Project #${projectNumber} not found.\n` +
                  `   Create it at https://github.com/orgs/${orgLogin}/projects/new\n` +
                  `   Or update the workflow with a valid project number.`
                );
              }
            } catch (error) {
              core.setOutput('exists', 'false');
              core.setFailed(`Error validating project: ${error.message}`);
            }

  agent:
    needs: validate-project
    if: needs.validate-project.outputs.project-exists == 'true'
    # ... rest of agent job
```

## Fix Option 4: Use Environment-Specific Project

Create different project numbers for different environments:

```yaml
# In workflow file
env:
  # Use different projects for different branches/environments
  PROJECT_NUMBER: ${{ github.ref == 'refs/heads/main' && '42' || '60' }}
```

```javascript
// In update_project.cjs
function getProjectUrl(projectNumber) {
  // Allow override via workflow environment
  const envProjectNumber = process.env.PROJECT_NUMBER;
  const finalNumber = envProjectNumber || projectNumber;
  
  return `https://github.com/orgs/githubnext/projects/${finalNumber}`;
}
```

## Recommended Implementation Plan

1. **Immediate** (Today):
   - [ ] Verify if project #60 exists at https://github.com/orgs/githubnext/projects/60
   - [ ] If it doesn't exist, create a test project or identify an existing one
   - [ ] Update the `dev.md` workflow with the correct project number

2. **Short-term** (This Week):
   - [ ] Implement **Fix Option 1** (Hybrid Approach) in `update_project.cjs`
   - [ ] Add better error messages with troubleshooting steps
   - [ ] Add unit tests for the findProject function

3. **Long-term** (Next Sprint):
   - [ ] Add pre-validation step to workflows (Fix Option 3)
   - [ ] Document project setup requirements in repository
   - [ ] Create dedicated test projects for CI/CD workflows
   - [ ] Add monitoring/alerting for project access issues

## Testing the Fix

After implementing the fix, test with:

```bash
# Test with valid project
node -e "
const { updateProject } = require('./pkg/workflow/js/update_project.cjs');
updateProject({
  type: 'update_project',
  project: 'https://github.com/orgs/githubnext/projects/60',
  content_number: 123
});
"

# Test with invalid project
node -e "
const { updateProject } = require('./pkg/workflow/js/update_project.cjs');
updateProject({
  type: 'update_project',
  project: 'https://github.com/orgs/githubnext/projects/99999',
  content_number: 123
});
" # Should provide helpful error message
```

## Additional Resources

- [GitHub GraphQL API - Projects](https://docs.github.com/en/graphql/reference/objects#projectv2)
- [Creating a personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
- [Project scopes documentation](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/scopes-for-oauth-apps#available-scopes)

## Summary

The recommended approach is **Fix Option 1 (Hybrid Approach)** because it:
- ✅ Maintains performance with direct queries when possible
- ✅ Falls back gracefully to list-based search
- ✅ Provides detailed error messages with available projects
- ✅ Works across different permission scenarios
- ✅ Minimizes code changes while maximizing robustness
