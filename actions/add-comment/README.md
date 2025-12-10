# Add Issue Comment Output

Output for adding a comment to an issue or pull request

## Overview

This action is generated from `pkg/workflow/js/add_comment.cjs` and provides functionality for GitHub Agentic Workflows.

## Usage

```yaml
- uses: ./actions/add_comment
  with:
    token: 'value'  # GitHub token for API authentication
```

## Inputs

### `token`

**Description**: GitHub token for API authentication

**Required**: true

## Outputs

### `comment_id`

**Description**: Output parameter: comment_id

### `comment_url`

**Description**: Output parameter: comment_url

## Dependencies

This action depends on the following JavaScript modules:

- `get_repository_url.cjs`
- `get_tracker_id.cjs`
- `load_agent_output.cjs`
- `messages_footer.cjs`
- `temporary_id.cjs`

## Development

### Building

To build this action, you need to:

1. Update the dependency mapping in `pkg/cli/actions_build_command.go` for `add_comment`
2. Run `make actions-build` to bundle the JavaScript dependencies
3. The bundled `index.js` will be generated and committed

### Testing

Test this action by creating a workflow:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: ./actions/add_comment
```

## License

MIT
