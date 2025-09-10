---
on:
  workflow_dispatch:

engine: 
  id: claude

safe-outputs:
  create-pull-request:
    title-prefix: "[claude-test] "
    labels: [claude, automation, bot]
---

# Test Claude Create Pull Request

This test workflow specifically tests multi-commit functionality in create-pull-request.

**IMPORTANT: Create multiple separate commits for this test case**

1. **First commit**: Create a file "README-test.md" with content:
   ```markdown
   # Test Project
   
   This is a test project created by Claude to test multi-commit pull requests.
   
   Created at: {{ current timestamp }}
   ```

2. **Second commit**: Create a Python script "test-script.py" with:
   ```python
   #!/usr/bin/env python3
   def hello():
       print("Hello from Claude multi-commit test!")
       
   if __name__ == "__main__":
       hello()
   ```

3. **Third commit**: Create a configuration file "config.json" with:
   ```json
   {
       "test": true,
       "engine": "claude",
       "purpose": "multi-commit-test",
       "timestamp": "{{ current timestamp }}"
   }
   ```

4. **Fourth commit**: Create a log file "test.log" containing the current time. This is just a log file.

Create a pull request with title "Multi-commit test from Claude" and body:
```markdown
# Multi-Commit Test Pull Request

This PR was created by Claude to test the multi-commit functionality in agentic workflows.

## Changes Made

This PR contains exactly 4 commits:
1. Added README-test.md with project description
2. Added test-script.py with Python hello function  
3. Added config.json with test configuration
4. Added test.log with timestamp

Each change was made in a separate commit to test that `git am` properly applies all commits from the patch file, not just the first one.

## Haiku about Multi-Commits

Git commits in line,  
Each change tells its own story—  
Patch applies them all.
```