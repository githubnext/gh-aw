---
name: Session Storage Example - Continue Mode
description: |
  Example workflow demonstrating session storage with automatic continuation
  of the most recent session.
on:
  workflow_dispatch:

engine:
  id: claude
  # Session configuration enables continuing the most recent session
  # This automatically resumes where the previous run left off
  session:
    continue: true

tools:
  github:
    mode: remote
    toolsets: [default]
---

# Session Storage Example

This workflow demonstrates the session storage feature which allows agents to:
- **Continue locally**: Pull session state from previous runs and continue work locally
- **Fork sessions**: Branch from a previous session to explore different approaches
- **Restart sessions**: Resume from a specific point to recover from errors

## How Session Storage Works

This workflow uses `session.continue: true` which automatically resumes the most 
recent session without prompting. This is useful for:
- Quick iteration on ongoing tasks
- Automated workflows that build on previous runs  
- Continuing work after interruption

## Other Session Modes

### Resume Mode (Interactive Picker)
To show a session picker for selecting from previous sessions:
```yaml
engine:
  id: claude  # or copilot
  session:
    resume: true
```

### Resume Specific Session (Claude only)
To resume a specific session by ID:
```yaml
engine:
  id: claude
  session:
    resume: true
    id: "session-abc123"
```

### Simple Enable (Storage Only)
To enable session storage without automatic resume:
```yaml
engine:
  id: claude
  session: true
```

## Example Task

Please analyze the repository structure and identify:
1. The main entry points for the codebase
2. Key architectural patterns used
3. Any potential improvements for code organization

Store your findings in a markdown file that can be referenced in future sessions.
