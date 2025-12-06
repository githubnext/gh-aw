---
name: Test Multi-Model Selection
description: Test workflow to verify multi-model selection with wildcards works correctly
on: workflow_dispatch
engine:
  id: copilot
  model:
    - "gpt-5"
    - "gpt-4o"
    - "gpt-*-mini"
---

# Test Multi-Model Selection

This is a simple test workflow to verify that multi-model selection works correctly.

The workflow should:
1. Validate secrets
2. Select the first available model from the list (gpt-5, gpt-4o, or any gpt-*-mini model)
3. Use the selected model for execution

Please respond with a simple message confirming which model was selected.
