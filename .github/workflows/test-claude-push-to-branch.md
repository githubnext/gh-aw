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

# Test Claude Push to Branch

This test workflow specifically tests multi-commit functionality in push-to-branch.

**IMPORTANT: Create multiple separate commits for this test case**

1. **First commit**: Create a file called "README-claude-test.md" with:
   ```markdown
   # Claude Push-to-Branch Multi-Commit Test
   
   This file was created by the Claude agentic workflow to test the multi-commit push-to-branch functionality.
   
   Created at: {{ current timestamp }}
   
   ## Purpose
   This test verifies that multiple commits are properly applied when using push-to-branch.
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
       print("This script was created to test multi-commit push-to-branch functionality.")
       print("This is commit #2 in the multi-commit test.")
   
   if __name__ == "__main__":
       main()
   ```

3. **Third commit**: Create a configuration file "claude-config.json" with:
   ```json
   {
       "test_type": "multi-commit-push-to-branch",
       "engine": "claude",
       "branch": "claude-test-branch",
       "commit_number": 3,
       "timestamp": "{{ current timestamp }}"
   }
   ```

4. **Fourth commit**: Create a simple test file "claude-test-results.txt" with:
   ```
   Claude Multi-Commit Push-to-Branch Test Results
   ===============================================
   
   Test Type: Multi-commit functionality
   Engine: Claude
   Branch: claude-test-branch
   Commits Expected: 4
   
   This file represents the final commit in the multi-commit test sequence.
   If you can see all 4 files (README, Python script, JSON config, and this file),
   then the multi-commit functionality is working correctly.
   ```

The workflow should push all these files to the claude-test-branch in separate commits, testing that `git am` properly applies multiple commits from the patch file.

Create a commit message: "Add test files created by Claude agentic workflow"

Push these changes to the branch for the pull request #${github.event.pull_request.number}
