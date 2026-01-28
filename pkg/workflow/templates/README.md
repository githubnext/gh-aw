# Documentation Templates

This directory contains reusable documentation templates designed to reduce token usage while maintaining quality and consistency when creating workflow documentation.

## Overview

These templates provide pre-structured documentation formats that can be filled with workflow-specific content. They follow consistent patterns and include best practices for navigation and organization.

## Available Templates

### INDEX.template.md
**Purpose**: Navigation hub for all documentation files

**Key Sections**:
- Quick Navigation - Links to all documentation files
- Documentation Structure - Overview of what each file contains
- Getting Started - Recommended reading order
- Additional Resources - Repository links and support

**Use Case**: Create as the entry point for workflow documentation sets.

### README.template.md
**Purpose**: Complete setup and getting started guide

**Key Sections**:
- Overview and description
- Prerequisites and installation
- Quick Start with step-by-step commands
- Configuration settings
- Features overview
- Troubleshooting common issues

**Use Case**: Primary documentation file that users read first.

### QUICKREF.template.md
**Purpose**: Fast reference for day-to-day operations

**Key Sections**:
- Common Commands with examples
- Command Reference table
- Common Patterns with code snippets
- Configuration Quick Reference
- Troubleshooting Cheat Sheet
- Tips & Tricks

**Use Case**: Cheat sheet for users already familiar with the workflow.

### EXAMPLE.template.md
**Purpose**: Real-world usage examples and sample outputs

**Key Sections**:
- Multiple numbered examples with inputs/outputs/explanations
- Common Use Cases with scenarios and steps
- Sample Workflow Runs with configurations
- Output Format Reference
- Interpreting Results
- Advanced Examples and Edge Cases

**Use Case**: Show users what to expect and how to interpret results.

### CONFIG.template.md
**Purpose**: Comprehensive configuration reference and deployment guide

**Key Sections**:
- Configuration File Structure and location
- Configuration Options (core and advanced)
- Environment Variables table
- Permission Requirements
- Deployment Checklist
- Configuration Examples (minimal, production, development)
- Security Considerations
- Performance Tuning
- Migration Guide

**Use Case**: Reference for configuration options and deployment procedures.

## Placeholder Syntax

All templates use consistent placeholder syntax:

```markdown
{{PLACEHOLDER_NAME}}
```

**Rules**:
- Use UPPERCASE letters
- Separate words with underscores
- No spaces inside placeholders
- Clear, descriptive names

**Common Placeholders**:
- `{{WORKFLOW_NAME}}` - Name of the workflow
- `{{WORKFLOW_DESCRIPTION}}` - Brief description
- `{{REPOSITORY_URL}}` - Full repository URL
- `{{COMMAND_1}}`, `{{COMMAND_2}}` - Command examples
- `{{EXAMPLE_1_TITLE}}` - Example titles
- `{{CONFIG_KEY_1}}` - Configuration keys

## Usage Guidelines

### For AI Agents

When creating documentation:

1. **Choose templates** based on what documentation is needed
2. **Copy template content** as a starting point
3. **Replace placeholders** with workflow-specific values
4. **Maintain structure** - keep sections and navigation intact
5. **Customize content** - add workflow-specific details
6. **Link consistency** - ensure all internal links work

### For Developers

When adding new templates:

1. Follow the existing placeholder naming convention
2. Include navigation links to other documentation files
3. Add comprehensive section structure
4. Test placeholders in `documentation_templates_test.go`
5. Update this README with template description

## Template Benefits

- **Token Efficiency**: Reduces token consumption by ~60-70% compared to generating from scratch
- **Consistency**: Ensures uniform documentation structure across workflows
- **Quality**: Built-in best practices for organization and navigation
- **Completeness**: Comprehensive sections prevent missing important information
- **Maintainability**: Easy to update and extend templates centrally

## Testing

Templates are validated by comprehensive tests in `documentation_templates_test.go`:

- File existence verification
- Required sections and placeholders validation
- Consistent placeholder syntax checking
- Internal linking validation
- Markdown structure verification
- Prevention of hardcoded values

Run tests:
```bash
cd /home/runner/work/gh-aw/gh-aw
go test -v ./pkg/workflow -run TestDocumentation
```

## Example Usage

### Filling a Template

Original template snippet:
```markdown
# {{WORKFLOW_NAME}}

{{WORKFLOW_DESCRIPTION}}

## Quick Start

1. **{{QUICK_START_STEP_1}}**
   ```bash
   {{QUICK_START_COMMAND_1}}
   ```
```

Filled with workflow-specific content:
```markdown
# Issue Classifier

Automatically categorize and label issues based on their content

## Quick Start

1. **Install the workflow**
   ```bash
   gh aw compile issue-classifier
   ```
```

## Navigation Structure

Templates include consistent navigation between files:

```
INDEX.md ──┬──> README.md (Overview & Setup)
           │
           ├──> QUICKREF.md (Command Reference)
           │
           ├──> EXAMPLE.md (Sample Outputs)
           │
           └──> CONFIG.md (Configuration)
```

Each file links back to INDEX.md and cross-links to related documentation.

## Future Enhancements

Potential improvements:

- Additional templates for specialized documentation (API reference, troubleshooting guide)
- Template versioning for different workflow types
- Automated placeholder detection and validation
- Integration with workflow compilation process
- Template customization based on workflow characteristics

## See Also

- [AGENTS.md](../../../AGENTS.md) - Documentation section for agent guidelines
- [documentation_templates_test.go](../documentation_templates_test.go) - Test suite
- [specs/testing.md](../../../specs/testing.md) - Testing guidelines
