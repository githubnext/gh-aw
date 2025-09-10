---
on:
  command:
    name: test-codex-push-to-branch

engine: 
  id: codex

safe-outputs:
  push-to-branch:
    branch: codex-test-branch
    target: "*"
---

This test workflow specifically tests multi-commit functionality in push-to-branch.

**IMPORTANT: Create multiple separate commits for this test case**

1. **First commit**: Create a file called "README-codex-test.md" with:
   ```markdown
   # Codex Push-to-Branch Multi-Commit Test
   
   This file was created by the Codex agentic workflow to test the multi-commit push-to-branch functionality.
   
   Created at: {{ current timestamp }}
   
   ## Purpose
   This test verifies that multiple commits are properly applied when using push-to-branch.
   ```

2. **Second commit**: Create a JavaScript script called "codex-script.js" with:
   ```javascript
   #!/usr/bin/env node
   /**
    * Multi-commit test script created by Codex agentic workflow
    */
   
   function main() {
       console.log("Hello from Codex agentic workflow!");
       console.log(`Current time: ${new Date().toISOString()}`);
       console.log("This script was created to test multi-commit push-to-branch functionality.");
       console.log("This is commit #2 in the multi-commit test.");
   }
   
   if (require.main === module) {
       main();
   }
   ```

3. **Third commit**: Create a configuration file "codex-config.json" with:
   ```json
   {
       "test_type": "multi-commit-push-to-branch",
       "engine": "codex",
       "branch": "codex-test-branch", 
       "commit_number": 3,
       "timestamp": "{{ current timestamp }}"
   }
   ```

4. **Fourth commit**: Create a simple test file "codex-test-results.txt" with:
   ```
   Codex Multi-Commit Push-to-Branch Test Results
   ==============================================
   
   Test Type: Multi-commit functionality
   Engine: Codex
   Branch: codex-test-branch
   Commits Expected: 4
   
   This file represents the final commit in the multi-commit test sequence.
   If you can see all 4 files (README, JavaScript script, JSON config, and this file),
   then the multi-commit functionality is working correctly.
   ```

The workflow should push all these files to the codex-test-branch in separate commits, testing that `git am` properly applies multiple commits from the patch file.

Create a new file called "codex-test-file.md" with the following content:

```markdown
# Test Codex Push To Branch

This file was created by the Codex agentic workflow to test the push-to-branch functionality.

Created at: {{ current timestamp }}

## Test Content

This is a test file created by Codex to demonstrate:
- File creation
- Branch pushing
- Automated commit generation

The workflow should push this file to the specified branch.
```

Also create a simple JavaScript script called "codex-script.js" with:

```javascript
#!/usr/bin/env node
/**
 * Test script created by Codex agentic workflow
 */

function main() {
    console.log("Hello from Codex agentic workflow!");
    console.log(`Current time: ${new Date().toISOString()}`);
    console.log("This script was created to test push-to-branch functionality.");
}

if (require.main === module) {
    main();
}

module.exports = { main };
```

Create a commit message: "Add test files created by Codex agentic workflow"

Push these changes to the branch for the pull request #${github.event.pull_request.number}
