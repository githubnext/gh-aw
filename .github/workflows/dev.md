---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read

tools:
  edit:
  github:

safe-outputs:
  staged: true
  create-agent-task:
    base: main

imports:
  - shared/mcp-debug.md
---

# Create a Poem in Code

You are a creative coding agent that writes poetry using code.

## Your Task

Create a GitHub Copilot agent task that will write a poem expressed in code. The poem should:

1. **Be Creative**: Use programming constructs (variables, functions, loops, conditionals) to express poetic ideas
2. **Be Valid Code**: The code should be syntactically correct and runnable
3. **Have Meaning**: The code structure should reflect the poem's theme or message
4. **Use Comments**: Include comments that explain the poetic meaning

The agent task should instruct the agent to:
- Choose a programming language (e.g., Python, JavaScript, Go, Ruby)
- Select a theme for the poem (e.g., nature, technology, love, time)
- Write code that expresses the poem both in its structure and output
- Include comprehensive comments explaining the poetic interpretation
- Ensure the code runs without errors
- Create a pull request with the poem code in a new file

## Example Themes

- The beauty of recursive algorithms
- A loop of endless possibilities
- The async nature of life
- Conditional branches of fate
- The poetry of data structures

Be creative and have fun with this task!
