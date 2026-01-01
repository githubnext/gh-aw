# Architecture

This document describes the architecture, design decisions, and operational processes for GitHub Agentic Workflows.

## Overview

GitHub Agentic Workflows (gh-aw) is a system for writing agentic workflows in natural language markdown files that are compiled and executed as GitHub Actions. The system provides a bridge between human-friendly workflow descriptions and machine-executable GitHub Actions configurations.

## Core Components

### 1. Workflow Compiler

The workflow compiler (`pkg/workflow/compiler.go`) transforms markdown workflow files into GitHub Actions YAML files.

**Key responsibilities:**
- Parse frontmatter YAML configuration
- Process workflow steps and instructions
- Generate GitHub Actions workflow files (`.lock.yml`)
- Validate schema compliance
- Handle safe outputs and permissions

### 2. CLI Interface

The command-line interface (`cmd/gh-aw/`) provides user interaction:

- `compile` - Compile markdown workflows to GitHub Actions
- `mcp` - Manage MCP (Model Context Protocol) servers
- `logs` - Download and analyze workflow logs
- `audit` - Audit workflow runs

### 3. Schema Validation

Schema validation ensures workflows conform to GitHub Actions specifications and security requirements.

**Location:** `pkg/parser/schemas/`

**Key validations:**
- GitHub Actions YAML schema compliance
- Security constraints (permissions, network access)
- Field usage and deprecation tracking

## Schema Field Adoption Monitoring

### Purpose

The adoption monitoring system tracks usage of low-adoption schema fields across all agentic workflows in the repository. This data informs decisions about field deprecation, API simplification, and documentation improvements.

### Monitored Fields

Five schema fields are monitored due to low adoption (≤1 workflow out of 90+ production workflows):

| Field | Purpose | Initial Adoption |
|-------|---------|------------------|
| `run-name` | Custom workflow run name | 0% (0 workflows) |
| `runtimes` | Multiple runtime environments | 0.6% (1 workflow) |
| `runs-on` | Custom runner specification | 0.6% (1 workflow) |
| `post-steps` | Cleanup steps after workflow | 0.6% (1 workflow) |
| `bots` | Bot configuration for workflow | 0.6% (1 workflow) |

**Decision criterion:** If usage remains <3% by Q2 2025, initiate deprecation discussion.

### Components

#### 1. Monitoring Script

**Location:** `scripts/monitor-adoption.sh`

**Functionality:**
- Scans all workflow markdown files in `.github/workflows/`
- Counts usage of each monitored field
- Calculates adoption percentage
- Generates trend analysis
- Stores results in JSON format
- Provides visual terminal output with color coding

**Usage:**
```bash
# Run with terminal output
./scripts/monitor-adoption.sh

# Output JSON only
./scripts/monitor-adoption.sh --json

# Show help
./scripts/monitor-adoption.sh --help
```

**Exit codes:**
- `0` - All fields above threshold
- `1` - One or more fields below threshold

#### 2. Metrics Storage

**Location:** `.github/adoption-metrics.json`

**Format:**
```json
[
  {
    "timestamp": "2026-01-01T04:20:10Z",
    "total_workflows": 176,
    "threshold_percent": 3,
    "fields": {
      "run-name": {
        "count": 0,
        "percentage": 0.0
      },
      "runtimes": {
        "count": 1,
        "percentage": 0.6
      }
    }
  }
]
```

The metrics file maintains historical data as a JSON array, with new measurements appended to track trends over time.

#### 3. Automated Workflow

**Location:** `.github/workflows/monitor-adoption.yml`

**Trigger:** Runs quarterly on January 1, April 1, July 1, and October 1 at 00:00 UTC

**Actions:**
1. Execute monitoring script
2. Commit updated metrics to repository
3. Create or update GitHub issue if fields remain below threshold

**Issue Management:**
- Creates new issue with label `adoption-monitoring` if none exists
- Updates existing open issue with new data if one exists
- Includes trend analysis and recommended next steps

### Monitoring Process

#### Quarterly Review Cycle

**Q1 2025 (Baseline):**
- Initial metrics captured: January 1, 2025
- Baseline established for all five fields
- No action required; observation phase

**Q2 2025 (First Decision Point):**
- April 1 measurement compared to baseline
- If fields remain <3%, initiate deprecation discussion
- Document reasons for low adoption
- Decide on remediation strategy

**Q3 2025 (Mid-year Review):**
- July 1 measurement
- Evaluate impact of Q2 decisions
- Adjust strategy if needed

**Q4 2025 (Annual Review):**
- October 1 measurement
- Year-end assessment
- Plan for next year

#### Decision Framework

When a field remains below 3% threshold:

1. **Analyze Root Cause:**
   - Is the feature poorly documented?
   - Is the API too complex?
   - Is the feature solving a rare use case?
   - Are there better alternatives?

2. **Evaluation Options:**
   - **Improve:** Better docs, examples, tutorials
   - **Simplify:** Redesign API for easier adoption
   - **Deprecate:** Remove if not serving users
   - **Wait:** Monitor for one more quarter

3. **Deprecation Process:**
   - Announce deprecation with 6-month notice
   - Provide migration guide
   - Add warnings to documentation
   - Remove in next major version

### Manual Monitoring

To manually check adoption:

```bash
# Run monitoring script
cd /path/to/gh-aw
./scripts/monitor-adoption.sh

# View historical metrics
cat .github/adoption-metrics.json | jq '.'

# Compare last two measurements
cat .github/adoption-metrics.json | jq '[.[-2], .[-1]]'
```

### Adding New Fields to Monitor

To monitor additional fields:

1. Edit `scripts/monitor-adoption.sh`
2. Add field name to `FIELDS` array:
   ```bash
   FIELDS=("run-name" "runtimes" "runs-on" "post-steps" "bots" "new-field")
   ```
3. Run script to establish baseline
4. Update this documentation

### Metrics Analysis

The monitoring system provides several views:

**Current Status:**
- Total workflow count
- Field usage count
- Adoption percentage
- Threshold comparison

**Trend Analysis:**
- Change since previous measurement
- Growth or decline indicators
- Historical pattern visualization

**Alerting:**
- Automatic issue creation
- Threshold violation tracking
- Recommended actions

## Development Workflow

### Building

```bash
make build        # Build the binary
make test         # Run all tests
make test-unit    # Run unit tests only
make lint         # Run linter
```

### Testing

```bash
# Unit tests
make test-unit

# Integration tests
make test

# Specific test
go test -v ./pkg/workflow -run TestCompiler
```

### Release

Releases are managed through changesets:

```bash
# Create changeset
npx changeset

# Version and publish
npx changeset version
npx changeset publish
```

## Security Considerations

### Permissions

Workflows use minimal permissions by default. Escalated permissions must be explicitly declared.

### Network Access

Network access is restricted by default. Allowed domains must be explicitly configured.

### Secrets

Secrets are never logged or exposed in outputs. Safe output system sanitizes all data.

## Extensibility

### Custom Engines

New AI engines can be added by implementing the engine interface in `pkg/workflow/`.

### Custom Tools

Tools can be configured in workflow frontmatter to extend agent capabilities.

### Custom Actions

Custom GitHub Actions are maintained in `actions/` directory.

## Monitoring and Observability

### Logs

Workflow logs can be downloaded and analyzed:

```bash
gh aw logs --start-date -7d
```

### Auditing

Audit specific workflow runs:

```bash
gh aw audit RUN_ID
```

### Metrics

See [Schema Field Adoption Monitoring](#schema-field-adoption-monitoring) above.

## Future Considerations

### Planned Improvements

- Enhanced schema validation
- Additional AI engine support
- Improved error messages
- Performance optimizations

### Deprecation Pipeline

Fields that consistently show <3% adoption will follow the deprecation process outlined in the monitoring section.

---

**Last Updated:** 2026-01-01  
**Version:** 1.0.0
