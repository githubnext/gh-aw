# Create Issue Output

Output for creating a GitHub issue

## Overview

This action is generated from `pkg/workflow/js/create_issue.cjs` and provides functionality for GitHub Agentic Workflows.

## Usage

```yaml
- uses: ./actions/create_issue
  with:
    token: 'value'  # GitHub token for API authentication
```

## Inputs

### `token`

**Description**: GitHub token for API authentication

**Required**: true

## Outputs

### `issue_number`

**Description**: Output parameter: issue_number

### `issue_url`

**Description**: Output parameter: issue_url

### `issues_to_assign_copilot`

**Description**: Output parameter: issues_to_assign_copilot

### `temporary_id_map`

**Description**: Output parameter: temporary_id_map

## Dependencies

This action depends on the following JavaScript modules:

- `expiration_helpers.cjs`
- `generate_footer.cjs`
- `get_tracker_id.cjs`
- `load_agent_output.cjs`
- `repo_helpers.cjs`
- `sanitize_label_content.cjs`
- `staged_preview.cjs`
- `temporary_id.cjs`

## Development

### Building

To build this action, you need to:

1. Update the dependency mapping in `pkg/cli/actions_build_command.go` for `create_issue`
2. Run `make actions-build` to bundle the JavaScript dependencies
3. The bundled `index.js` will be generated and committed

### Testing

Test this action by creating a workflow:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: ./actions/create_issue
```

## License

MIT
