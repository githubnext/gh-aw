# Artifact File Locations Reference

This document provides a reference for artifact file locations across all agentic workflows.
It is generated automatically and meant to be used by agents when generating file paths in JavaScript and Go code.

## Overview

When artifacts are uploaded, GitHub Actions strips the common parent directory from file paths.
When artifacts are downloaded, files are extracted based on the download mode:

- **Download by name**: Files extracted directly to `path/` (e.g., `path/file.txt`)
- **Download by pattern (no merge)**: Files in `path/artifact-name/` (e.g., `path/artifact-1/file.txt`)
- **Download by pattern (merge)**: Files extracted directly to `path/` (e.g., `path/file.txt`)

## Workflows

### agent-performance-analyzer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `repo-memory-default`
  - **Upload paths**:
    - `/tmp/gh-aw/repo-memory/default`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection push_repo_memory safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `push_repo_memory`

**Downloads:**

- **Artifact**: `repo-memory-default` (by name)
  - **Download path**: `/tmp/gh-aw/repo-memory/default`
  - **Depends on jobs**: [agent detection]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### ai-moderator.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent safe_outputs]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent]

### archie.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### artifacts-summary.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### blog-auditor.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### brave.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### breaking-change-checker.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### campaign-generator.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### changeset.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/mcp-config/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

### ci-coach.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### ci-doctor.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### cli-consistency-checker.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### cloclo.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### commit-changes-analyzer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### copilot-pr-merged-report.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/safe-inputs/logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### copilot-pr-nlp-analysis.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `data-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `python-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `repo-memory-default`
  - **Upload paths**:
    - `/tmp/gh-aw/repo-memory/default`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection push_repo_memory safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `push_repo_memory`

**Downloads:**

- **Artifact**: `repo-memory-default` (by name)
  - **Download path**: `/tmp/gh-aw/repo-memory/default`
  - **Depends on jobs**: [agent detection]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### craft.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

### daily-choice-test.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection test_environment]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `test_environment`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safe-jobs/`
  - **Depends on jobs**: [agent detection]

### daily-copilot-token-report.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `data-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `python-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `repo-memory-default`
  - **Upload paths**:
    - `/tmp/gh-aw/repo-memory/default`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection push_repo_memory safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `push_repo_memory`

**Downloads:**

- **Artifact**: `repo-memory-default` (by name)
  - **Download path**: `/tmp/gh-aw/repo-memory/default`
  - **Depends on jobs**: [agent detection]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### daily-fact.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/mcp-config/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### daily-file-diet.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### daily-issues-report.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `data-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `python-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/mcp-config/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### daily-news.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `data-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `python-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `repo-memory-default`
  - **Upload paths**:
    - `/tmp/gh-aw/repo-memory/default`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection push_repo_memory safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `push_repo_memory`

**Downloads:**

- **Artifact**: `repo-memory-default` (by name)
  - **Download path**: `/tmp/gh-aw/repo-memory/default`
  - **Depends on jobs**: [agent detection]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### daily-repo-chronicle.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `data-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `python-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### deep-report.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/mcp-config/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `repo-memory-default`
  - **Upload paths**:
    - `/tmp/gh-aw/repo-memory/default`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection push_repo_memory safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `push_repo_memory`

**Downloads:**

- **Artifact**: `repo-memory-default` (by name)
  - **Download path**: `/tmp/gh-aw/repo-memory/default`
  - **Depends on jobs**: [agent detection]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### dependabot-go-checker.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### dev-hawk.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### dev.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### dictation-prompt.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

### example-custom-error-patterns.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

### example-permissions-warning.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

### example-workflow-analyzer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### firewall.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

### github-mcp-structural-analysis.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `data-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `python-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### github-mcp-tools-report.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### glossary-maintainer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### go-fan.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### go-pattern-detector.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### grumpy-reviewer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### hourly-ci-cleaner.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

### issue-classifier.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### issue-template-optimizer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### issue-triage-agent.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### layout-spec-maintainer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

### mcp-inspector.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection notion_add_comment post_to_slack_channel safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `notion_add_comment`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safe-jobs/`
  - **Depends on jobs**: [agent detection]

#### Job: `post_to_slack_channel`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safe-jobs/`
  - **Depends on jobs**: [agent detection]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### mergefest.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

### metrics-collector.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `repo-memory-default`
  - **Upload paths**:
    - `/tmp/gh-aw/repo-memory/default`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `push_repo_memory`

**Downloads:**

- **Artifact**: `repo-memory-default` (by name)
  - **Download path**: `/tmp/gh-aw/repo-memory/default`
  - **Depends on jobs**: [agent]

### notion-issue-summary.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection notion_add_comment]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `notion_add_comment`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safe-jobs/`
  - **Depends on jobs**: [agent detection]

### org-health-report.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `data-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `python-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### pdf-summary.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### plan.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### playground-org-project-update-issue.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### playground-snapshots-refresh.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

### poem-bot.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### portfolio-analyst.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `trending-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `trending-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### pr-nitpick-reviewer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### python-data-charts.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `data-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `python-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### q.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### release.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `generate-sbom`

**Uploads:**

- **Artifact**: `sbom-artifacts`
  - **Upload paths**:
    - `sbom.spdx.json
sbom.cdx.json
`

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### repo-tree-map.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### repository-quality-improver.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory-focus-areas`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory-focus-areas`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory-focus-areas` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory-focus-areas`
  - **Depends on jobs**: [agent detection]

### research.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### scout.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### security-compliance.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `repo-memory-default`
  - **Upload paths**:
    - `/tmp/gh-aw/repo-memory/default`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection push_repo_memory safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `push_repo_memory`

**Downloads:**

- **Artifact**: `repo-memory-default` (by name)
  - **Download path**: `/tmp/gh-aw/repo-memory/default`
  - **Depends on jobs**: [agent detection]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### slide-deck-maintainer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### smoke-copilot-playwright.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `playwright-debug-logs-${{ github.run_id }}`
  - **Upload paths**:
    - `/tmp/gh-aw/playwright-debug-logs/`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/safe-inputs/logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### smoke-detector.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### smoke-srt-custom-config.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/agent-stdio.log
`

### smoke-srt.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

### spec-kit-execute.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `repo-memory-default`
  - **Upload paths**:
    - `/tmp/gh-aw/repo-memory/default`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection push_repo_memory safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `push_repo_memory`

**Downloads:**

- **Artifact**: `repo-memory-default` (by name)
  - **Download path**: `/tmp/gh-aw/repo-memory/default`
  - **Depends on jobs**: [agent detection]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### speckit-dispatcher.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### stale-repo-identifier.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `data-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `python-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `trending-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `trending-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### super-linter.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

**Downloads:**

- **Artifact**: `super-linter-log` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation super_linter]

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `super_linter`

**Uploads:**

- **Artifact**: `super-linter-log`
  - **Upload paths**:
    - `super-linter.log`

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

### technical-doc-writer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### tidy.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/aw.patch
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection]

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/`
  - **Depends on jobs**: [activation agent detection]

### typist.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### video-analyzer.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### weekly-issue-summary.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `data-charts`
  - **Upload paths**:
    - `/tmp/gh-aw/python/charts/*.png`

- **Artifact**: `python-source-and-data`
  - **Upload paths**:
    - `/tmp/gh-aw/python/*.py
/tmp/gh-aw/python/data/*
`

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `cache-memory`
  - **Upload paths**:
    - `/tmp/gh-aw/cache-memory`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
/tmp/gh-aw/safeoutputs/assets/
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs update_cache_memory upload_assets]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

#### Job: `update_cache_memory`

**Downloads:**

- **Artifact**: `cache-memory` (by name)
  - **Download path**: `/tmp/gh-aw/cache-memory`
  - **Depends on jobs**: [agent detection]

#### Job: `upload_assets`

**Downloads:**

- **Artifact**: `safe-outputs-assets` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/assets/`
  - **Depends on jobs**: [agent detection]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### workflow-generator.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

### workflow-health-manager.md

#### Job: `agent`

**Uploads:**

- **Artifact**: `safe-output`
  - **Upload paths**:
    - `${{ env.GH_AW_SAFE_OUTPUTS }}`

- **Artifact**: `agent-output`
  - **Upload paths**:
    - `${{ env.GH_AW_AGENT_OUTPUT }}`

- **Artifact**: `agent_outputs`
  - **Upload paths**:
    - `/tmp/gh-aw/sandbox/agent/logs/
/tmp/gh-aw/redacted-urls.log
`

- **Artifact**: `repo-memory-default`
  - **Upload paths**:
    - `/tmp/gh-aw/repo-memory/default`

- **Artifact**: `agent-artifacts`
  - **Upload paths**:
    - `/tmp/gh-aw/aw-prompts/prompt.txt
/tmp/gh-aw/aw_info.json
/tmp/gh-aw/mcp-logs/
/tmp/gh-aw/sandbox/firewall/logs/
/tmp/gh-aw/agent-stdio.log
`

#### Job: `conclusion`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [activation agent detection push_repo_memory safe_outputs]

#### Job: `detection`

**Uploads:**

- **Artifact**: `threat-detection.log`
  - **Upload paths**:
    - `/tmp/gh-aw/threat-detection/detection.log`

**Downloads:**

- **Artifact**: `agent-artifacts` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-artifacts`
  - **Depends on jobs**: [agent]

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/threat-detection/agent-output`
  - **Depends on jobs**: [agent]

#### Job: `push_repo_memory`

**Downloads:**

- **Artifact**: `repo-memory-default` (by name)
  - **Download path**: `/tmp/gh-aw/repo-memory/default`
  - **Depends on jobs**: [agent detection]

#### Job: `safe_outputs`

**Downloads:**

- **Artifact**: `agent-output` (by name)
  - **Download path**: `/tmp/gh-aw/safeoutputs/`
  - **Depends on jobs**: [agent detection]

## Usage Examples

### JavaScript (actions/github-script)

```javascript
// Reading a file from a downloaded artifact
const fs = require('fs');
const path = require('path');

// If artifact 'build-output' was downloaded to '/tmp/artifacts'
// and contains 'dist/app.js' (after common parent stripping)
const filePath = path.join('/tmp/artifacts', 'dist', 'app.js');
const content = fs.readFileSync(filePath, 'utf8');
```

### Go

```go
// Reading a file from a downloaded artifact
import (
    "os"
    "path/filepath"
)

// If artifact 'build-output' was downloaded to '/tmp/artifacts'
// and contains 'dist/app.js' (after common parent stripping)
filePath := filepath.Join("/tmp/artifacts", "dist", "app.js")
content, err := os.ReadFile(filePath)
```

## Notes

- This document is auto-generated from workflow analysis
- Actual file paths may vary based on the workflow execution context
- Always verify file existence before reading in production code
- Common parent directories are automatically stripped during upload
- Use `ComputeDownloadPath()` from the artifact manager for accurate path computation
