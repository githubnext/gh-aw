---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  scripts:
    echo-message:
      name: Echo Message
      description: Echo a message back as a noop confirmation
      inputs:
        message:
          description: Message to echo
          required: true
          type: string
      script: |
        return async function handleEchoMessage(item) {
          return { success: true, echoed: item.message };
        };
timeout-minutes: 5
---

# Test Copilot Safe-Outputs Scripts

Test the `safe-outputs.scripts` configuration which mounts custom inline
JavaScript handlers as MCP tools in the safe_outputs job.

Call the `echo_message` tool with:
- message: "Hello from scripts test"

Output as JSONL using the `echo_message` tool.
