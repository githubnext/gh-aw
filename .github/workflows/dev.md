---
on: 
  workflow_dispatch:
  reaction: "eyes"
  push:
    branches:
      - copilot/*
engine: claude
safe-outputs:
    staged: true
safe-jobs:
  print:
    #name: "print the message"
    runs-on: ubuntu-latest
    inputs:
      message:
        description: "Message to print"
        required: true
        type: string
    steps:
      - name: See artifacts
        run: cd /tmp/safe-jobs && find . -mindepth 1 -maxdepth 3 -type f
      - name: print message
        run: |
          
          if [ -f "$GITHUB_AW_AGENT_OUTPUT" ]; then
            MESSAGE=$(cat "$GITHUB_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "echo") | .message // "Hello from safe-job!"')
            echo "Echoing message: $MESSAGE"
          else
            echo "No agent output found, using default: Hello from safe-job!"
          fi
---
Summarize and use print the message using the `print` tool.

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.
