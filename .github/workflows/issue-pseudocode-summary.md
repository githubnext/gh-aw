---
name: Issue Pseudocode Summary
description: Reads an issue and publishes a pseudo-code summary comment using the latest tag
on: 
  issues:
    types: [opened]
  workflow_dispatch:
engine: copilot
permissions:
  contents: read
  issues: read
runtimes:
  node:
    version: "22"
    action-tag: "latest"
tools:
  github:
safe-outputs:
  add-comment:
    hide-older-comments: false
---

# Issue Pseudocode Summary

Read the issue body and generate a concise pseudo-code summary of the problem or feature request.

## Instructions

1. Read the issue title and body
2. Analyze the content to understand the core problem or feature request
3. Generate a pseudo-code summary that:
   - Uses clear, structured pseudo-code syntax
   - Highlights key steps or logic flow
   - Is no more than 20 lines
   - Uses comments to explain intent
4. Publish the pseudo-code summary as a comment on the issue

## Example Format

```pseudocode
// Problem: [Brief description]
FUNCTION solve_problem():
    // Step 1: [Description]
    input = get_input()
    
    // Step 2: [Description]
    result = process(input)
    
    // Step 3: [Description]
    RETURN result
END FUNCTION
```

Keep the summary concise and focused on the core logic or steps needed.
