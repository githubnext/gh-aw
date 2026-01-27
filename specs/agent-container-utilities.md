# Agent Container Utilities Audit

**Last Updated**: 2026-01-27  
**Related Issue**: #11970

This document provides a comprehensive audit of `/usr/bin` utilities used in agentic workflows, with recommendations for mounting into the agent container.

## Overview

The agent container currently mounts only three utilities from `/usr/bin`:
- `/usr/bin/date` - Date/time operations
- `/usr/bin/gh` - GitHub CLI
- `/usr/bin/yq` - YAML processor

This audit identifies additional utilities commonly used in workflows and provides categorized recommendations for container mounting.

## Methodology

The audit analyzed:
1. **184 workflow files** in `.github/workflows/*.md`
2. **Usage frequency** of common utilities via pattern matching
3. **Lock file analysis** to identify current mounts
4. **Ubuntu runner image** available utilities (from `specs/ubuntulatest.md`)

## Usage Frequency Analysis

The following table shows utility usage frequency in workflow markdown files:

| Utility | Usage Count | Category | Currently Mounted |
|---------|-------------|----------|-------------------|
| `file` | 666* | Optional | ❌ |
| `date` | 344 | Essential | ✅ |
| `jq` | 253 | Essential | ❌ |
| `find` | 172 | Common | ❌ |
| `git` | 160 | Essential | ❌ (via PATH) |
| `grep` | 155 | Essential | ❌ |
| `cat` | 147 | Essential | ❌ |
| `which` | 89 | Common | ❌ |
| `mkdir` | 66 | Common | ❌ |
| `wc` | 60 | Common | ❌ |
| `head` | 53 | Common | ❌ |
| `sort` | 45 | Common | ❌ |
| `diff` | 41 | Common | ❌ |
| `cp` | 37 | Common | ❌ |
| `curl` | 35 | Essential | ❌ |
| `ls` | 23 | Common | ❌ |
| `yq` | 17 | Essential | ✅ |
| `awk` | 15 | Common | ❌ |
| `rm` | 13 | Optional | ❌ |
| `sed` | 10 | Common | ❌ |
| `cut` | 10 | Common | ❌ |
| `chmod` | 8 | Optional | ❌ |
| `zip` | 7 | Optional | ❌ |
| `tail` | 6 | Common | ❌ |
| `mv` | 6 | Optional | ❌ |
| `stat` | 4 | Optional | ❌ |
| `tee` | 4 | Optional | ❌ |
| `ln` | 4 | Optional | ❌ |
| `xargs` | 3 | Optional | ❌ |
| `wget` | 3 | Optional | ❌ |
| `touch` | 3 | Optional | ❌ |
| `unzip` | 2 | Optional | ❌ |
| `base64` | 1 | Optional | ❌ |
| `tr` | 1 | Optional | ❌ |

## Categorized Recommendations

### Essential Utilities (Required for Most Workflows)

These utilities are fundamental to workflow operation and should be mounted.

#### 1. `jq` - JSON Processor
- **Path**: `/usr/bin/jq`
- **Usage**: 253 references, 231 direct command invocations
- **Purpose**: JSON parsing, transformation, and filtering
- **Security**: Low risk - processes data, no network access
- **Recommendation**: **MOUNT** - Critical for API response processing

#### 2. `grep` - Pattern Matcher
- **Path**: `/usr/bin/grep`
- **Usage**: 155 references
- **Purpose**: Text searching and filtering
- **Security**: Low risk - read-only pattern matching
- **Recommendation**: **MOUNT** - Essential for log analysis and text processing

#### 3. `cat` - File Concatenation
- **Path**: `/usr/bin/cat`
- **Usage**: 147 references
- **Purpose**: Reading and displaying file contents
- **Security**: Low risk - read-only file access
- **Recommendation**: **MOUNT** - Basic file reading capability

#### 4. `curl` - HTTP Client
- **Path**: `/usr/bin/curl`
- **Usage**: 35 references
- **Purpose**: HTTP requests, API calls, file downloads
- **Security**: **Medium risk** - network access capability
- **Mitigations**:
  - Already controlled by network firewall rules
  - Workflows define allowed domains in `network.allowed`
- **Recommendation**: **MOUNT** - Required for API integrations

#### 5. `find` - File Search
- **Path**: `/usr/bin/find`
- **Usage**: 172 references
- **Purpose**: Locating files by name, type, or attributes
- **Security**: Low risk - filesystem traversal only
- **Recommendation**: **MOUNT** - Essential for file discovery

#### 6. `git` - Version Control
- **Path**: `/usr/bin/git`
- **Usage**: 160 references
- **Purpose**: Source control operations
- **Security**: **Medium risk** - can fetch from/push to remotes
- **Mitigations**:
  - Network access controlled by firewall
  - Credentials passed via environment
- **Recommendation**: **MOUNT** - Critical for code operations
- **Note**: May already be available via `/opt/hostedtoolcache` mount

### Common Utilities (Frequently Used)

These utilities are commonly used but workflows can function without them.

#### 7. `which` - Command Location
- **Path**: `/usr/bin/which`
- **Usage**: 89 references
- **Purpose**: Finding executable paths
- **Security**: Low risk - PATH inspection only
- **Recommendation**: **MOUNT** - Useful for tool detection

#### 8. `mkdir` - Directory Creation
- **Path**: `/usr/bin/mkdir`
- **Usage**: 66 references
- **Purpose**: Creating directories
- **Security**: Low risk - filesystem write (sandboxed)
- **Recommendation**: **MOUNT** - Common file operations

#### 9. `wc` - Word Count
- **Path**: `/usr/bin/wc`
- **Usage**: 60 references
- **Purpose**: Counting lines, words, bytes
- **Security**: Low risk - read-only counting
- **Recommendation**: **MOUNT** - Useful for metrics and validation

#### 10. `head` / `tail` - File Preview
- **Path**: `/usr/bin/head`, `/usr/bin/tail`
- **Usage**: 53 (head), 6 (tail) references
- **Purpose**: Viewing file beginning/end
- **Security**: Low risk - partial file reading
- **Recommendation**: **MOUNT** - Log and output inspection

#### 11. `sort` - Line Sorting
- **Path**: `/usr/bin/sort`
- **Usage**: 45 references
- **Purpose**: Sorting text lines
- **Security**: Low risk - data transformation
- **Recommendation**: **MOUNT** - Data processing

#### 12. `diff` - File Comparison
- **Path**: `/usr/bin/diff`
- **Usage**: 41 references
- **Purpose**: Comparing files, detecting changes
- **Security**: Low risk - read-only comparison
- **Recommendation**: **MOUNT** - Change detection

#### 13. `cp` - File Copy
- **Path**: `/usr/bin/cp`
- **Usage**: 37 references
- **Purpose**: Copying files and directories
- **Security**: Low risk - filesystem write (sandboxed)
- **Recommendation**: **MOUNT** - File management

#### 14. `ls` - Directory Listing
- **Path**: `/usr/bin/ls`
- **Usage**: 23 references
- **Purpose**: Listing directory contents
- **Security**: Low risk - read-only listing
- **Recommendation**: **MOUNT** - Basic filesystem inspection

#### 15. `sed` / `awk` - Stream Editors
- **Path**: `/usr/bin/sed`, `/usr/bin/awk`
- **Usage**: 10 (sed), 15 (awk) references
- **Purpose**: Text transformation and processing
- **Security**: Low risk - data transformation
- **Recommendation**: **MOUNT** - Advanced text processing

#### 16. `cut` - Column Extraction
- **Path**: `/usr/bin/cut`
- **Usage**: 10 references
- **Purpose**: Extracting text columns
- **Security**: Low risk - text parsing
- **Recommendation**: **MOUNT** - Data extraction

### Optional Utilities (Specialized Use Cases)

These utilities are used in specific workflows and can be mounted on-demand.

#### 17. `file` - File Type Detection
- **Path**: `/usr/bin/file`
- **Usage**: 666 references* (most are variable names like `file_path`, actual command usage is minimal)
- **Purpose**: Detecting file types by content  
- **Security**: Low risk - metadata inspection
- **Recommendation**: **OPTIONAL** - Specialized file analysis (low actual command usage)

#### 18. `rm` - File Removal
- **Path**: `/usr/bin/rm`
- **Usage**: 13 references
- **Purpose**: Deleting files and directories
- **Security**: **Medium risk** - destructive operation
- **Mitigations**: Sandboxed to workspace directory
- **Recommendation**: **MOUNT** - Cleanup operations

#### 19. `chmod` - Permission Modifier
- **Path**: `/usr/bin/chmod`
- **Usage**: 8 references
- **Purpose**: Changing file permissions
- **Security**: Low risk - permission changes (sandboxed)
- **Recommendation**: **OPTIONAL** - Script execution prep

#### 20. `zip` / `unzip` - Compression
- **Path**: `/usr/bin/zip`, `/usr/bin/unzip`
- **Usage**: 7 (zip), 2 (unzip) references
- **Purpose**: Creating and extracting archives
- **Security**: Low risk - file compression
- **Recommendation**: **OPTIONAL** - Artifact handling

#### 21. `mv` - File Move
- **Path**: `/usr/bin/mv`
- **Usage**: 6 references
- **Purpose**: Moving/renaming files
- **Security**: Low risk - filesystem reorganization (sandboxed)
- **Recommendation**: **OPTIONAL** - File management

#### 22. `wget` - File Download
- **Path**: `/usr/bin/wget`
- **Usage**: 3 references
- **Purpose**: Downloading files from URLs
- **Security**: **Medium risk** - network access
- **Mitigations**: Network firewall rules apply
- **Recommendation**: **OPTIONAL** - curl usually preferred

#### 23. `touch` - Timestamp Modifier
- **Path**: `/usr/bin/touch`
- **Usage**: 3 references
- **Purpose**: Creating empty files, updating timestamps
- **Security**: Low risk - minimal filesystem impact
- **Recommendation**: **OPTIONAL** - File creation

#### 24. `xargs` - Argument Builder
- **Path**: `/usr/bin/xargs`
- **Usage**: 3 references
- **Purpose**: Building command lines from input
- **Security**: **Medium risk** - command execution
- **Recommendation**: **OPTIONAL** - Advanced scripting

#### 25. `base64` - Encoding
- **Path**: `/usr/bin/base64`
- **Usage**: 1 reference
- **Purpose**: Base64 encoding/decoding
- **Security**: Low risk - data encoding
- **Recommendation**: **OPTIONAL** - Data encoding

#### 26. `tar` - Archive Tool
- **Path**: `/usr/bin/tar`
- **Usage**: Referenced in documentation
- **Purpose**: Creating and extracting tar archives
- **Security**: Low risk - file archiving
- **Recommendation**: **OPTIONAL** - Large artifact handling

#### 27. `tee` - Output Splitter
- **Path**: `/usr/bin/tee`
- **Usage**: 4 references
- **Purpose**: Writing to file and stdout simultaneously
- **Security**: Low risk - output duplication
- **Recommendation**: **OPTIONAL** - Logging

#### 28. `stat` - File Status
- **Path**: `/usr/bin/stat`
- **Usage**: 4 references
- **Purpose**: Displaying file/filesystem status
- **Security**: Low risk - metadata reading
- **Recommendation**: **OPTIONAL** - File inspection

## Security Considerations

### Risk Categories

| Risk Level | Description | Examples |
|------------|-------------|----------|
| **Low** | Read-only or sandboxed operations | `cat`, `grep`, `wc`, `jq` |
| **Medium** | Network access or command execution | `curl`, `wget`, `git`, `xargs` |
| **High** | System modification, privilege escalation | `sudo`, `chown`, system utilities |

### Mitigation Strategies

1. **Read-Only Mounts**: All `/usr/bin` mounts use `:ro` (read-only)
2. **Network Firewall**: Workflows define allowed domains, blocking unauthorized network access
3. **Workspace Sandboxing**: File operations are restricted to the workspace directory
4. **Environment Control**: Sensitive data passed via environment variables, not command arguments

### Utilities NOT Recommended for Mounting

| Utility | Reason |
|---------|--------|
| `sudo` | Privilege escalation risk |
| `chown` | Ownership manipulation |
| `mount` | Filesystem manipulation |
| `passwd` | User credential modification |
| `ssh` | Direct remote access (use gh CLI instead) |
| `nc`/`netcat` | Raw network access |
| `dd` | Low-level disk operations |

## Implementation Recommendations

### Immediate (Priority 1)

Add these utilities to `copilot_engine_execution.go`:

```go
// Essential utilities for most workflows
awfArgs = append(awfArgs, "--mount", "/usr/bin/jq:/usr/bin/jq:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/grep:/usr/bin/grep:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/cat:/usr/bin/cat:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/curl:/usr/bin/curl:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/find:/usr/bin/find:ro")
```

### Short Term (Priority 2)

Add commonly used utilities:

```go
// Common utilities for file operations
awfArgs = append(awfArgs, "--mount", "/usr/bin/which:/usr/bin/which:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/mkdir:/usr/bin/mkdir:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/wc:/usr/bin/wc:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/head:/usr/bin/head:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/tail:/usr/bin/tail:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/sort:/usr/bin/sort:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/diff:/usr/bin/diff:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/cp:/usr/bin/cp:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/ls:/usr/bin/ls:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/sed:/usr/bin/sed:ro")
awfArgs = append(awfArgs, "--mount", "/usr/bin/cut:/usr/bin/cut:ro")
```

### Long Term (Priority 3)

Consider a configuration-based approach allowing workflows to specify required utilities:

```yaml
---
engine: copilot
sandbox:
  utilities:
    - jq
    - curl
    - tar
    - zip
---
```

## Summary

This audit identifies **28 utilities** commonly used in agentic workflows:
- **6 Essential**: `jq`, `grep`, `cat`, `curl`, `find`, `git`
- **10 Common**: `which`, `mkdir`, `wc`, `head`, `tail`, `sort`, `diff`, `cp`, `ls`, `sed`, `awk`, `cut`
- **12 Optional**: `file`, `rm`, `chmod`, `zip`, `unzip`, `mv`, `wget`, `touch`, `xargs`, `base64`, `tar`, `tee`, `stat`

The current configuration (only `date`, `gh`, `yq`) is insufficient for most workflows. Adding the essential and common utilities would significantly improve workflow compatibility while maintaining security through read-only mounts and existing network controls.

## References

- [Ubuntu Runner Image Analysis](./ubuntulatest.md) - Available utilities on Ubuntu runner
- [Copilot Engine Execution](../pkg/workflow/copilot_engine_execution.go) - Current mount implementation
- Related Issue: #11970

---

*Note: Some usage counts include variable names and text references, not just command invocations. The categorization is based on actual command usage analysis.*
