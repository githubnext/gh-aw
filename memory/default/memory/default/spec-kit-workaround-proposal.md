# Spec-Kit Execute Workaround Proposal

## Problem Summary
The spec-kit-execute workflow cannot create new directories under `pkg/` due to bash command whitelist restrictions. This blocks implementation of specs that require new directory creation.

## Immediate Workaround for 001-test-feature

Since we cannot create `pkg/test/`, we can:

1. **Modify the spec to use existing `pkg/testutil/`**
   - Update `plan.md` to target `pkg/testutil/test_feature.go`
   - Update `tasks.md` accordingly
   - Implement the test feature in the existing directory

2. **Document this as a limitation**
   - Note in the PR that directory creation is not supported
   - Recommend workflow configuration update

## Implementation Plan

### Modified Tasks for 001-test-feature:

**Phase 1: Setup**
- [x] 1.1: ~~Create `pkg/test/` directory~~ - SKIPPED (use existing pkg/testutil/)
- [ ] 1.2: Create `pkg/testutil/test_feature.go` file

**Phase 2: Tests (TDD)**
- [ ] 2.1: Create `pkg/testutil/test_feature_test.go` file  
- [ ] 2.2: Write test for basic functionality

**Phase 3: Core Implementation**
- [ ] 3.1: Implement basic test function
- [ ] 3.2: Ensure tests pass

**Phase 4: Validation**
- [ ] 4.1: Run `make fmt`
- [ ] 4.2: Run `make lint`
- [ ] 4.3: Run `make test-unit`

## Long-term Solution

Update `.github/workflows/spec-kit-execute.md` to include:
```yaml
bash:
  - "mkdir -p pkg/**"
```

This would allow directory creation under pkg/ while maintaining security.

## Date
2025-12-12
