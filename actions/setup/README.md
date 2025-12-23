# Setup Action

This action copies workflow script files to the agent environment.

## Description

This action runs in all workflow jobs to provide JavaScript scripts that can be required instead of being inlined in the workflow. This includes scripts for activation jobs, agent jobs, and safe-output jobs.

## Usage

```yaml
- name: Setup Scripts
  uses: ./actions/setup
  with:
    # Destination directory for script files
    # Default: /tmp/gh-aw/actions/activation
    destination: /tmp/gh-aw/actions/activation
```

## Inputs

### `destination`

**Optional** Destination directory for script files.

Default: `/tmp/gh-aw/actions/activation`

## Outputs

### `files-copied`

The number of files copied to the destination directory.

## Example

```yaml
steps:
  - uses: actions/checkout@v4
  
  - name: Setup Scripts
    uses: ./actions/setup
    with:
      destination: /tmp/gh-aw/actions/activation
```

## Files Included

This action copies all .cjs files from the script registry, including:

- `check_stop_time.cjs` - Check stop-time limit script
- `check_skip_if_match.cjs` - Check skip-if-match query script
- `check_command_position.cjs` - Check command position script
- `check_workflow_timestamp_api.cjs` - Check workflow file timestamps script
- `lock-issue.cjs` - Lock issue for agent workflow script
- `compute_text.cjs` - Compute current body text script (bundled with dependencies)
- `add_reaction_and_edit_comment.cjs` - Add reaction and edit comment script (bundled with dependencies)
- And all other registered .cjs files (82 total)
