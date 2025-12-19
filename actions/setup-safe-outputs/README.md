# Setup Safe Outputs Action

This action copies safe-outputs MCP server files to the agent environment.

## Description

The safe-outputs MCP server provides write operations for GitHub Actions workflows. This action copies all necessary JavaScript files to the agent environment so they can be used by the workflow.

## Usage

```yaml
- name: Setup Safe Outputs Files
  uses: ./actions/setup-safe-outputs
  with:
    # Destination directory for safe-outputs files
    # Default: /tmp/gh-aw/safeoutputs
    destination: /tmp/gh-aw/safeoutputs
```

## Inputs

### `destination`

**Optional** Destination directory for safe-outputs files.

Default: `/tmp/gh-aw/safeoutputs`

## Outputs

### `files-copied`

The number of files copied to the destination directory.

## Example

```yaml
steps:
  - uses: actions/checkout@v4
  
  - name: Setup Safe Outputs Files
    uses: ./actions/setup-safe-outputs
    
  - name: Use Safe Outputs
    run: |
      node /tmp/gh-aw/safeoutputs/safe_outputs_mcp_server.cjs
```

## Development

This action uses a bash script to copy JavaScript files from the `js/` directory. The files are generated during the build process:

```bash
make actions-build
```

The build process copies all required JavaScript files from `pkg/workflow/js/` to `actions/setup-safe-outputs/js/`, and the bash script (`copy-files.sh`) copies them to the destination at runtime.
