# Setup Safe Inputs Action

This action copies safe-inputs MCP server files to the agent environment.

## Description

The safe-inputs MCP server provides read operations for GitHub Actions workflows. This action copies all necessary JavaScript files to the agent environment so they can be used by the workflow.

## Usage

```yaml
- name: Setup Safe Inputs Files
  uses: ./actions/setup-safe-inputs
  with:
    # Destination directory for safe-inputs files
    # Default: /tmp/gh-aw/safe-inputs
    destination: /tmp/gh-aw/safe-inputs
```

## Inputs

### `destination`

**Optional** Destination directory for safe-inputs files.

Default: `/tmp/gh-aw/safe-inputs`

## Outputs

### `files-copied`

The number of files copied to the destination directory.

## Example

```yaml
steps:
  - uses: actions/checkout@v4
  
  - name: Setup Safe Inputs Files
    uses: ./actions/setup-safe-inputs
    
  - name: Use Safe Inputs
    run: |
      node /tmp/gh-aw/safe-inputs/safe_inputs_mcp_server.cjs
```

## Development

This action copies JavaScript files from the `js/` directory. To populate the `js/` directory with the latest files from `pkg/workflow/js/`:

```bash
make actions-build
```

The build process copies all required safe-inputs files to the `js/` directory. These files are committed to the repository for use in CI workflows.
