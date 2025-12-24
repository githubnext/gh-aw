// Safe Inputs Copy Action
// Copies safe-inputs MCP server files to the agent environment

const core = require('@actions/core');
const fs = require('fs');
const path = require('path');

// Embedded safe-inputs files will be inserted here during build
const FILES = {
    "mcp_logger.cjs": "",
    "mcp_server_core.cjs": "",
    "safe_inputs_bootstrap.cjs": "",
    "safe_inputs_config_loader.cjs": "",
    "safe_inputs_mcp_server.cjs": "",
    "safe_inputs_tool_factory.cjs": "",
    "safe_inputs_validation.cjs": ""
  };

async function run() {
  try {
    const destination = core.getInput('destination') || '/tmp/gh-aw/safe-inputs';
    
    core.info(`Copying safe-inputs files to ${destination}`);
    
    // Create destination directory if it doesn't exist
    if (!fs.existsSync(destination)) {
      fs.mkdirSync(destination, { recursive: true });
      core.info(`Created directory: ${destination}`);
    }
    
    let fileCount = 0;
    
    // Copy each embedded file
    for (const [filename, content] of Object.entries(FILES)) {
      const filePath = path.join(destination, filename);
      fs.writeFileSync(filePath, content, 'utf8');
      core.info(`Copied: ${filename}`);
      fileCount++;
    }
    
    core.setOutput('files-copied', fileCount.toString());
    core.info(`✓ Successfully copied ${fileCount} files`);
    
  } catch (error) {
    core.setFailed(`Action failed: ${error.message}`);
  }
}

run();
