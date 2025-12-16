# Feature Specification: Workflow Validation Report

## Overview

Add a validation report feature that analyzes agentic workflow markdown files and generates a comprehensive validation report. The report highlights potential issues, security concerns, and best practice violations to help users create safe and effective workflows.

This feature improves workflow quality by providing actionable feedback during development, reducing the likelihood of security issues or configuration errors in production.

## User Stories

### User Story 1: Validate Workflow Configuration

**As a** workflow developer  
**I want** to validate my workflow configuration file  
**So that** I can identify issues before deployment

**Acceptance Criteria:**
- Command accepts a workflow markdown file path
- Validates frontmatter configuration
- Checks for common security issues
- Reports results in a clear format
- Returns appropriate exit codes (0 for valid, 1 for errors)

### User Story 2: Security Best Practices Check

**As a** security-conscious developer  
**I want** to check if my workflow follows security best practices  
**So that** I can ensure the workflow operates safely

**Acceptance Criteria:**
- Detects overly broad permissions
- Identifies missing safe-output configurations
- Flags unvalidated external inputs
- Warns about unrestricted network access
- Provides recommendations for fixing issues

### User Story 3: Detailed Validation Report

**As a** workflow maintainer  
**I want** to see a detailed validation report  
**So that** I can understand and address all issues

**Acceptance Criteria:**
- Report includes severity levels (error, warning, info)
- Each issue has a clear description
- Recommendations include examples
- Report is formatted for readability
- Can output in multiple formats (console, JSON)

### User Story 4: CI/CD Integration

**As a** CI/CD engineer  
**I want** to integrate validation into automated testing  
**So that** workflows are validated automatically

**Acceptance Criteria:**
- Command supports JSON output for parsing
- Exit codes indicate validation status
- Works in non-interactive environments
- Can validate multiple files in batch
- Performance is suitable for CI pipelines

## Requirements

### Functional Requirements

- **FR-1**: Accept workflow markdown file path(s) as input
- **FR-2**: Parse frontmatter YAML configuration
- **FR-3**: Validate workflow configuration against schema
- **FR-4**: Check for security best practices violations
- **FR-5**: Generate validation report with issues and recommendations
- **FR-6**: Support multiple output formats (console, JSON)
- **FR-7**: Return exit code 0 for valid workflows, 1 for errors
- **FR-8**: Support batch validation of multiple files
- **FR-9**: Display file location for each issue
- **FR-10**: Include severity levels for all issues

### Non-Functional Requirements

- **NFR-1**: Validation completes in under 2 seconds for typical workflows
- **NFR-2**: Reports follow console formatting standards (pkg/console)
- **NFR-3**: Code follows Go best practices and repository patterns
- **NFR-4**: Unit test coverage of 80% or higher
- **NFR-5**: Compatible with existing workflow compilation pipeline
- **NFR-6**: Clear error messages for invalid inputs
- **NFR-7**: Memory efficient for large workflow files

### Security Requirements

- **SR-1**: Detect workflows with `permissions: write` without safe-outputs
- **SR-2**: Flag workflows with unrestricted network access
- **SR-3**: Identify workflows accepting unvalidated external inputs
- **SR-4**: Warn about workflows with overly broad repository permissions
- **SR-5**: Check for hardcoded secrets or credentials

### Quality Requirements

- **QR-1**: Follow TDD approach (tests before implementation)
- **QR-2**: Use table-driven tests for validation rules
- **QR-3**: Include integration tests for CLI command
- **QR-4**: Code comments explain validation logic
- **QR-5**: Documentation includes usage examples

## Validation Rules

### Security Validation Rules

1. **Broad Permissions**: Workflows with `permissions: write` must use safe-outputs
2. **Network Access**: Network access should be domain-restricted
3. **External Inputs**: Inputs from webhooks must be validated
4. **Secret Handling**: No hardcoded secrets in workflow files
5. **Tool Access**: Tools should be explicitly declared

### Configuration Validation Rules

1. **Schema Compliance**: Frontmatter must match workflow schema
2. **Engine Support**: Engine must be supported (copilot, claude, codex)
3. **Trigger Configuration**: Triggers must be valid GitHub Actions triggers
4. **Tool Configuration**: Tool settings must be valid for the tool
5. **Output Format**: Safe-outputs must follow approved patterns

### Best Practice Validation Rules

1. **Descriptive Title**: Workflow should have clear, descriptive title
2. **Documentation**: Complex workflows should include explanatory comments
3. **Error Handling**: Workflows should handle expected error cases
4. **Resource Limits**: Long-running workflows should have timeouts
5. **Minimal Permissions**: Use least-privilege permission model

## Success Metrics

### Adoption Metrics
- Validation command is run by 50%+ of users before first deployment
- 80%+ of workflows pass validation without errors

### Quality Metrics
- 30% reduction in workflow configuration errors reported
- 50% reduction in security-related workflow issues
- Average resolution time for validation issues < 10 minutes

### Performance Metrics
- Validation completes in < 2 seconds for 95% of workflows
- Memory usage < 50MB for typical workflows
- Zero false positives in security rule detection

## Out of Scope

The following are explicitly out of scope for this feature:

- **Automatic Fixing**: Tool will not automatically fix issues
- **Runtime Validation**: Only validates configuration, not runtime behavior
- **Custom Rules**: Users cannot add custom validation rules (future enhancement)
- **Workflow Testing**: Does not execute workflows for testing
- **Historical Analysis**: Does not analyze past workflow runs

## Dependencies

### Internal Dependencies
- Existing workflow parser (`pkg/parser/`)
- Workflow schema definitions (`pkg/parser/schemas/`)
- Console formatting utilities (`pkg/console/`)
- CLI framework (`pkg/cli/`)

### External Dependencies
- None (uses only standard library and existing packages)

## Future Enhancements

Potential future enhancements not included in this initial implementation:

1. **Custom Rules**: Allow users to define organization-specific rules
2. **Auto-Fix**: Suggest and apply automatic fixes for common issues
3. **CI Integration**: GitHub App for automatic PR validation
4. **Historical Analysis**: Analyze workflow run history for patterns
5. **Performance Profiling**: Estimate workflow execution time and resource usage

## Acceptance Checklist

The feature is considered complete when:

- [ ] All functional requirements are implemented
- [ ] All security validation rules are working
- [ ] Unit test coverage ≥ 80%
- [ ] Integration tests for CLI command pass
- [ ] Documentation is complete with examples
- [ ] `make agent-finish` passes without errors
- [ ] Manual testing validates all user stories
- [ ] Code review feedback is addressed
