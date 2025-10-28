# Wildcard Network Filtering Implementation Status

## Summary

This document provides a comprehensive analysis of wildcard network filtering support in GitHub Agentic Workflows.

**Key Finding**: Wildcard support varies by engine:
- **Claude Engine**: ✅ Fully supported via Python hooks
- **Copilot Engine with AWF**: ❌ NOT supported (AWF doesn't support wildcard syntax)

## Current Implementation Status

### ✅ Claude Engine - FULLY IMPLEMENTED

Wildcard patterns are **fully implemented and tested** for the Claude engine via Python network permission hooks.

**Implementation Details:**
- **File**: `pkg/workflow/engine_network_hooks.go` (lines 61-75)
- **Pattern Conversion**: `pattern.replace('.', r'\.').replace('*', '.*')`
- **Matching Logic**: `re.match(f'^{regex}$', domain)`
- **Test Coverage**: `TestWildcardNetworkPermissionsClaudeEngine`

**Verified Behavior:**
- ✅ `*.example.com` matches `api.example.com` (single-level subdomain)
- ✅ `*.example.com` matches `nested.api.example.com` (multi-level subdomain)
- ✅ `*.example.com` does NOT match `example.com` (base domain)
- ✅ `*.example.com` does NOT match `notexample.com` (partial suffix)

### ❌ Copilot Engine with AWF - NOT SUPPORTED

AWF (Agent Workflow Firewall) **does NOT support wildcard syntax** according to its [official documentation](https://github.com/githubnext/gh-aw-firewall/blob/main/docs/QUICKSTART.md#limitations):

> **✗ No wildcard syntax** (use base domain instead)  
> `--allow-domains '*.github.com'`  
> `--allow-domains github.com        # ✓ matches subdomains automatically`

**Current gh-aw Behavior (INCORRECT):**
- **File**: `pkg/workflow/copilot_engine.go` (line 218)
- **What Happens**: gh-aw passes `*.example.com` to AWF via `--allow-domains`
- **Problem**: AWF doesn't support wildcard syntax and will reject or ignore it
- **Impact**: Wildcards in network configuration for Copilot engine won't work as expected

**How AWF Actually Works:**
- AWF automatically matches subdomains for base domains
- `example.com` in AWF matches: `api.example.com`, `nested.api.example.com`, etc.
- No wildcard prefix (`*.`) is needed or supported

**What is Verified:**
- ✅ gh-aw extracts wildcards from workflow configuration
- ✅ gh-aw includes wildcards in domain list
- ✅ gh-aw passes wildcards to AWF via `--allow-domains` flag

**What is BROKEN:**
- ❌ AWF does not support wildcards
- ❌ Users expecting wildcard behavior with Copilot will be confused
- ❌ No validation or warning when wildcards are used with Copilot

## Security Guide Accuracy

The security guide (`docs/src/content/docs/guides/security.md`) previously stated:

> **Use Wildcards Carefully**: `*.example.com` matches any subdomain including nested ones

**Original Assessment**: INACCURATE for Copilot engine

**Corrected Version** (line 496):
> **Domain Matching Behavior**:
>    - **Claude engine**: Supports wildcard syntax - `*.example.com` matches any subdomain
>    - **Copilot engine with AWF**: Does NOT support wildcard syntax. Use base domain instead (e.g., `example.com` auto-matches subdomains)

**Assessment:**
- ✅ **Now accurate** - distinguishes between Claude and Copilot behavior
- ✅ **Helpful** - explains AWF's subdomain auto-matching feature

## Required Fixes

### High Priority

1. **Add Validation for Copilot/AWF**
   - Detect wildcards in network configuration when using Copilot engine
   - Issue warning or error message
   - Suggest using base domain instead

2. **Auto-Convert Wildcards** (Optional)
   - Strip `*.` prefix when compiling for AWF
   - Convert `*.example.com` → `example.com` automatically
   - Add compiler message explaining the conversion

3. **Update Tests**
   - Current `TestWildcardNetworkPermissionsCopilotEngine` is misleading
   - Only verifies wildcards are passed, not that they work
   - Should either be removed or updated to test the actual (non-working) behavior

### Documentation

- [x] Security guide updated to distinguish Claude vs Copilot
- [x] Investigation summary updated with correct findings
- [ ] Add examples showing correct usage per engine
- [ ] Update user-facing docs with clear guidance

## Recommendations

### For Claude Engine Users
```yaml
engine: claude
network:
  allowed:
    - "*.example.com"  # ✓ Wildcard syntax works
    - "api.github.com"  # ✓ Exact domain works
```

Wildcards work as documented. No changes needed.

### For Copilot Engine Users

**INCORRECT (Won't Work):**
```yaml
engine: copilot
network:
  firewall: true
  allowed:
    - "*.example.com"  # ✗ AWF doesn't support wildcards
```

**CORRECT:**
```yaml
engine: copilot
network:
  firewall: true
  allowed:
    - "example.com"  # ✓ Auto-matches api.example.com, etc.
    - "api.github.com"  # ✓ Exact domain
```

**Note**: AWF automatically matches subdomains, so `example.com` will allow `api.example.com`, `nested.api.example.com`, etc.

### For gh-aw Maintainers

1. **Immediate Action**: Add compiler warning when wildcards detected with Copilot/AWF
2. **Consider**: Auto-strip `*.` prefix for AWF compatibility
3. **Testing**: Add integration test with actual AWF to verify behavior
4. **Documentation**: Ensure all examples use correct syntax per engine

## Test Coverage

### Valid Tests
- ✅ `TestWildcardNetworkPermissionsClaudeEngine` - Correctly verifies Claude wildcard support
- ✅ `TestWildcardDomainMatching` - Documents expected behavior (for Claude)
- ✅ `TestWildcardSecurityGuideAccuracy` - Verifies documentation exists

### Problematic Tests
- ⚠️ `TestWildcardNetworkPermissionsCopilotEngine` - Misleading test that only verifies wildcards are passed to AWF, not that they work

## Conclusion

**Original Problem Statement Was PARTIALLY CORRECT:**

The security guide claimed wildcards work, but this is only true for Claude engine. For Copilot engine with AWF, wildcards are NOT supported.

**Summary:**
- ✅ Claude engine: Wildcards fully implemented
- ❌ Copilot/AWF: Wildcards NOT supported (AWF limitation)
- ⚠️ gh-aw: Passes wildcards to AWF without validation (bug)
- ✅ Documentation: Now corrected to distinguish between engines

**Next Steps:**
1. Add validation/warning for wildcards with Copilot
2. Update or remove misleading test
3. Consider auto-conversion of wildcards for AWF
4. Update user-facing documentation with clear examples

