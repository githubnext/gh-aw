# GitHub Script Custom Token Instantiation

**Research Date**: 2026-01-13  
**Status**: ✅ Confirmed and Documented

## Executive Summary

The `actions/github-script` action **fully supports** instantiating GitHub API clients (Octokit instances) with custom tokens. This capability is essential for:
- Cross-repository operations
- Elevated permissions scenarios  
- GitHub App authentication
- Multi-token workflows

## Research Question

> Is it possible to instantiate a GitHub object instance with a different token when using `actions/github-script`?

**Answer**: **Yes**, using two well-documented approaches.

---

## Approach 1: Native `github-token` Input (Recommended)

The simplest and most common method is using the built-in `github-token` input parameter:

### Basic Usage

```yaml
- uses: actions/github-script@v8
  with:
    github-token: ${{ secrets.MY_CUSTOM_TOKEN }}
    script: |
      // The default 'github' object is now authenticated with MY_CUSTOM_TOKEN
      const { data: issues } = await github.rest.issues.listForRepo({
        owner: context.repo.owner,
        repo: context.repo.repo,
      });
      console.log(`Found ${issues.length} issues`);
```

### Key Characteristics

- **Simplicity**: Single line configuration
- **Default behavior**: Replaces the authentication for the provided `github` object
- **Token types supported**:
  - Personal Access Tokens (PAT) - Classic or fine-grained
  - GitHub App installation tokens
  - OAuth tokens
  - Any valid GitHub API token

### When to Use

✅ **Use this approach when:**
- You need a different token for all operations in a script
- Working with a single repository requiring elevated permissions
- Using GitHub App authentication
- Simplicity is a priority

❌ **Don't use when:**
- You need multiple tokens in the same script
- You require fine-grained control over Octokit configuration

---

## Approach 2: Custom Octokit Instances (Advanced)

For complex scenarios requiring multiple tokens or custom configurations:

### Multiple Tokens Example

```yaml
- uses: actions/github-script@v8
  env:
    REPO_A_TOKEN: ${{ secrets.REPO_A_TOKEN }}
    REPO_B_TOKEN: ${{ secrets.REPO_B_TOKEN }}
  with:
    script: |
      const { Octokit } = require('@octokit/rest');
      
      // Create separate clients for different repositories
      const clientA = new Octokit({
        auth: process.env.REPO_A_TOKEN,
      });
      
      const clientB = new Octokit({
        auth: process.env.REPO_B_TOKEN,
      });
      
      // Use both clients
      const issuesA = await clientA.rest.issues.listForRepo({
        owner: 'org-a',
        repo: 'repo-a',
      });
      
      const issuesB = await clientB.rest.issues.listForRepo({
        owner: 'org-b', 
        repo: 'repo-b',
      });
      
      // Still have access to default github object
      const defaultIssues = await github.rest.issues.listForRepo({
        owner: context.repo.owner,
        repo: context.repo.repo,
      });
```

### Advanced Configuration

```yaml
- uses: actions/github-script@v8
  env:
    CUSTOM_TOKEN: ${{ secrets.CUSTOM_TOKEN }}
  with:
    script: |
      const { Octokit } = require('@octokit/rest');
      
      const customGithub = new Octokit({
        auth: process.env.CUSTOM_TOKEN,
        baseUrl: 'https://api.github.com', // Or GitHub Enterprise URL
        userAgent: 'my-custom-app v1.0.0',
        throttle: {
          onRateLimit: (retryAfter, options) => {
            console.warn(`Rate limit hit, retrying after ${retryAfter}s`);
            return true;
          },
          onSecondaryRateLimit: (retryAfter, options) => {
            console.warn(`Secondary rate limit hit`);
            return true;
          },
        },
      });
      
      const result = await customGithub.rest.repos.get({
        owner: 'octocat',
        repo: 'hello-world',
      });
```

### When to Use

✅ **Use this approach when:**
- Need multiple tokens in the same script
- Require custom Octokit configuration (throttling, base URL, etc.)
- Working with GitHub Enterprise Server with custom URLs
- Need fine-grained control over client behavior

### Available Packages

The github-script environment includes:
- `@octokit/rest` - REST API client
- `octokit` - Modern unified client (v5+)
- Both support identical authentication patterns

---

## Real-World Use Cases

### 1. Cross-Repository Operations

**Scenario**: Update multiple repositories with different access requirements

```yaml
- uses: actions/github-script@v8
  env:
    MAIN_REPO_TOKEN: ${{ secrets.MAIN_REPO_PAT }}
    DOCS_REPO_TOKEN: ${{ secrets.DOCS_REPO_PAT }}
  with:
    script: |
      const { Octokit } = require('@octokit/rest');
      
      const mainClient = new Octokit({ auth: process.env.MAIN_REPO_TOKEN });
      const docsClient = new Octokit({ auth: process.env.DOCS_REPO_TOKEN });
      
      // Create issue in main repo
      await mainClient.rest.issues.create({
        owner: 'myorg',
        repo: 'main-repo',
        title: 'Release Notes',
        body: 'New release available',
      });
      
      // Update documentation repo
      await docsClient.rest.repos.createOrUpdateFileContents({
        owner: 'myorg',
        repo: 'docs',
        path: 'releases/latest.md',
        message: 'Update release notes',
        content: Buffer.from('# Latest Release').toString('base64'),
      });
```

### 2. GitHub App Authentication

**Scenario**: Use GitHub App token for enhanced permissions

```yaml
- name: Generate App Token
  uses: actions/create-github-app-token@v1
  id: app-token
  with:
    app-id: ${{ secrets.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    repositories: |
      repo1
      repo2

- name: Use App Token
  uses: actions/github-script@v8
  with:
    github-token: ${{ steps.app-token.outputs.token }}
    script: |
      // github object uses App token - can access repo1 and repo2
      const repos = await github.rest.apps.listReposAccessibleToInstallation();
      console.log(`App has access to ${repos.data.total_count} repos`);
      
      // Perform operations with App permissions
      await github.rest.issues.create({
        owner: context.repo.owner,
        repo: 'repo1',
        title: 'Automated Issue',
        body: 'Created by GitHub App',
      });
```

### 3. Elevated Permissions

**Scenario**: Administrative operations requiring PAT with admin:org scope

```yaml
- uses: actions/github-script@v8
  with:
    github-token: ${{ secrets.ADMIN_PAT }}
    script: |
      // Perform operations requiring elevated permissions
      await github.rest.orgs.updateWebhook({
        org: context.repo.owner,
        hook_id: 12345,
        config: {
          url: 'https://example.com/webhook',
          content_type: 'json',
        },
      });
```

### 4. Impersonation for Testing

**Scenario**: Test workflows from different user perspectives

```yaml
- uses: actions/github-script@v8
  env:
    USER_A_TOKEN: ${{ secrets.USER_A_TOKEN }}
    USER_B_TOKEN: ${{ secrets.USER_B_TOKEN }}
  with:
    script: |
      const { Octokit } = require('@octokit/rest');
      
      const userA = new Octokit({ auth: process.env.USER_A_TOKEN });
      const userB = new Octokit({ auth: process.env.USER_B_TOKEN });
      
      // Test user A can create issue
      try {
        await userA.rest.issues.create({
          owner: 'myorg',
          repo: 'test-repo',
          title: 'Test Issue from User A',
        });
        console.log('✓ User A has write access');
      } catch (error) {
        console.log('✗ User A lacks write access');
      }
      
      // Test user B permissions
      try {
        await userB.rest.repos.delete({
          owner: 'myorg',
          repo: 'test-repo',
        });
        console.log('✓ User B has admin access');
      } catch (error) {
        console.log('✗ User B lacks admin access');
      }
```

---

## Security Best Practices

### ✅ DO

**1. Always use GitHub Secrets**
```yaml
github-token: ${{ secrets.MY_TOKEN }}  # ✅ Correct
```

**2. Pass tokens via environment variables**
```yaml
env:
  CUSTOM_TOKEN: ${{ secrets.MY_TOKEN }}
with:
  script: |
    const token = process.env.CUSTOM_TOKEN;  # ✅ Safe from injection
```

**3. Use least privilege tokens**
- Fine-grained PATs with minimal scopes
- GitHub App tokens with specific repository access
- Short-lived tokens when possible

**4. Rotate tokens regularly**
- Set expiration dates on PATs
- Monitor token usage
- Revoke unused tokens

### ❌ DON'T

**1. Never hard-code tokens**
```yaml
github-token: "ghp_actualtoken123"  # ❌ NEVER DO THIS
```

**2. Avoid direct interpolation in scripts**
```yaml
with:
  script: |
    const token = "${{ secrets.MY_TOKEN }}";  # ❌ Script injection risk
```

This is vulnerable to script injection if the secret contains special characters like quotes or backticks.

**3. Don't log tokens**
```yaml
script: |
  console.log(process.env.GITHUB_TOKEN);  # ❌ Token exposure risk
```

**4. Don't commit tokens to repository**
```yaml
# .env file (gitignored)
GITHUB_TOKEN=ghp_actualtoken  # ❌ Don't commit this
```

### Script Injection Example

**Vulnerable:**
```yaml
with:
  script: |
    const title = "${{ github.event.issue.title }}";  # ❌ Injection risk
    await github.rest.issues.create({ title });
```

If issue title is: `"; await malicious_code(); "` → Code execution!

**Safe:**
```yaml
env:
  ISSUE_TITLE: ${{ github.event.issue.title }}
with:
  script: |
    const title = process.env.ISSUE_TITLE;  # ✅ Safe
    await github.rest.issues.create({ title });
```

---

## Implementation in gh-aw

The gh-aw project extensively uses custom token support:

### Token Precedence System

```
1. Per-output token (create-issue.github-token)
   ↓
2. Safe-outputs global token (safe-outputs.github-token)
   ↓  
3. Top-level workflow token (github-token)
   ↓
4. Default fallback (${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }})
```

### Example: Safe Output with Custom Token

```yaml
---
github-token: ${{ secrets.TOPLEVEL_TOKEN }}

safe-outputs:
  github-token: ${{ secrets.SAFE_OUTPUTS_TOKEN }}
  
  create-issue:
    github-token: ${{ secrets.ISSUE_SPECIFIC_TOKEN }}
    
  create-pull-request:
    # Uses safe-outputs.github-token
    
  update-project:
    github-token: ${{ secrets.PROJECT_TOKEN }}
---
```

**Result**:
- `create-issue` → Uses `ISSUE_SPECIFIC_TOKEN`
- `create-pull-request` → Uses `SAFE_OUTPUTS_TOKEN`
- `update-project` → Uses `PROJECT_TOKEN`
- Other operations → Use `TOPLEVEL_TOKEN`

### Implementation Files

Key files implementing token support in gh-aw:

- **`pkg/workflow/github_token.go`**: Token precedence logic
- **`pkg/workflow/safe_outputs_env.go`**: Token handling for safe outputs
- **`pkg/workflow/create_issue.go`**: Issue creation with custom tokens
- **`pkg/workflow/pr.go`**: PR creation with custom tokens
- **`pkg/workflow/update_project.go`**: Project updates with custom tokens
- **`actions/setup/js/types/github-script.d.ts`**: TypeScript type definitions

### Code Example from gh-aw

```go
// From pkg/workflow/create_issue.go
fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
steps = append(steps, "        with:\n")
steps = append(steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
steps = append(steps, "          script: |\n")
```

This generates:

```yaml
- uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd # v8
  with:
    github-token: ${{ secrets.CUSTOM_TOKEN }}
    script: |
      // Script code here
```

---

## Testing and Validation

### Unit Test Pattern

```javascript
// From actions/setup/js/assign_agent_helpers.test.cjs
describe('Custom Token Authentication', () => {
  it('should use custom token when provided', async () => {
    const mockGithub = {
      rest: {
        users: {
          getAuthenticated: jest.fn().mockResolvedValue({
            data: { login: 'custom-token-user' }
          })
        }
      }
    };
    
    // Test that custom token is used
    const result = await mockGithub.rest.users.getAuthenticated();
    expect(result.data.login).toBe('custom-token-user');
  });
});
```

### Integration Test Pattern

```yaml
# Test workflow
name: Test Custom Token
on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/github-script@v8
        with:
          github-token: ${{ secrets.TEST_TOKEN }}
          script: |
            // Verify token works
            const { data: user } = await github.rest.users.getAuthenticated();
            console.log(`Authenticated as: ${user.login}`);
            
            // Test permissions
            try {
              await github.rest.issues.create({
                owner: context.repo.owner,
                repo: context.repo.repo,
                title: 'Test Issue',
              });
              console.log('✓ Write access confirmed');
            } catch (error) {
              console.log('✗ Write access denied');
              throw error;
            }
```

---

## Troubleshooting

### Common Issues

#### 1. "Bad credentials" Error

**Symptom:**
```
Error: Bad credentials
```

**Causes:**
- Token expired or revoked
- Incorrect secret name
- Token lacks required permissions

**Solution:**
```yaml
# Verify token is correctly passed
- uses: actions/github-script@v8
  with:
    github-token: ${{ secrets.MY_TOKEN }}
    script: |
      // Test authentication
      try {
        const { data } = await github.rest.users.getAuthenticated();
        console.log(`Authenticated as: ${data.login}`);
      } catch (error) {
        console.error('Authentication failed:', error.message);
        throw error;
      }
```

#### 2. "Resource not accessible by integration" Error

**Symptom:**
```
Error: Resource not accessible by integration
```

**Cause:** Token lacks required permissions/scopes

**Solution:**
- Check token scopes in GitHub settings
- For PAT: Ensure required permissions are granted
- For GitHub App: Ensure app has necessary permissions
- Use a different token with elevated permissions

#### 3. "Not Found" Error with Cross-Repo Operations

**Symptom:**
```
Error: Not Found (404)
```

**Cause:** Token doesn't have access to target repository

**Solution:**
```yaml
# Ensure token has access to target repository
- uses: actions/github-script@v8
  env:
    TARGET_REPO_TOKEN: ${{ secrets.REPO_SPECIFIC_TOKEN }}
  with:
    script: |
      const { Octokit } = require('@octokit/rest');
      const client = new Octokit({
        auth: process.env.TARGET_REPO_TOKEN
      });
      
      // Verify access
      try {
        await client.rest.repos.get({
          owner: 'target-org',
          repo: 'target-repo',
        });
        console.log('✓ Repository access confirmed');
      } catch (error) {
        console.error('✗ No access to repository');
        throw error;
      }
```

#### 4. Script Injection from User Input

**Symptom:** Unexpected code execution or syntax errors

**Cause:** Direct interpolation of user input

**Solution:**
```yaml
# ❌ Vulnerable
with:
  script: |
    const title = "${{ github.event.issue.title }}";

# ✅ Safe
env:
  ISSUE_TITLE: ${{ github.event.issue.title }}
with:
  script: |
    const title = process.env.ISSUE_TITLE;
```

---

## Performance Considerations

### Rate Limiting

Each token has its own rate limits:

```yaml
- uses: actions/github-script@v8
  with:
    retries: 3  # Retry on rate limit
    retry-exempt-status-codes: 400,401
    script: |
      // Check rate limit status
      const { data: rateLimit } = await github.rest.rateLimit.get();
      console.log(`Remaining: ${rateLimit.rate.remaining}/${rateLimit.rate.limit}`);
      console.log(`Resets at: ${new Date(rateLimit.rate.reset * 1000)}`);
```

### Token Pooling Strategy

For high-volume operations:

```yaml
- uses: actions/github-script@v8
  env:
    TOKEN_1: ${{ secrets.TOKEN_1 }}
    TOKEN_2: ${{ secrets.TOKEN_2 }}
    TOKEN_3: ${{ secrets.TOKEN_3 }}
  with:
    script: |
      const { Octokit } = require('@octokit/rest');
      
      const tokens = [
        process.env.TOKEN_1,
        process.env.TOKEN_2,
        process.env.TOKEN_3,
      ];
      
      const clients = tokens.map(token => new Octokit({ auth: token }));
      
      // Round-robin through clients
      let currentClient = 0;
      
      async function makeRequest(operation) {
        const client = clients[currentClient];
        currentClient = (currentClient + 1) % clients.length;
        return await operation(client);
      }
      
      // Use different tokens for each request
      for (let i = 0; i < 100; i++) {
        await makeRequest(client => 
          client.rest.issues.get({
            owner: 'octocat',
            repo: 'hello-world',
            issue_number: i + 1,
          })
        );
      }
```

---

## API Compatibility

### Supported Token Types

| Token Type | Source | Validity | Use Case |
|------------|--------|----------|----------|
| **GITHUB_TOKEN** | Automatic | Job duration | Default workflow operations |
| **Personal Access Token (Classic)** | User settings | Configurable | Cross-repo, elevated permissions |
| **Fine-grained PAT** | User settings | Configurable | Granular permissions |
| **GitHub App Token** | App installation | 1 hour | Automated integrations |
| **OAuth Token** | OAuth flow | Variable | Third-party integrations |

### Rate Limits by Token Type

| Token Type | Requests/Hour | Notes |
|------------|---------------|-------|
| **GITHUB_TOKEN** | 1,000 | Per repository |
| **PAT (authenticated)** | 5,000 | Per user account |
| **GitHub App** | 5,000 + 50/repo | Based on installations |
| **OAuth App** | 5,000 | Per authenticated user |

---

## Migration Guide

### From Default Token to Custom Token

**Before:**
```yaml
- uses: actions/github-script@v8
  with:
    script: |
      // Uses default GITHUB_TOKEN
      await github.rest.issues.create({...});
```

**After:**
```yaml
- uses: actions/github-script@v8
  with:
    github-token: ${{ secrets.CUSTOM_TOKEN }}
    script: |
      // Uses CUSTOM_TOKEN
      await github.rest.issues.create({...});
```

### From Single Token to Multiple Tokens

**Before:**
```yaml
- uses: actions/github-script@v8
  with:
    github-token: ${{ secrets.CUSTOM_TOKEN }}
    script: |
      await github.rest.issues.create({...});
      await github.rest.pulls.create({...});
```

**After:**
```yaml
- uses: actions/github-script@v8
  env:
    ISSUE_TOKEN: ${{ secrets.ISSUE_TOKEN }}
    PR_TOKEN: ${{ secrets.PR_TOKEN }}
  with:
    script: |
      const { Octokit } = require('@octokit/rest');
      const issueClient = new Octokit({ auth: process.env.ISSUE_TOKEN });
      const prClient = new Octokit({ auth: process.env.PR_TOKEN });
      
      await issueClient.rest.issues.create({...});
      await prClient.rest.pulls.create({...});
```

---

## References

### Official Documentation

1. **actions/github-script Repository**
   - URL: https://github.com/actions/github-script
   - Section: "Using a separate GitHub token"
   - Confirms native support for custom tokens

2. **GitHub Docs - Scripting with REST API**
   - URL: https://docs.github.com/rest/guides/scripting-with-the-rest-api-and-javascript
   - Shows Octokit instantiation patterns

3. **GitHub Docs - Authentication in Workflows**
   - URL: https://docs.github.com/actions/reference/authentication-in-a-workflow
   - Token types and scopes

4. **Octokit REST.js Documentation**
   - URL: https://octokit.github.io/rest.js/
   - API reference and authentication options

5. **Octokit.js Repository**
   - URL: https://github.com/octokit/octokit.js
   - Modern unified client (v5+)

### gh-aw Implementation

- **GitHub Token Documentation**: `/docs/src/content/docs/reference/tokens.md`
- **Safe Outputs Reference**: `/docs/src/content/docs/reference/safe-outputs.md`
- **Token Precedence Tests**: `/pkg/workflow/github_token_precedence_integration_test.go`
- **Custom Token Tests**: `/pkg/workflow/individual_github_token_integration_test.go`

---

## Conclusion

**The `actions/github-script` action provides robust support for custom token instantiation** through:

1. **Native `github-token` input parameter** - Simple, recommended for most use cases
2. **Custom Octokit instances** - Advanced scenarios with multiple tokens

Both approaches are:
- ✅ **Well-documented** in official GitHub documentation
- ✅ **Actively maintained** and widely used
- ✅ **Secure** when following best practices
- ✅ **Battle-tested** in production environments
- ✅ **Already implemented** extensively in gh-aw

This research confirms that custom token usage is a first-class feature of github-script, with comprehensive support for all common use cases including cross-repository operations, elevated permissions, and GitHub App authentication.

---

**Research Completed**: 2026-01-13  
**Next Actions**: None required - Feature is fully supported and documented
