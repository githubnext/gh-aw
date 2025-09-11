---
on:
  command:
    name: test-claude-push-to-pr-branch

engine: 
  id: claude

safe-outputs:
  push-to-pr-branch:
---

# Test Claude Push to Branch

This test workflow specifically tests multi-commit functionality in push-to-pr-branch.

**IMPORTANT: Create multiple separate commits for this test case**

1. **First commit**: Create a file called "README-claude-test.md" with:
   ```markdown
   # Claude Push-to-Branch Multi-Commit Test
   
   This file was created by the Claude agentic workflow to test the multi-commit push-to-pr-branch functionality.
   
   Created at: {{ current timestamp }}
   
   ## Purpose
   This test verifies that multiple commits are properly applied when using push-to-pr-branch.
   ```

2. **Second commit**: Create a Python script called "claude-script.py" with:
   ```python
   #!/usr/bin/env python3
   """
   Multi-commit test script created by Claude agentic workflow
   """
   
   import datetime
   
   def main():
       print("Hello from Claude agentic workflow!")
       print(f"Current time: {datetime.datetime.now()}")
       print("This script was created to test multi-commit push-to-pr-branch functionality.")
       print("This is commit #2 in the multi-commit test.")
   
   if __name__ == "__main__":
       main()
   ```

Push these changes to the branch for the pull request #${github.event.pull_request.number}

