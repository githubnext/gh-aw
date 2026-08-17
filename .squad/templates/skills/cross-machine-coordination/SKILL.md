---
name: "cross-machine-coordination"
description: "Enables squad agents on different machines to share work via git-based task queuing"
domain: "orchestration"
confidence: "medium"
source: "manual"
---

# Skill: Cross-Machine Coordination Pattern

**Skill ID:** `cross-machine-coordination`  
**Owner:** Ralph (Work Monitor)  
**Squad Integration:** All agents  
**Status:** Specification (ready for implementation)  

---

## Overview

Enables squad agents running on different machines (laptop, DevBox, Azure VM) to securely share work, coordinate execution, and pass results without manual intervention.

**Pattern:** Git-based task queuing + GitHub Issues supplement

---

## Usage

### For Task Sources (Orchestrating Machine)

**To assign work to DevBox:**

```bash
# Create task file
cat > .squad/cross-machine/tasks/2026-03-14T1530Z-laptop-gpu-voice-clone.yaml << 'EOF'
id: gpu-voice-clone-001
source_machine: laptop-machine
target_machine: devbox
priority: high
created_at: 2026-03-14T15:30:00Z
task_type: gpu_workload
payload:
  command: "python scripts/voice-clone.py --input voice.wav --output cloned.wav"
  expected_duration_min: 15
  resources:
    gpu: true
    memory_gb: 8
status: pending
EOF

# Commit & push
git add .squad/cross-machine/tasks/
git commit -m "Cross-machine task: GPU voice cloning [squad:machine-devbox]"
git push origin main
```

Ralph on DevBox will:
1. Pull the task on next cycle (5-10 min)
2. Validate schema & command whitelist
3. Execute the GPU workload
4. Write result to `.squad/cross-machine/results/gpu-voice-clone-001.yaml`
5. Commit & push the result

---

### For Task Executors (DevBox, Azure VMs)

Ralph automatically watches `.squad/cross-machine/tasks/` for work targeted at this machine.

**On each cycle (5-10 min):**

```python
# Pseudo-code (Ralph implementation)
1. git pull origin main
2. Load all .yaml files in .squad/cross-machine/tasks/
3. Filter for status=pending AND target_machine=HOSTNAME
4. For each task:
   a. Validate schema (must have: id, source_machine, target_machine, payload)
   b. Validate command against whitelist
   c. Execute task (with timeout)
   d. Write result to .squad/cross-machine/results/{id}.yaml
   e. Commit & push result
```

---

### For Urgent/Ad-Hoc Tasks

**Use GitHub Issues with `squad:machine-{name}` label:**

```bash
# Create issue
gh issue create \
  --title "GPU: Clone voice profile from sample.wav" \
  --body "Execute voice cloning on DevBox. Input: /path/to/voice-input.wav" \
  --label "squad:machine-devbox" \
  --label "urgent"
```

Ralph on DevBox will:
1. Detect issue with `squad:machine-devbox` label
2. Parse task from issue body
3. Execute task
4. Comment with result
5. Close issue

---

## File Formats

### Task File (YAML)

**Location:** `.squad/cross-machine/tasks/{timestamp}-{machine}-{task-id}.yaml`

**Required Fields:**
```yaml
id: {task-id}                      # Unique identifier (alphanumeric + dash)
source_machine: {hostname}         # Where task was created
target_machine: {hostname}         # Where task will execute
priority: high|normal|low          # Execution priority
created_at: 2026-03-14T15:30:00Z   # ISO 8601 timestamp
task_type: gpu_workload|script|... # Category
payload:
  command: "..."                   # Shell command to execute
  expected_duration_min: 15        # Timeout (minutes)
  resources:
    gpu: true|false
    memory_gb: 8
    cpu_cores: 4
status: pending|executing|completed|failed
```

**Optional Fields:**
```yaml
description: "Human-readable task description"
timeout_override_min: 120          # Override default timeout
retry_count: 3                     # Retry failed tasks
```

### Result File (YAML)

**Location:** `.squad/cross-machine/results/{task-id}.yaml`

```yaml
id: {task-id}                          # Links back to task
target_machine: devbox                 # Executed on
completed_at: 2026-03-14T15:45:00Z    # When it finished
status: completed|failed|timeout       # Outcome
exit_code: 0                           # Shell exit code
stdout: "..."                          # Captured output
stderr: "..."                          # Captured errors
duration_seconds: 900                  # How long it took
artifacts:
  - path: "/path/to/artifacts/..."   # Location of results
    type: audio|text|model|...
    size_mb: 2.5
```

---

## Security Model

### Validation Pipeline

All tasks go through:

1. **Schema Validation**
   - YAML structure matches spec
   - Required fields present
   - No unexpected fields (reject)

2. **Command Whitelist**
   - Only approved commands allowed
   - Path validation (no `../../` escapes)
   - Environment variable sanitization
   - No inline shell operators (`&&`, `|`, `>`)

3. **Resource Limits**
   - Timeout enforced (default: 60 min)
   - Memory cap: 16GB (adjustable)
   - CPU threads: 4 (adjustable)
   - Disk write: 100GB (adjustable)

4. **Execution Isolation**
   - Runs as unprivileged user
   - Temp directory cleaned after execution
   - Network access: read-only (no outbound writes)

5. **Audit Trail**
   - All executions logged to git
   - Commit signed with Ralph's key
   - Result stored immutably

### Threat Mitigations

| Threat | Mitigation |
|--------|-----------|
| **Malicious task injection** | Branch protection + PR review before merge |
| **Credential leakage** | Pre-commit secret scan + environment scrubbing |
| **Resource exhaustion** | Timeout + memory limits |
| **Code injection** | Command whitelist + no shell evaluation |
| **Result tampering** | Git commit history is immutable |

---

## Configuration

Ralph reads config from `.squad/config.json`:

```json
{
  "cross_machine": {
    "enabled": true,
    "poll_interval_seconds": 300,
    "this_machine": "devbox",
    "max_concurrent_tasks": 2,
    "task_timeout_minutes": 60,
    "command_whitelist": [
      "python scripts/voice-clone.py",
      "python scripts/data-process.py",
      "bash scripts/cleanup.sh"
    ],
    "result_ttl_days": 30
  }
}
```

---

## Examples

### Example 1: GPU Voice Cloning (Laptop → DevBox)

**1. Laptop creates task:**

```yaml
# .squad/cross-machine/tasks/2026-03-14T1530Z-laptop-gpu-001.yaml
id: gpu-voice-clone-001
source_machine: laptop-machine
target_machine: devbox
priority: high
created_at: 2026-03-14T15:30:00Z
task_type: gpu_workload
payload:
  command: "python scripts/voice-clone.py --input voice.wav --output cloned.wav"
  expected_duration_min: 15
  resources:
    gpu: true
    memory_gb: 8
status: pending
```

**2. Laptop commits & pushes:**

```bash
git add .squad/cross-machine/tasks/
git commit -m "Task: GPU voice cloning [squad:machine-devbox]"
git push origin main
```

**3. DevBox Ralph (5 min later):**

```
[Ralph Watch Cycle]
- Pulled origin/main
- Detected: gpu-voice-clone-001 (status: pending, target: devbox)
- Validation: ✅ Schema OK, command whitelisted
- Executing: python scripts/voice-clone.py ...
- [15 minutes of processing]
- Completed: exit code 0
- Writing result...
- Committing & pushing...
```

**4. Laptop Ralph (next cycle) sees result:**

```yaml
# .squad/cross-machine/results/gpu-voice-clone-001.yaml
id: gpu-voice-clone-001
target_machine: devbox
completed_at: 2026-03-14T15:45:00Z
status: completed
exit_code: 0
stdout: "Voice cloning completed. Output written to /tmp/cloned.wav"
stderr: ""
duration_seconds: 900
artifacts:
  - path: "/path/to/artifacts/voice-clone-001/output.wav"
    type: audio
    size_mb: 2.5
```

---

### Example 2: Urgent Debug Request (Human → DevBox via Issue)

**Create issue:**

```bash
gh issue create \
  --title "DevBox: Debug voice model failure" \
  --body "Error: Model failed to load on last run. Please check /tmp/model.log and report findings." \
  --label "squad:machine-devbox" \
  --label "urgent"
```

**DevBox Ralph detects → executes → comments:**

```
✅ Executed on devbox at 2026-03-14 15:47:00
Command: python scripts/debug-model.py

Result:
------
Model file: /tmp/model-v2.bin (OK)
Checksum: a1b2c3d4e5f6 (matches expected)
Memory available: 12 GB (sufficient)

ERROR FOUND: Config file permission issue
  - File: ~/.config/voice/model.yaml
  - Permissions: -rw------- (owner-only)
  - Expected: -rw-r--r-- (world-readable for service)

FIX: Run: chmod 644 ~/.config/voice/model.yaml
```

---

## Error Handling

### Task Execution Failures

If a task fails (exit code != 0):

1. Result written with `status: failed` + exit code
2. stderr captured in result
3. Committed to git for audit
4. Source machine can retry by re-pushing task with `status: pending`

### Stalled Tasks

If a task doesn't complete within timeout:

1. Process killed
2. Result written with `status: timeout`
3. stderr: "Execution exceeded X minutes"
4. Source can investigate or retry

### Network Failures

If git push/pull fails:

- Ralph retries on next cycle
- Tasks queue locally until connectivity restored
- No tasks lost (stored in local repo)

---

## Monitoring & Debugging

### Check Task Queue

```bash
ls -la .squad/cross-machine/tasks/
cat .squad/cross-machine/tasks/*.yaml | grep -E "^(id|status|target_machine):"
```

### Check Results

```bash
ls -la .squad/cross-machine/results/
cat .squad/cross-machine/results/{task-id}.yaml
```

### View Execution History

```bash
git log --oneline .squad/cross-machine/ | head -20
```

### Monitor Ralph Cycles

```bash
tail -f .squad/log/ralph-watch.log | grep "cross-machine"
```

---

## Integration with Ralph Watch

Ralph automatically includes this pattern in its watch loop:

```
Ralph Watch Cycle (every 5-10 min):
1. Fetch GitHub issues with squad:machine-* labels
2. Poll .squad/cross-machine/tasks/
3. For each matching task:
   - Validate
   - Execute
   - Write result
   - Commit & push
4. Update status in issue (if applicable)
5. Sleep until next cycle
```

No manual Ralph configuration needed — just create task files or issues with the right labels.

---

## Migration from Manual Handoff

**Before (today):**
- Laptop → user manually copies file to Teams chat
- user pastes into target terminal
- user copies output back
- user pastes result manually

**After (with this pattern):**
- Laptop Ralph writes task file → git push
- DevBox Ralph auto-executes → git push result
- Laptop Ralph auto-reads result
- 0 human intervention needed

---

## Future Enhancements

Potential expansions (Phase 2+):

1. **Task Priorities:** Execution order based on priority field
2. **Serial Pipelines:** Machine A → B → C task chains
3. **GPU Availability Polling:** Query DevBox before submitting work
4. **Cost Tracking:** Log resource usage per task
5. **Notification Webhooks:** Alert on task completion
6. **Web Dashboard:** Real-time task status visualization

---

## Questions?

Refer to research report: `research/active/cross-machine-agents/README.md`

Contact: Seven (Research & Docs) or Ralph (Work Monitor)
