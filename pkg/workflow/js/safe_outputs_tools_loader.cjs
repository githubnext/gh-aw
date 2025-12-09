// @ts-check

const fs = require("fs");

/**
 * Load tools from tools.json file
 * @param {Object} server - The MCP server instance for logging
 * @returns {Array} Array of tool definitions
 */
function loadTools(server) {
  const toolsPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH || "/tmp/gh-aw/safeoutputs/tools.json";
  let ALL_TOOLS = [];

  server.debug(`Reading tools from file: ${toolsPath}`);

  try {
    if (fs.existsSync(toolsPath)) {
      server.debug(`Tools file exists at: ${toolsPath}`);
      const toolsFileContent = fs.readFileSync(toolsPath, "utf8");
      server.debug(`Tools file content length: ${toolsFileContent.length} characters`);
      server.debug(`Tools file read successfully, attempting to parse JSON`);
      ALL_TOOLS = JSON.parse(toolsFileContent);
      server.debug(`Successfully parsed ${ALL_TOOLS.length} tools from file`);
    } else {
      server.debug(`Tools file does not exist at: ${toolsPath}`);
      server.debug(`Using empty tools array`);
      ALL_TOOLS = [];
    }
  } catch (error) {
    server.debug(`Error reading tools file: ${error instanceof Error ? error.message : String(error)}`);
    server.debug(`Falling back to empty tools array`);
    ALL_TOOLS = [];
  }

  return ALL_TOOLS;
}

/**
 * Attach handlers to tools
 * @param {Array} tools - Array of tool definitions
 * @param {Object} handlers - Object containing handler functions
 * @returns {Array} Tools with handlers attached
 */
function attachHandlers(tools, handlers) {
  tools.forEach(tool => {
    if (tool.name === "create_pull_request") {
      tool.handler = handlers.createPullRequestHandler;
    } else if (tool.name === "push_to_pull_request_branch") {
      tool.handler = handlers.pushToPullRequestBranchHandler;
    } else if (tool.name === "upload_asset") {
      tool.handler = handlers.uploadAssetHandler;
    }
  });
  return tools;
}

/**
 * Register predefined tools based on configuration
 * @param {Object} server - The MCP server instance
 * @param {Array} tools - Array of tool definitions
 * @param {Object} config - Safe outputs configuration
 * @param {Function} registerTool - Function to register a tool
 * @param {Function} normalizeTool - Function to normalize tool names
 */
function registerPredefinedTools(server, tools, config, registerTool, normalizeTool) {
  tools.forEach(tool => {
    if (Object.keys(config).find(configKey => normalizeTool(configKey) === tool.name)) {
      registerTool(server, tool);
    }
  });
}

/**
 * Register dynamic safe-job tools based on configuration
 * @param {Object} server - The MCP server instance
 * @param {Array} tools - Array of predefined tool definitions
 * @param {Object} config - Safe outputs configuration
 * @param {string} outputFile - Path to the output file
 * @param {Function} registerTool - Function to register a tool
 * @param {Function} normalizeTool - Function to normalize tool names
 */
function registerDynamicTools(server, tools, config, outputFile, registerTool, normalizeTool) {
  Object.keys(config).forEach(configKey => {
    const normalizedKey = normalizeTool(configKey);

    // Skip if it's already a predefined tool
    if (server.tools[normalizedKey]) {
      return;
    }

    // Check if this is a safe-job (not in ALL_TOOLS)
    if (!tools.find(t => t.name === normalizedKey)) {
      const jobConfig = config[configKey];

      // Create a dynamic tool for this safe-job
      const dynamicTool = {
        name: normalizedKey,
        description: jobConfig && jobConfig.description ? jobConfig.description : `Custom safe-job: ${configKey}`,
        inputSchema: {
          type: "object",
          properties: {},
          additionalProperties: true, // Allow any properties for flexibility
        },
        handler: args => {
          // Create a generic safe-job output entry
          const entry = {
            type: normalizedKey,
            ...args,
          };

          // Write the entry to the output file in JSONL format
          // CRITICAL: Use JSON.stringify WITHOUT formatting parameters for JSONL format
          // Each entry must be on a single line, followed by a newline character
          const entryJSON = JSON.stringify(entry);
          fs.appendFileSync(outputFile, entryJSON + "\n");

          // Use output from safe-job config if available
          const outputText =
            jobConfig && jobConfig.output
              ? jobConfig.output
              : `Safe-job '${configKey}' executed successfully with arguments: ${JSON.stringify(args)}`;

          return {
            content: [
              {
                type: "text",
                text: JSON.stringify({ result: outputText }),
              },
            ],
          };
        },
      };

      // Add input schema based on job configuration if available
      if (jobConfig && jobConfig.inputs) {
        dynamicTool.inputSchema.properties = {};
        dynamicTool.inputSchema.required = [];

        Object.keys(jobConfig.inputs).forEach(inputName => {
          const inputDef = jobConfig.inputs[inputName];
          const propSchema = {
            type: inputDef.type || "string",
            description: inputDef.description || `Input parameter: ${inputName}`,
          };

          if (inputDef.options && Array.isArray(inputDef.options)) {
            propSchema.enum = inputDef.options;
          }

          dynamicTool.inputSchema.properties[inputName] = propSchema;

          if (inputDef.required) {
            dynamicTool.inputSchema.required.push(inputName);
          }
        });
      }

      registerTool(server, dynamicTool);
    }
  });
}

module.exports = {
  loadTools,
  attachHandlers,
  registerPredefinedTools,
  registerDynamicTools,
};
