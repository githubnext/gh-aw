---
on:
  command:
    name: test-claude-push-to-branch

engine: 
  id: claude

safe-outputs:
  push-to-branch:
    branch: claude-test-branch
    target: "*"
---

Create a new file called "claude-test-file.md" with the following content:

```markdown
# Claude Test File

This file was created by the Claude agentic workflow to test the push-to-branch functionality.

Created at: {{ current timestamp }}

## Test Content

This is a test file created by Claude to demonstrate:
- File creation
- Branch pushing
- Automated commit generation

The workflow should push this file to the specified branch.
```

Also create a simple Python script called "claude-script.py" with:

```python
#!/usr/bin/env python3
"""
Test script created by Claude agentic workflow
"""

import datetime

def main():
    print("Hello from Claude agentic workflow!")
    print(f"Current time: {datetime.datetime.now()}")
    print("This script was created to test push-to-branch functionality.")

if __name__ == "__main__":
    main()
```

Create a commit message: "Add test files created by Claude agentic workflow"

Push these changes to the branch for the pull request
