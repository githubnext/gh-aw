# Implementation Plan: Workflow Validation Report

## Technical Approach

Implement a workflow validation system using Go that analyzes workflow markdown files and generates comprehensive validation reports. The system will leverage existing workflow parsing infrastructure and extend it with validation rules and reporting capabilities.

The validation engine will use a rule-based architecture where each rule independently checks for specific issues. Rules are organized by category (security, configuration, best practices) and can be executed in parallel for performance.

## Technology Stack

- **Language**: Go (following repository standards)
- **Testing**: Standard Go testing framework with table-driven tests
- **Location**: `pkg/workflow/validator/` for validation logic
- **CLI Integration**: `cmd/gh-aw/validate_command.go` for command implementation
- **Output**: Console formatting via `pkg/console/`, JSON via `encoding/json`

## Architecture

### Component Design

```
pkg/workflow/validator/
├── validator.go           # Main validator interface and orchestration
├── validator_test.go      # Comprehensive unit tests
├── rules.go               # Validation rule definitions
├── rules_test.go          # Rule-specific tests
├── security.go            # Security-focused validation rules
├── security_test.go       # Security rule tests
├── config.go              # Configuration validation rules
├── config_test.go         # Configuration rule tests
├── report.go              # Report generation and formatting
├── report_test.go         # Report formatting tests
└── types.go               # Shared types and interfaces

cmd/gh-aw/
└── validate_command.go    # CLI command implementation
```

### Key Components

#### 1. Validator (`validator.go`)

The main orchestration component that:
- Accepts workflow file paths
- Parses workflow markdown using existing parser
- Executes validation rules
- Aggregates results into a report
- Returns appropriate exit codes

```go
type Validator struct {
    rules []ValidationRule
}

type ValidationRule interface {
    Name() string
    Severity() Severity
    Validate(workflow *Workflow) []Issue
}

type Issue struct {
    Rule        string
    Severity    Severity
    Message     string
    Location    string
    Recommendation string
}
```

#### 2. Rules Engine (`rules.go`, `security.go`, `config.go`)

Implements validation rules organized by category:

**Security Rules**:
- Overly broad permissions detection
- Missing safe-output validation
- Unvalidated external input detection
- Unrestricted network access warnings
- Hardcoded secret detection

**Configuration Rules**:
- Schema compliance validation
- Engine support verification
- Trigger configuration validation
- Tool configuration verification
- Output format validation

**Best Practice Rules**:
- Descriptive title checks
- Documentation presence
- Error handling patterns
- Resource limit recommendations
- Minimal permission suggestions

#### 3. Report Generator (`report.go`)

Generates formatted reports in multiple formats:
- **Console**: Uses `pkg/console` formatting for styled output
- **JSON**: Machine-readable format for CI/CD integration

Report structure:
```go
type Report struct {
    FilePath string
    Valid    bool
    Issues   []Issue
    Summary  ReportSummary
}

type ReportSummary struct {
    TotalIssues int
    Errors      int
    Warnings    int
    Infos       int
}
```

#### 4. CLI Command (`validate_command.go`)

Cobra command implementation:
```go
func NewValidateCommand() *cobra.Command {
    // Command: gh aw validate [file...]
    // Flags: --format (console|json), --strict
}
```

### Data Flow

```
User Input (file paths)
    ↓
CLI Command Parser
    ↓
Validator.Validate()
    ↓
For each file:
    Parse Workflow (existing parser)
    ↓
    Execute Validation Rules (parallel)
    ↓
    Aggregate Issues
    ↓
Report Generator
    ↓
Formatted Output (console/JSON)
    ↓
Exit Code (0=valid, 1=errors)
```

## Implementation Phases

### Phase 1: Setup (Estimated: 30 minutes)

1. Create directory structure: `pkg/workflow/validator/`
2. Create file stubs: `validator.go`, `types.go`, `report.go`
3. Create test file stubs
4. Define interfaces and types
5. Set up CLI command structure in `cmd/gh-aw/`

**Validation**: Directory structure exists, files compile

### Phase 2: Tests (TDD) (Estimated: 2 hours)

1. Write tests for validator orchestration
2. Write tests for security rules
3. Write tests for configuration rules
4. Write tests for report generation
5. Write tests for CLI command
6. All tests should fail initially (not implemented)

**Validation**: Tests exist and fail expectedly

### Phase 3: Core Implementation (Estimated: 3 hours)

1. Implement validator orchestration logic
2. Implement security validation rules
3. Implement configuration validation rules
4. Implement report generator (console format)
5. Implement report generator (JSON format)
6. Ensure all tests pass

**Validation**: `make test-unit` passes, coverage ≥ 80%

### Phase 4: Integration (Estimated: 1.5 hours)

1. Integrate validator with existing workflow parser
2. Implement CLI command with flags
3. Add command to main CLI application
4. Wire up console formatting
5. Add integration tests
6. Test with real workflow files

**Validation**: `make test` passes, manual testing works

### Phase 5: Polish (Estimated: 1 hour)

1. Add comprehensive code comments
2. Update CLI help text and examples
3. Add usage documentation
4. Optimize performance if needed
5. Run `make agent-finish` for final validation

**Validation**: All checks pass, documentation complete

## Dependencies

### Internal Dependencies

- **pkg/parser**: Workflow markdown parsing
- **pkg/workflow**: Workflow types and structures
- **pkg/console**: Console output formatting
- **pkg/cli**: CLI command framework
- **cmd/gh-aw**: Main application integration

### External Dependencies

None - uses only Go standard library:
- `encoding/json`: JSON output
- `fmt`: String formatting
- `os`: File operations
- `path/filepath`: Path handling
- `testing`: Test framework

## Testing Strategy

### Unit Tests

All packages must have comprehensive unit tests:

- **Validator Tests**: Test orchestration, rule execution, error handling
- **Rule Tests**: Table-driven tests for each validation rule
- **Report Tests**: Test formatting, aggregation, output generation
- **CLI Tests**: Test command parsing, flag handling, integration

**Target Coverage**: ≥ 80% for all packages

### Table-Driven Tests

Use table-driven tests for validation rules:

```go
func TestSecurityRules(t *testing.T) {
    tests := []struct {
        name     string
        workflow *Workflow
        wantErr  bool
        issues   int
    }{
        {
            name: "broad permissions without safe-outputs",
            workflow: &Workflow{
                Permissions: "write",
                SafeOutputs: nil,
            },
            wantErr: true,
            issues: 1,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Integration Tests

Test end-to-end workflow validation:

1. Create sample workflow files with known issues
2. Run validation command
3. Verify correct issues are detected
4. Verify report format and content
5. Verify exit codes

### Manual Testing

Test scenarios:
1. Valid workflow with no issues
2. Workflow with security violations
3. Workflow with configuration errors
4. Workflow with multiple issues
5. Batch validation of multiple files
6. JSON output format
7. Error handling (invalid file, missing file, etc.)

## Error Handling

### Input Validation

- Check file exists before parsing
- Validate file is readable
- Verify file is markdown format
- Handle parse errors gracefully

### Rule Execution

- Continue validation even if one rule fails
- Log rule execution errors
- Report rule failures in output

### Output Generation

- Handle write errors for JSON output
- Fall back to console format on JSON errors
- Provide clear error messages

## Performance Considerations

### Optimization Strategies

1. **Parallel Rule Execution**: Run independent rules concurrently
2. **Cached Parsing**: Parse each file once, reuse for all rules
3. **Lazy Loading**: Load validation rules only when needed
4. **Efficient Matching**: Use compiled regex for pattern matching

### Performance Targets

- Validation time: < 2 seconds for typical workflows
- Memory usage: < 50MB per workflow
- Batch processing: Linear scaling with file count

### Profiling

Use Go profiling tools if performance issues arise:
```bash
go test -cpuprofile cpu.prof -memprofile mem.prof -bench .
go tool pprof cpu.prof
```

## Documentation Requirements

### Code Documentation

- Package-level documentation for validator package
- Function-level comments for exported functions
- Inline comments for complex logic
- Example code in package documentation

### User Documentation

**README.md Updates**: Add validation command to main README

**CLI Help Text**:
```bash
gh aw validate [file...] [flags]

Validate agentic workflow configuration files.

Examples:
  # Validate a single workflow
  gh aw validate .github/workflows/my-workflow.md
  
  # Validate multiple workflows
  gh aw validate .github/workflows/*.md
  
  # Output as JSON for CI/CD
  gh aw validate --format=json workflow.md

Flags:
  --format string   Output format: console or json (default "console")
  --strict          Treat warnings as errors
```

**Guide Documentation**: Usage examples in documentation site

### API Documentation

Generate Go documentation:
```bash
go doc pkg/workflow/validator
```

## Security Considerations

### Validation Security

- Do not execute workflow code during validation
- Safely parse untrusted YAML frontmatter
- Prevent path traversal attacks in file handling
- Limit resource usage for large files

### Rule Accuracy

- Minimize false positives in security rules
- Provide clear recommendations for fixing issues
- Document rule rationale in code comments

## Rollout Strategy

### Development

1. Implement in feature branch
2. Run all tests locally
3. Request code review
4. Address feedback

### Testing

1. Test with real workflow files from repository
2. Validate against known good and bad workflows
3. Performance testing with large files
4. CI/CD integration testing

### Deployment

1. Merge to main branch
2. Release in next version
3. Announce in changelog
4. Update documentation site

### Monitoring

Track metrics after deployment:
- Command usage frequency
- Average validation time
- Common issues detected
- False positive reports

## Future Enhancements

Not included in this implementation but planned for future:

1. **Custom Rules**: Plugin system for organization-specific rules
2. **Auto-Fix**: Automatic correction of common issues
3. **GitHub App**: PR validation bot
4. **IDE Integration**: Editor extensions for real-time validation
5. **Rule Configuration**: Allow disabling specific rules

## Acceptance Criteria

Implementation is complete when:

- [ ] All components implemented and tested
- [ ] Unit test coverage ≥ 80%
- [ ] All validation rules working correctly
- [ ] Console and JSON output formats working
- [ ] CLI command integrated and tested
- [ ] Documentation complete with examples
- [ ] `make agent-finish` passes
- [ ] Manual testing validates all scenarios
- [ ] Code review approved
- [ ] No security vulnerabilities introduced
