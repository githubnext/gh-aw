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
