# Update Project Error Analysis

## Workflow Run
- **URL**: https://github.com/githubnext/gh-aw/actions/runs/20234787361/job/58086697021
- **Branch**: `test-new-update-project`  
- **Commit**: accde6d62135fcee807787bcd906b6e68ec4d1fd
- **Status**: Completed with errors in Update Project step

## Error Summary

The Update Project safe-output step failed with the following GraphQL error:

```
GraphQL Error during: Resolving project from URL
Message: Project not found or not accessible: https://github.com/orgs/githubnext/projects/60 (organization=githubnext, number=60)
```

## Root Cause

The refactored code in `test-new-update-project` branch uses a **direct GraphQL query** to fetch project by number:

```graphql
query($login: String!, $number: Int!) {
  organization(login: $login) {
    projectV2(number: $number) {
      id
      number
      title
      url
      owner {
        __typename
        ... on Organization { login }
        ... on User { login }
      }
    }
  }
}
```

This query returned `null` for `projectV2(number: 60)`, triggering the error.

## Comparison: Main Branch vs Test Branch

### Main Branch Approach
- Lists all projects: `projectsV2(first: 100)`
- Searches through the list to find matching project
- More resilient to permission issues (returns empty list vs null)
- Limitation: Only checks first 100 projects

### Test Branch Approach  
- Direct query: `projectV2(number: $number)`
- More efficient (single query vs list + search)
- **Problem**: Returns `null` if project doesn't exist OR token lacks permission
- Better error reporting but harder to distinguish between "doesn't exist" vs "no permission"

## Possible Causes

1. **Project #60 Does Not Exist**
   - The project may have been deleted
   - The project number in the URL may be incorrect
   - Project #60 may never have existed in githubnext org

2. **Token Permission Issue**
   - The `GH_AW_PROJECT_GITHUB_TOKEN` secret may not have Projects v2 read access
   - Required scopes: `read:project` for reading, `write:project` for updates
   - The token may not have org-level project access

3. **Project Type Mismatch**
   - Could be a classic Projects board (v1) instead of Projects v2
   - Classic boards use different GraphQL schema

## Recommendations

### Immediate Fixes

1. **Verify Project Exists**
   ```bash
   # Check if project #60 exists
   gh api graphql -f query='
     query {
       organization(login: "githubnext") {
         projectV2(number: 60) {
           id
           title
           number
         }
       }
     }
   '
   ```

2. **Use a Known-Valid Project for Testing**
   - Create a dedicated test project for the gh-aw repository
   - Document the project number in the test workflow
   - Example: Create "GH-AW Test Project" and use its actual number

3. **Verify Token Permissions**
   ```bash
   # Test with the secret token
   gh api graphql -f query='
     query {
       organization(login: "githubnext") {
         projectsV2(first: 10) {
           nodes {
             number
             title
           }
         }
       }
     }
   ' --header "Authorization: Bearer $GH_AW_PROJECT_GITHUB_TOKEN"
   ```

### Code Improvements

1. **Add Fallback to List Query**
   ```javascript
   // Try direct query first (efficient)
   let project = await queryProjectDirectly(number);
   
   // Fallback to list query if null (may be permission issue)
   if (!project) {
     project = await findProjectInList(number);
   }
   
   if (!project) {
     throw new Error(`Project #${number} not found or not accessible`);
   }
   ```

2. **Better Error Differentiation**
   ```javascript
   try {
     projectResult = await github.graphql(query, vars);
     if (!projectResult.organization.projectV2) {
       // Try to list projects to see if it's a permission issue
       const listResult = await github.graphql(listQuery, { login });
       if (listResult.organization.projectsV2.nodes.length === 0) {
         throw new Error(
           `Cannot access any projects in ${login}. ` +
           `This likely indicates the token lacks 'read:project' permission.`
         );
       } else {
         throw new Error(
           `Project #${number} not found in ${login}. ` +
           `Available projects: ${listResult.organization.projectsV2.nodes.map(p => `#${p.number}`).join(', ')}`
         );
       }
     }
   } catch (error) {
     // Enhanced error logging
   }
   ```

3. **Add Validation Step**
   Add a preliminary validation step in the workflow that checks if the project exists before running the agent:
   ```yaml
   - name: Validate Project Exists
     uses: actions/github-script@v7
     with:
       script: |
         const projectNumber = 60;
         const result = await github.graphql(`
           query($login: String!, $number: Int!) {
             organization(login: $login) {
               projectV2(number: $number) {
                 id
                 title
               }
             }
           }
         `, { login: context.repo.owner, number: projectNumber });
         
         if (!result.organization.projectV2) {
           core.setFailed(`Project #${projectNumber} not found or not accessible`);
         } else {
           core.info(`✓ Project validated: ${result.organization.projectV2.title}`);
         }
   ```

## Action Items

- [ ] Verify project #60 exists at https://github.com/orgs/githubnext/projects
- [ ] If it doesn't exist, create a dedicated test project or use an existing one
- [ ] Update the dev.md workflow with a valid project number
- [ ] Verify `GH_AW_PROJECT_GITHUB_TOKEN` has `read:project` and `write:project` scopes
- [ ] Consider merging the two approaches: direct query with fallback to list
- [ ] Add better error messages to distinguish between different failure modes
- [ ] Document the test project setup in the repository

## Related Files

- **Workflow**: `.github/workflows/dev.md` (test-new-update-project branch)
- **Safe Output Implementation**: `pkg/workflow/js/update_project.cjs`
- **Compiled Workflow**: `.github/workflows/dev.lock.yml` (test-new-update-project branch)

## Logs Reference

Key log lines from the failed run:

```
Looking up project #60 from URL: https://github.com/orgs/githubnext/projects/60
[1/5] Fetching repository information...
✓ Repository: githubnext/gh-aw (Organization)
[2/5] Resolving project from URL (scope=orgs, login=githubnext, number=60)...
##[error]GraphQL Error during: Resolving project from URL
##[error]Message: Project not found or not accessible: https://github.com/orgs/githubnext/projects/60 (organization=githubnext, number=60)
```

The logs confirm that:
1. Repository lookup succeeded ✓
2. Project lookup failed ✗
3. No GraphQL errors array was logged (would indicate permission issues with detailed error types)
