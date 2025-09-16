---
on:
  workflow_dispatch: # do not remove this trigger
  push:
    branches:
      - copilot/*
      - pelikhan/*
tools:
  playwright:
    docker_image_version: "v1.41.0"
    allowed_domains: ["localhost", "127.0.0.1", "*.github.com", "github.com"]
safe-outputs:
  push-to-orphaned-branch:
    max: 3
  create-issue:
    max: 1
  missing-tool:
  staged: true
engine: 
  id: claude
  max-turns: 5
permissions: read-all
---

You have access to a `push-to-orphaned-branch` tool that can upload files (like screenshots) to an orphaned branch and return a GitHub raw URL. Use the expected URL from the response in your issue descriptions.

Please:
1. Build the documentation by running appropriate build commands
2. Take a screenshot of the documentation using playwright 
3. Upload the screenshot using the push-to-orphaned-branch tool - it will give you a URL to use
4. Create an issue describing the documentation status and include the screenshot using the URL provided by the upload tool