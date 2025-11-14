---
on:
  workflow_dispatch:
    inputs:
      workflow_name:
        description: "Name of the workflow to migrate from githubnext/gh-aw (e.g., 'triage-issues' or 'triage-issues.md')"
        required: true
        type: string
permissions: read-all
timeout-minutes: 15
network:
  allowed:
    - node
    - raw.githubusercontent.com
steps:
  - name: Install gh-aw extension
    run: gh extension install githubnext/gh-aw
    env:
      GH_TOKEN: ${{ github.token }}
tools:
  github:
    allowed:
      - get_file_contents
  edit:
  web-fetch:
  bash:
    - "*"
safe-outputs:
  create-pull-request:
source: githubnext/agentics/.github/workflows/migrate-workflow.md@9586b5fc47d008cd1cf42f6c298a46abfd774fb5
---
# Migrate Agentic Workflow from githubnext/gh-aw

You are tasked with migrating an agentic workflow from the **githubnext/gh-aw** repository to this repository.

## Workflow to Migrate

Target workflow: **${{ inputs.workflow_name }}**

## Migration Steps

1. **Normalize the workflow name**:
   - If the input ends with `.md`, use it as-is
   - Otherwise, append `.md` to the workflow name
   - Store the normalized name (e.g., `triage-issues.md`)

2. **Fetch the workflow from githubnext/gh-aw**:
   - Use the GitHub tool to fetch the content from `githubnext/gh-aw` repository
   - Path: `.github/workflows/<workflow_name>`
   - If the workflow is not found, try searching in subdirectories

3. **Identify shared workflow dependencies**:
   - Scan the fetched workflow content for any `imports:` sections
   - Make a list of all shared workflow files referenced (these are typically in `.github/workflows/shared/`)

4. **Fetch all shared workflows**:
   - For each shared workflow identified in the imports:
     - Fetch it from `githubnext/gh-aw` at path `.github/workflows/shared/<shared-workflow-name>`
     - Save it to `.github/workflows/shared/<shared-workflow-name>` in this repository

5. **Save the main workflow**:
   - Write the main workflow content to `workflows/<workflow_name>` (note: `workflows/` not `.github/workflows/`)
   - Ensure the file is saved with the correct name

6. **Update the source field**:
   - If the workflow has a `source:` field in its frontmatter, update it to reflect the migration
   - Add or update it to: `source: githubnext/gh-aw/.github/workflows/<workflow_name>@main`

7. **Compile the workflow**:
   - **IMPORTANT**: Use the globally installed `gh aw` CLI (via `which gh`), NOT any locally built version from the source repository
   - Run `gh aw compile workflows/<workflow_name>` to generate the lock file
   - This will validate the syntax and create `workflows/<workflow_name>.lock.yml`

8. **Report results**:
   - Confirm successful migration with a summary:
     - ✅ Main workflow: `workflows/<workflow_name>`
     - ✅ Shared workflows imported: [list them]
     - ✅ Compiled lock file: `workflows/<workflow_name>.lock.yml`
   - If any errors occurred, report them clearly
   - Remind the user to commit and push the changes to activate the workflow

## Security Considerations

- Overwrite existing files if they already exist (as per user instruction)
- Maintain the original workflow's permissions and security settings
- Ensure all network access patterns are preserved

## Error Handling

If the workflow is not found in githubnext/gh-aw:
- Check if the user provided the correct name
- Suggest using `gh aw list` or checking the githubnext/gh-aw repository directly
- List available workflows if possible
