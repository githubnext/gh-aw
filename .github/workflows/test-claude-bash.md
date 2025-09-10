---
engine: claude
on:
  workflow_dispatch:
permissions:
  contents: read
tools:
  bash:
    timeout: 300
    allowed: 
      - "gh aw logs"
---

# Test Claude Bash - Agentic Activity Overview


Please run `gh aw logs -c 1000` and provide a brief summary of the agentic workflow activity.

Look at the output and tell me:
1. How many workflow runs you can see
2. Which workflows appear most frequently 
3. Any patterns in success/failure rates

Use the bash tool to run the command and analyze the results.
