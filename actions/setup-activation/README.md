# Setup Activation Action

This action copies activation job files to the agent environment.

## Description

The activation job runs JavaScript scripts and shell scripts to check permissions, timestamps, and reactions. This action copies all necessary files to the agent environment so they can be required or executed instead of being inlined in the workflow.

## Usage

```yaml
- name: Setup Activation Files
  uses: ./actions/setup-activation
  with:
    # Destination directory for activation files
    # Default: /tmp/gh-aw/actions/activation
    destination: /tmp/gh-aw/actions/activation
```

## Inputs

### `destination`

**Optional** Destination directory for activation files.

Default: `/tmp/gh-aw/actions/activation`

## Outputs

### `files-copied`

The number of files copied to the destination directory.

## Example

```yaml
steps:
  - uses: actions/checkout@v4
  
  - name: Setup Activation Files
    uses: ./actions/setup-activation
    with:
      destination: /tmp/gh-aw/actions/activation
```

## Files Included

This action copies the following files:

- `check_stop_time.cjs` - Check stop-time limit script
- `check_skip_if_match.cjs` - Check skip-if-match query script
- `check_command_position.cjs` - Check command position script
- `check_workflow_timestamp_api.cjs` - Check workflow file timestamps script
- `lock-issue.cjs` - Lock issue for agent workflow script
- `compute_text.cjs` - Compute current body text script (bundled with dependencies)
- `add_reaction_and_edit_comment.cjs` - Add reaction and edit comment script (bundled with dependencies)
