---
on:
  workflow_dispatch:
    inputs:
      topic:
        description: 'Documentation topic to review'
        required: true
        type: string

permissions: read-all

engine:
  id: copilot
  custom-agent: technical-doc-writer.md

network:
  allowed:
    - defaults
    - github

imports:
  - ../instructions/documentation.instructions.md

safe-outputs:
  add-comment:
    max: 1
  create-pull-request:
    title-prefix: "[docs] "
    labels: [documentation]
    reviewers: copilot
    draft: false
  upload-assets:

steps:
  - name: Setup Node.js
    uses: actions/setup-node@v6
    with:
      node-version: '24'
      cache: 'npm'
      cache-dependency-path: 'docs/package-lock.json'

  - name: Install dependencies
    working-directory: ./docs
    run: npm ci

  - name: Build documentation
    working-directory: ./docs
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: npm run build

tools:
  cache-memory: true
  github:
    allowed:
      - issue_read
      - get_pull_request
      - pull_request_read
      - get_file_contents
      - list_commits
      - add_reaction
  edit:
  bash:

timeout_minutes: 10

---

## Your Task

This workflow is triggered manually via workflow_dispatch with a documentation topic.

**Topic to review:** "${{ github.event.inputs.topic }}"

The documentation has been built successfully in the `docs/dist` folder. You can review both the source files in `docs/` and the built output in `docs/dist`.

**To run the Astro dev server locally for live preview:**
```bash
cd docs && npm run dev
```

When reviewing documentation for the specified topic in the **docs/** folder, apply these principles to:

1. **Analyze the topic** provided in the workflow input
2. **Review relevant documentation files** in the docs/ folder related to: "${{ github.event.inputs.topic }}"
3. **Verify the built documentation** in docs/dist is properly generated
4. **Provide constructive feedback** as a comment addressing:
   - Clarity and conciseness
   - Tone and voice consistency with GitHub Docs
   - Code block formatting and examples
   - Structure and organization
   - Developer experience considerations
   - Any missing prerequisites or setup steps
   - Appropriate use of admonitions
   - Link quality and accessibility
   - Build output quality and completeness
5. **Create a pull request with improvements** if you identify any changes needed:
   - Make the necessary edits to improve the documentation
   - Create a pull request with your changes using the safe-outputs create-pull-request functionality
   - Include a clear description of the improvements made
   - Only create a pull request if you have made actual changes to the documentation files

Keep your feedback specific, actionable, and empathetic. Focus on the most impactful improvements for the topic: "${{ github.event.inputs.topic }}"

You have access to cache-memory for persistent storage across runs, which you can use to track documentation patterns and improvement suggestions.
