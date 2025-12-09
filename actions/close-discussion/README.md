# Close Discussion Output

Output for closing a GitHub discussion with an optional comment and resolution reason

## Overview

This action is generated from `pkg/workflow/js/close_discussion.cjs` and provides functionality for GitHub Agentic Workflows.

## Usage

```yaml
- uses: ./actions/close_discussion
  with:
    token: 'value'  # GitHub token for API authentication
```

## Inputs

### `token`

**Description**: GitHub token for API authentication

**Required**: true

## Outputs

### `comment_url`

**Description**: Output parameter: comment_url

### `discussion_number`

**Description**: Output parameter: discussion_number

### `discussion_url`

**Description**: Output parameter: discussion_url

## Dependencies

This action depends on the following JavaScript modules:

- `generate_footer.cjs`
- `get_repository_url.cjs`
- `get_tracker_id.cjs`
- `load_agent_output.cjs`

## Development

### Building

To build this action, you need to:

1. Update the dependency mapping in `pkg/cli/actions_build_command.go` for `close_discussion`
2. Run `make actions-build` to bundle the JavaScript dependencies
3. The bundled `index.js` will be generated and committed

### Testing

Test this action by creating a workflow:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: ./actions/close_discussion
```

## License

MIT
