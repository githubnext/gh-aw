---
on:
  command:
    name: test-codex-push-to-pr-branch

engine: 
  id: codex

safe-outputs:
  push-to-pr-branch:
---

# Test Codex Push to Branch

This test workflow specifically tests multi-commit functionality in push-to-pr-branch.

**IMPORTANT: Create multiple separate commits for this test case**

1. **First commit**: Create a file called "README-codex-test.md" with:
   ```markdown
   # Codex Push-to-Branch Multi-Commit Test
   
   This file was created by the Codex agentic workflow to test the multi-commit push-to-pr-branch functionality.
   
   Created at: {{ current timestamp }}
   
   ## Purpose
   This test verifies that multiple commits are properly applied when using push-to-pr-branch.
   ```

2. **Second commit**: Create a Python script called "codex-script.py" with:
   ```python
   #!/usr/bin/env python3
   """
   Multi-commit test script created by Codex agentic workflow
   """
   
   import datetime
   
   def main():
       print("Hello from Codex agentic workflow!")
       print(f"Current time: {datetime.datetime.now()}")
       print("This script was created to test multi-commit push-to-pr-branch functionality.")
       print("This is commit #2 in the multi-commit test.")
   
   if __name__ == "__main__":
       main()
   ```

Push these changes to the branch for the pull request #${github.event.pull_request.number}
