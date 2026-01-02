# ValidationContext Migration Plan

## Overview

This specification provides a detailed migration plan for transitioning all 22 validation files from the fail-fast pattern to the error aggregation pattern using ValidationContext. The migration is designed to be incremental, allowing validators to be updated one at a time while maintaining backward compatibility throughout the process.

## Current State

**Infrastructure**: ✅ Complete (3 commits)
- `pkg/workflow/validation_context.go` - Core ValidationContext struct with error/warning collection
- `pkg/workflow/validation_context_test.go` - Comprehensive test coverage (35+ tests)
- `pkg/workflow/compiler.go` - Demonstration method `runPreCompileValidations()`

**Migrated Validators**: 3/22 (14%)
- ✅ `features_validation.go` - Feature flag validation
- ✅ `sandbox_validation.go` - Sandbox configuration validation  
- ✅ `strict_mode_validation.go` - Strict mode security validation

**Remaining Validators**: 19/22 (86%)

## Migration Pattern

### Old Pattern (Fail-Fast)
```go
func validateSomething(data *WorkflowData) error {
    if /* invalid */ {
        return fmt.Errorf("validation failed: %s", reason)
    }
    return nil
}
```

### New Pattern (Error Aggregation)
```go
// Keep legacy function for backward compatibility
func validateSomething(data *WorkflowData) error {
    if /* invalid */ {
        return fmt.Errorf("validation failed: %s", reason)
    }
    return nil
}

// Add new context-aware function
func validateSomethingWithContext(ctx *ValidationContext, data *WorkflowData) {
    if /* invalid */ {
        ctx.AddError("validator_name", fmt.Errorf("validation failed: %s", reason))
    }
}
```

## Validation Phases

Validators are organized by the phase in which they should run:

### Phase 1: ParseTime
Validation during markdown parsing (frontmatter, syntax)

**Validators**: None explicitly in this phase (handled by parser)

### Phase 2: PreCompile
Validation before YAML generation (configuration, features)

**Migrated** (3):
- ✅ `features_validation.go` - Feature flags
- ✅ `sandbox_validation.go` - Sandbox configuration
- ✅ `strict_mode_validation.go` - Strict mode policies

**To Migrate** (9):
- ⏭️ `engine_validation.go` - AI engine configuration
- ⏭️ `mcp_config_validation.go` - MCP server configuration
- ⏭️ `gateway_validation.go` - Gateway port validation
- ⏭️ `firewall_validation.go` - Firewall configuration
- ⏭️ `secrets_validation.go` - Secret references
- ⏭️ `agent_validation.go` - Agent file validation
- ⏭️ `bundler_runtime_validation.go` - JavaScript runtime validation
- ⏭️ `bundler_safety_validation.go` - JavaScript bundle safety
- ⏭️ `bundler_script_validation.go` - JavaScript script validation

### Phase 3: PostYAMLGeneration
Validation after YAML is generated (expression sizes, schema)

**To Migrate** (7):
- ⏭️ `expression_validation.go` - Expression safety and size limits
- ⏭️ `schema_validation.go` - GitHub Actions schema validation
- ⏭️ `runtime_validation.go` - Runtime packages and containers
- ⏭️ `docker_validation.go` - Docker image validation
- ⏭️ `npm_validation.go` - NPM package validation
- ⏭️ `pip_validation.go` - Python package validation
- ⏭️ `step_order_validation.go` - Step ordering validation

### Phase 4: PreEmit
Final validation before writing lock file

**To Migrate** (3):
- ⏭️ `template_validation.go` - Template structure validation
- ⏭️ `compiler_filters_validation.go` - Filter validation
- ⏭️ `repository_features_validation.go` - Repository capabilities

## Migration Priorities

### Priority 1: High-Impact (PreCompile Phase)
These validators catch configuration errors early and benefit most from aggregation:

1. **engine_validation.go** (120 lines)
   - Complex: Multiple engine types, version checks
   - Impact: High - catches engine configuration errors
   - Estimated effort: 2-3 hours

2. **mcp_config_validation.go** (283 lines)
   - Complex: MCP server configuration, network settings
   - Impact: High - complex validation with many rules
   - Estimated effort: 4-5 hours

3. **secrets_validation.go** (28 lines)
   - Simple: Secret reference validation
   - Impact: Medium - validates secret syntax
   - Estimated effort: 30 minutes

4. **agent_validation.go** (273 lines)
   - Complex: Agent file validation, custom agent support
   - Impact: Medium - validates agent configuration
   - Estimated effort: 3-4 hours

### Priority 2: Medium-Impact (PostYAMLGeneration Phase)
These validators check generated YAML and external resources:

5. **expression_validation.go** (253 lines)
   - Complex: Expression safety, size limits
   - Impact: High - prevents runtime failures
   - Estimated effort: 3-4 hours

6. **schema_validation.go** (208 lines)
   - Complex: JSON schema validation with caching
   - Impact: High - ensures GitHub Actions compatibility
   - Estimated effort: 3-4 hours

7. **runtime_validation.go** (299 lines)
   - Complex: Container images, packages, expression sizes
   - Impact: High - validates external resources
   - Estimated effort: 4-5 hours

8. **docker_validation.go** (130 lines)
   - Medium: Docker image existence checks
   - Impact: Medium - external API validation
   - Estimated effort: 2 hours

9. **npm_validation.go** (88 lines)
   - Simple: NPM package validation
   - Impact: Low - external API validation
   - Estimated effort: 1 hour

10. **pip_validation.go** (179 lines)
    - Medium: Python package validation
    - Impact: Low - external API validation
    - Estimated effort: 2 hours

### Priority 3: Low-Impact (Supporting Validators)
These validators handle specific edge cases:

11. **step_order_validation.go** (186 lines)
    - Medium: Step ordering constraints
    - Impact: Low - specific workflow patterns
    - Estimated effort: 2-3 hours

12. **bundler_runtime_validation.go** (171 lines)
    - Medium: JavaScript runtime compatibility
    - Impact: Low - bundler-specific checks
    - Estimated effort: 2 hours

13. **bundler_safety_validation.go** (233 lines)
    - Medium: JavaScript require/module checks
    - Impact: Low - bundler security
    - Estimated effort: 2-3 hours

14. **bundler_script_validation.go** (148 lines)
    - Medium: JavaScript script content validation
    - Impact: Low - bundler content checks
    - Estimated effort: 2 hours

15. **firewall_validation.go** (30 lines)
    - Simple: Firewall log-level validation
    - Impact: Low - configuration enum check
    - Estimated effort: 30 minutes

16. **gateway_validation.go** (24 lines)
    - Simple: Gateway port validation
    - Impact: Low - port range check
    - Estimated effort: 30 minutes

17. **template_validation.go** (76 lines)
    - Simple: Template structure validation
    - Impact: Low - template-specific checks
    - Estimated effort: 1 hour

18. **compiler_filters_validation.go** (106 lines)
    - Medium: Filter validation
    - Impact: Low - filter-specific checks
    - Estimated effort: 1-2 hours

19. **repository_features_validation.go** (334 lines)
    - Complex: Repository capability detection
    - Impact: Medium - GitHub API validation
    - Estimated effort: 4-5 hours

## Incremental Migration Strategy

### Week 1-2: Priority 1 Validators (PreCompile)
**Goal**: Migrate high-impact configuration validators

**Tasks**:
1. Migrate `secrets_validation.go` (simple, good warmup)
2. Migrate `engine_validation.go` (complex, high value)
3. Migrate `mcp_config_validation.go` (complex, high value)
4. Migrate `agent_validation.go` (complex, medium value)
5. Update `compiler.go` to use context for PreCompile phase

**Deliverable**: PreCompile phase fully migrated with error aggregation

### Week 3-4: Priority 2 Validators (PostYAMLGeneration)
**Goal**: Migrate YAML generation and external resource validators

**Tasks**:
1. Migrate `expression_validation.go` (critical for size limits)
2. Migrate `schema_validation.go` (GitHub Actions compatibility)
3. Migrate `runtime_validation.go` (external resources)
4. Migrate `docker_validation.go`, `npm_validation.go`, `pip_validation.go` (external APIs)
5. Update `compiler.go` to use context for PostYAMLGeneration phase

**Deliverable**: PostYAMLGeneration phase fully migrated

### Week 5: Priority 3 Validators (Supporting)
**Goal**: Migrate remaining validators and finalize migration

**Tasks**:
1. Migrate all remaining validators (bundler, firewall, gateway, etc.)
2. Update `compiler.go` to use context for all phases
3. Add comprehensive integration tests
4. Update all documentation

**Deliverable**: Complete migration with all validators using ValidationContext

### Week 6: Cleanup and Optimization
**Goal**: Remove legacy patterns and optimize

**Tasks**:
1. Remove legacy validation functions (keeping only *WithContext variants)
2. Optimize error reporting and formatting
3. Add performance benchmarks
4. Final documentation review
5. Migration complete announcement

## Implementation Checklist

For each validator migration:

### 1. Analysis Phase
- [ ] Review validator function signatures
- [ ] Identify all validation checks
- [ ] Document error messages and formatting
- [ ] Check for external dependencies (API calls, file I/O)
- [ ] Identify test coverage

### 2. Implementation Phase
- [ ] Add `validateXWithContext()` function
- [ ] Convert error returns to `ctx.AddError()` calls
- [ ] Convert early returns to continue execution
- [ ] Add validator name to all error calls
- [ ] Preserve error message quality

### 3. Testing Phase
- [ ] Add unit tests for context-aware function
- [ ] Test error aggregation with multiple failures
- [ ] Verify backward compatibility (legacy function still works)
- [ ] Test phase assignment
- [ ] Add integration test if needed

### 4. Integration Phase
- [ ] Update `compiler.go` phase orchestration
- [ ] Add context-aware call in appropriate phase
- [ ] Keep legacy call for backward compatibility
- [ ] Test with real workflows
- [ ] Update documentation

### 5. Verification Phase
- [ ] Run full test suite
- [ ] Run linter
- [ ] Test with multiple error scenarios
- [ ] Verify error messages are clear
- [ ] Check performance impact

## Compiler Orchestration Update

The `compiler.go` file needs to be updated to orchestrate validation using ValidationContext. Here's the pattern:

### Current Pattern
```go
// PreCompile validations (fail-fast)
if err := validateFeatures(workflowData); err != nil {
    return formatError(err)
}
if err := validateSandboxConfig(workflowData); err != nil {
    return formatError(err)
}
// ... more validators
```

### Target Pattern
```go
// Create validation context
ctx := NewValidationContext(markdownPath, workflowData)

// Phase 1: PreCompile validations (aggregate errors)
ctx.SetPhase(PhasePreCompile)
validateFeaturesWithContext(ctx, workflowData)
validateSandboxConfigWithContext(ctx, workflowData)
validateEngineWithContext(ctx, workflowData)
validateMCPConfigWithContext(ctx, workflowData)
// ... all PreCompile validators

if ctx.HasErrors() {
    return errors.New(ctx.Error())  // Report all errors together
}

// Phase 2: PostYAMLGeneration validations
ctx.SetPhase(PhasePostYAMLGeneration)
ctx.SetYAMLContent(yamlContent)
validateExpressionSizesWithContext(ctx, yamlContent)
validateSchemaWithContext(ctx, yamlContent)
// ... all PostYAMLGeneration validators

if ctx.HasErrors() {
    return errors.New(ctx.Error())
}

// Phase 3: PreEmit validations
ctx.SetPhase(PhasePreEmit)
validateFileSizeWithContext(ctx, lockFile)
// ... all PreEmit validators

if ctx.HasErrors() {
    return errors.New(ctx.Error())
}
```

## Testing Strategy

### Unit Tests
Each migrated validator should have tests for:
- Error aggregation with multiple failures
- Warning collection
- Phase tracking
- Backward compatibility

### Integration Tests
- Test full compilation with multiple validation errors
- Verify all errors are collected and reported
- Test phase progression
- Verify error message formatting

### Performance Tests
- Benchmark validation time before/after migration
- Ensure error aggregation doesn't significantly slow compilation
- Test with large workflows

## Documentation Updates

### Files to Update
1. **specs/validation-architecture.md**
   - Add ValidationContext section
   - Document migration pattern
   - Update validator list with migration status

2. **pkg/workflow/validation.go**
   - Update package documentation
   - Add migration examples
   - Document both patterns

3. **AGENTS.md**
   - ✅ Already updated with ValidationContext section
   - Update migration progress as validators are completed

4. **README.md**
   - Add note about error aggregation improvement
   - Link to validation documentation

## Success Metrics

### Developer Experience
- **Before**: Fix one error → recompile → discover next error → repeat
- **After**: See all errors in one compilation → fix all → success

### Quantitative Metrics
- Average compilation cycles to fix all errors: 4-5 → 1
- Time to discover all validation issues: 5-10 minutes → 30 seconds
- Developer satisfaction: Measure via surveys

### Quality Metrics
- All validators migrated: 3/22 → 22/22
- Test coverage maintained: ≥90%
- Error message quality: No degradation
- Compilation time impact: <5% increase

## Risk Mitigation

### Backward Compatibility
- Keep legacy functions during migration
- Add deprecation warnings after full migration
- Remove legacy functions only after transition period

### Performance
- Monitor compilation time
- Optimize error collection if needed
- Consider lazy error formatting

### Error Message Quality
- Follow specs/error-messages.md guidelines
- Review all error messages during migration
- Test with real-world scenarios

## Timeline Summary

| Week | Phase | Validators | Effort |
|------|-------|-----------|---------|
| 1-2 | Priority 1 | 4 validators | PreCompile phase complete |
| 3-4 | Priority 2 | 7 validators | PostYAMLGeneration phase complete |
| 5 | Priority 3 | 9 validators | All validators migrated |
| 6 | Cleanup | Optimization | Migration complete |

**Total Estimated Effort**: 6 weeks with 1 developer
**Current Progress**: Week 0 complete (infrastructure + 3 validators)

## Next Steps

1. **Immediate (Next PR)**:
   - Migrate `secrets_validation.go` (simple warmup)
   - Migrate `engine_validation.go` (high value)
   - Add integration test for multi-error scenarios

2. **Short-term (Next 2 weeks)**:
   - Complete Priority 1 validators
   - Update compiler.go PreCompile phase orchestration
   - Add performance benchmarks

3. **Medium-term (Next month)**:
   - Complete Priority 2 validators
   - Update compiler.go for all phases
   - Comprehensive documentation update

4. **Long-term (Next 6 weeks)**:
   - Complete all validators
   - Remove legacy patterns
   - Announce migration complete

## References

- **Implementation**: `pkg/workflow/validation_context.go`
- **Tests**: `pkg/workflow/validation_context_test.go`
- **Example**: `pkg/workflow/features_validation.go` (migrated)
- **Architecture**: `specs/validation-architecture.md`
- **Error Messages**: `specs/error-messages.md`
- **Testing Guide**: `specs/testing.md`

## Conclusion

This migration plan provides a structured approach to transitioning all validators to the error aggregation pattern. The incremental strategy allows for careful migration while maintaining backward compatibility and code quality. The end result will significantly improve developer experience by showing all validation issues in a single compilation run.

---
**Status**: Phase 0 Complete (Infrastructure + 3 validators)  
**Last Updated**: 2026-01-02  
**Owner**: @copilot  
**Related**: Issue #8633
