# Unauthenticated REST API Fallback Implementation

## Summary

Added fallback code paths in `pkg/parser/remote_fetch.go` to fetch remote workflow files using GitHub's public REST API without authentication when `gh` CLI authentication fails.

## Changes

### 1. Modified `resolveRefToSHA` function
- Detects authentication failures from `gh` CLI
- Falls back to `resolveRefToSHAUnauthenticated` on auth failures
- Logs the fallback attempt for debugging

### 2. Modified `downloadFileFromGitHub` function  
- Detects authentication failures from `gh` CLI
- Falls back to `downloadFileFromGitHubUnauthenticated` on auth failures
- Logs the fallback attempt for debugging

### 3. Added `resolveRefToSHAUnauthenticated` function
- Uses `curl` to call GitHub's public REST API without authentication
- Endpoint: `https://api.github.com/repos/{owner}/{repo}/commits/{ref}`
- Proper JSON parsing using `encoding/json`
- Error handling for Not Found, rate limits, and other API errors
- SHA validation (40 hex characters)

### 4. Added `downloadFileFromGitHubUnauthenticated` function
- Uses `curl` to call GitHub's public REST API without authentication
- Endpoint: `https://api.github.com/repos/{owner}/{repo}/contents/{path}?ref={ref}`
- Proper JSON parsing using `encoding/json`
- Handles base64 decoding of file content
- Error handling for Not Found, rate limits, and other API errors

### 5. Added unit tests
- `TestJSONParsing` - Tests JSON response parsing for all scenarios
- Tests cover success cases, error responses, and edge cases

## Behavior

### With GH_TOKEN (authenticated)
1. Uses `gh` CLI commands (existing behavior)
2. Has access to private repositories
3. Higher rate limits

### Without GH_TOKEN (unauthenticated fallback)
1. Detects authentication failure from `gh` CLI
2. Falls back to public REST API via `curl`
3. Only works with public repositories
4. Subject to GitHub's unauthenticated rate limits (60 requests/hour per IP)

## Import Caching

The import cache mechanism (`.github/aw/imports/`) continues to work with the fallback:
- Downloaded files are cached by SHA in `.github/aw/imports/{owner}/{repo}/{sha}/{filename}`
- Subsequent compilations use the cache, avoiding repeated API calls
- Cache persists across workflow runs

## Testing

### Unit Tests
```bash
# From the repository root
go test ./pkg/parser -run TestJSONParsing -v
```

### Manual Testing (when network is available)
```bash
# Remove GH_TOKEN to simulate unauthenticated environment
unset GH_TOKEN
unset GITHUB_TOKEN

# Test compile with remote import
./gh-aw compile /path/to/workflow-with-remote-import.md

# Verify cache directory was created
ls -la .github/aw/imports/
```

## Rate Limits

GitHub's unauthenticated API has strict rate limits:
- 60 requests per hour per IP address
- Rate limit headers are not currently parsed
- Users should use GH_TOKEN for production workflows

## Future Improvements

1. Parse rate limit headers and warn users
2. Add retry logic with exponential backoff
3. Support for GitHub Enterprise endpoints
4. Cache rate limit information to avoid hitting limits

## Compatibility

- Works with all existing workflows
- Backward compatible - authenticated mode is still preferred
- Only public repositories are accessible without authentication
- Private repository imports will still require authentication
