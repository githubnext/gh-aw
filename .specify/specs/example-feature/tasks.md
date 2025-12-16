# Task Breakdown: Workflow Validation Report

This task breakdown follows the Test-Driven Development (TDD) approach. Tests are written before implementation code, and validation is performed after each phase.

## Phase 1: Setup

### 1.1: Create directory structure
- **Dependencies**: None
- **Acceptance**: `pkg/workflow/validator/` directory exists

### 1.2: Create core type definitions
- **Dependencies**: 1.1
- **File**: `pkg/workflow/validator/types.go`
- **Acceptance**: File exists with package declaration and core types defined:
  - `ValidationRule` interface
  - `Issue` struct
  - `Severity` enum
  - `Report` struct
  - `ReportSummary` struct

### 1.3: Create validator stub
- **Dependencies**: 1.2
- **File**: `pkg/workflow/validator/validator.go`
- **Acceptance**: File exists with `Validator` struct and stub methods

### 1.4: Create test file stubs
- **Dependencies**: 1.1, 1.2, 1.3
- **Files**: 
  - `pkg/workflow/validator/validator_test.go`
  - `pkg/workflow/validator/rules_test.go`
  - `pkg/workflow/validator/security_test.go`
  - `pkg/workflow/validator/report_test.go`
- **Acceptance**: Test files exist with package declaration and import statements

### 1.5: Create CLI command stub
- **Dependencies**: None
- **File**: `cmd/gh-aw/validate_command.go`
- **Acceptance**: File exists with Cobra command stub

**Phase 1 Validation**: Run `make fmt` and `make build` - should compile successfully

---

## Phase 2: Tests (TDD)

### 2.1: Write validator orchestration tests
- **Dependencies**: 1.4
- **File**: `pkg/workflow/validator/validator_test.go`
- **Tests**:
  - `TestValidator_Validate_Success`: Valid workflow passes
  - `TestValidator_Validate_WithIssues`: Detects issues
  - `TestValidator_Validate_MultipleFiles`: Handles batch validation
  - `TestValidator_Validate_FileNotFound`: Handles missing files
- **Acceptance**: Tests compile and fail expectedly (not implemented)

### 2.2: Write security rule tests
- **Dependencies**: 1.4
- **File**: `pkg/workflow/validator/security_test.go`
- **Tests** (table-driven):
  - `TestSecurityRules_BroadPermissions`: Detects `permissions: write` without safe-outputs
  - `TestSecurityRules_UnrestrictedNetwork`: Flags unrestricted network access
  - `TestSecurityRules_UnvalidatedInputs`: Identifies unvalidated external inputs
  - `TestSecurityRules_HardcodedSecrets`: Detects potential hardcoded secrets
  - `TestSecurityRules_ValidWorkflow`: Valid workflow passes all rules
- **Acceptance**: Tests compile and fail expectedly

### 2.3: Write configuration rule tests
- **Dependencies**: 1.4
- **File**: `pkg/workflow/validator/config_test.go`
- **Tests** (table-driven):
  - `TestConfigRules_SchemaCompliance`: Validates against workflow schema
  - `TestConfigRules_EngineSupport`: Checks for supported engines
  - `TestConfigRules_TriggerValidation`: Validates trigger configuration
  - `TestConfigRules_ToolConfiguration`: Checks tool settings
  - `TestConfigRules_ValidWorkflow`: Valid workflow passes all rules
- **Acceptance**: Tests compile and fail expectedly

### 2.4: Write report generator tests
- **Dependencies**: 1.4
- **File**: `pkg/workflow/validator/report_test.go`
- **Tests**:
  - `TestReport_Generate_Console`: Tests console format output
  - `TestReport_Generate_JSON`: Tests JSON format output
  - `TestReport_Summary`: Tests summary aggregation
  - `TestReport_SeverityFiltering`: Tests filtering by severity
- **Acceptance**: Tests compile and fail expectedly

### 2.5: Write CLI command tests
- **Dependencies**: 1.5
- **File**: `cmd/gh-aw/validate_command_test.go`
- **Tests**:
  - `TestValidateCommand_SingleFile`: Validates single file
  - `TestValidateCommand_MultipleFiles`: Validates multiple files
  - `TestValidateCommand_JSONOutput`: Tests JSON output flag
  - `TestValidateCommand_ExitCodes`: Tests exit code behavior
- **Acceptance**: Tests compile and fail expectedly

**Phase 2 Validation**: Run `make test-unit` - all new tests should fail with "not implemented" or similar messages

---

## Phase 3: Core Implementation

### 3.1: Implement validator orchestration
- **Dependencies**: 2.1
- **File**: `pkg/workflow/validator/validator.go`
- **Implementation**:
  - `NewValidator()` constructor
  - `Validate()` method that orchestrates rule execution
  - `ValidateFile()` helper for single file
  - Error handling and aggregation
- **Acceptance**: Tests from 2.1 pass

### 3.2: Implement security rules
- **Dependencies**: 2.2
- **File**: `pkg/workflow/validator/security.go`
- **Implementation**:
  - `BroadPermissionsRule` struct and methods
  - `UnrestrictedNetworkRule` struct and methods
  - `UnvalidatedInputsRule` struct and methods
  - `HardcodedSecretsRule` struct and methods
- **Acceptance**: Tests from 2.2 pass

### 3.3: Implement configuration rules
- **Dependencies**: 2.3
- **File**: `pkg/workflow/validator/config.go`
- **Implementation**:
  - `SchemaComplianceRule` struct and methods
  - `EngineSupportRule` struct and methods
  - `TriggerValidationRule` struct and methods
  - `ToolConfigurationRule` struct and methods
- **Acceptance**: Tests from 2.3 pass

### 3.4: Implement best practice rules
- **Dependencies**: 3.1, 3.2, 3.3
- **File**: `pkg/workflow/validator/bestpractices.go`
- **Implementation**:
  - `DescriptiveTitleRule` struct and methods
  - `DocumentationRule` struct and methods
  - `ErrorHandlingRule` struct and methods
- **Acceptance**: New tests for best practices pass

### 3.5: Implement report generator (console format)
- **Dependencies**: 2.4
- **File**: `pkg/workflow/validator/report.go`
- **Implementation**:
  - `Report` struct methods
  - `GenerateConsoleReport()` using `pkg/console` formatting
  - `Summary()` aggregation method
  - Issue formatting with colors and severity indicators
- **Acceptance**: Console format tests from 2.4 pass

### 3.6: Implement report generator (JSON format)
- **Dependencies**: 3.5
- **File**: `pkg/workflow/validator/report.go`
- **Implementation**:
  - `GenerateJSONReport()` method
  - JSON marshaling with proper structure
  - Error handling for JSON generation
- **Acceptance**: JSON format tests from 2.4 pass

### 3.7: Integrate with workflow parser
- **Dependencies**: 3.1, 3.2, 3.3
- **File**: `pkg/workflow/validator/validator.go`
- **Implementation**:
  - Use existing `pkg/parser` to parse workflow files
  - Extract frontmatter for validation
  - Handle parse errors gracefully
- **Acceptance**: Validator can parse real workflow files

### 3.8: Implement CLI command
- **Dependencies**: 2.5, 3.7
- **File**: `cmd/gh-aw/validate_command.go`
- **Implementation**:
  - Cobra command with flags (--format, --strict)
  - File path argument handling
  - Validator instantiation and execution
  - Output formatting based on flags
  - Exit code handling (0 for valid, 1 for errors)
- **Acceptance**: Tests from 2.5 pass

**Phase 3 Validation**: Run `make test-unit` - all tests should pass, coverage ≥ 80%

---

## Phase 4: Integration

### 4.1: Register CLI command
- **Dependencies**: 3.8
- **File**: `cmd/gh-aw/main.go` or CLI command registry
- **Implementation**:
  - Add validate command to root command
  - Register flags and help text
- **Acceptance**: `./gh-aw validate --help` displays help text

### 4.2: Add integration tests
- **Dependencies**: 4.1
- **File**: `pkg/cli/validate_integration_test.go`
- **Tests**:
  - Test with real workflow files from repository
  - Test batch validation
  - Test error scenarios
  - Test both output formats
- **Acceptance**: Integration tests pass

### 4.3: Test with sample workflows
- **Dependencies**: 4.1
- **Manual Testing**:
  - Validate existing workflows in `.github/workflows/`
  - Test with intentionally broken workflows
  - Verify report accuracy and readability
- **Acceptance**: Manual testing validates all scenarios

### 4.4: Performance testing
- **Dependencies**: 4.1
- **Testing**:
  - Measure validation time for typical workflows
  - Test with large workflow files
  - Verify parallel rule execution
- **Acceptance**: Validation completes < 2 seconds for typical workflows

**Phase 4 Validation**: Run `make test` - all tests pass including integration tests

---

## Phase 5: Polish

### 5.1: Add comprehensive code comments
- **Dependencies**: All previous tasks
- **Files**: All validator package files
- **Implementation**:
  - Package-level documentation
  - Function-level comments for exported functions
  - Inline comments for complex logic
  - Example code in documentation
- **Acceptance**: `go doc pkg/workflow/validator` displays complete documentation

### 5.2: Update CLI help text
- **Dependencies**: 4.1
- **File**: `cmd/gh-aw/validate_command.go`
- **Implementation**:
  - Detailed command description
  - Usage examples
  - Flag descriptions
- **Acceptance**: Help text is clear and includes examples

### 5.3: Create usage documentation
- **Dependencies**: 4.3
- **File**: Documentation site (if applicable) or README.md
- **Implementation**:
  - Add validate command to README
  - Include usage examples
  - Document output formats
  - Explain validation rules
- **Acceptance**: Documentation is complete and accurate

### 5.4: Add error message improvements
- **Dependencies**: 3.8
- **File**: `pkg/workflow/validator/report.go`
- **Implementation**:
  - Ensure all error messages use `console.FormatErrorMessage()`
  - Add helpful recommendations for each issue type
  - Include examples in recommendations
- **Acceptance**: All user-facing messages use console formatting

### 5.5: Run final validation
- **Dependencies**: 5.1, 5.2, 5.3, 5.4
- **Commands**:
  ```bash
  make fmt           # Format code
  make lint          # Run linter
  make test-unit     # Run unit tests
  make test          # Run all tests
  make recompile     # Recompile workflows (if any changed)
  make agent-finish  # Complete validation
  ```
- **Acceptance**: All commands complete successfully with no errors

**Phase 5 Validation**: `make agent-finish` passes completely

---

## Completion Checklist

Feature is complete when all of the following are true:

- [ ] All tasks in all phases completed
- [ ] Unit test coverage ≥ 80%
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Code is formatted (`make fmt`)
- [ ] Console formatting used for all output
- [ ] CLI command registered and working
- [ ] Documentation complete with examples
- [ ] Manual testing validates all user stories
- [ ] Performance targets met (< 2 seconds)
- [ ] No security vulnerabilities introduced
- [ ] `make agent-finish` passes
- [ ] Code review completed
- [ ] All feedback addressed

## Notes

- **TDD Approach**: Write tests before implementation in Phase 2, implement in Phase 3
- **Incremental Validation**: Run validation commands after each phase
- **Test Coverage**: Aim for 80%+ coverage but focus on meaningful tests
- **Console Formatting**: Always use `pkg/console` for user-facing output
- **Error Handling**: Provide helpful error messages and recommendations
- **Performance**: Keep validation fast enough for CI/CD integration
