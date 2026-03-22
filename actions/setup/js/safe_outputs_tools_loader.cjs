// @ts-check

const { getErrorMessage } = require("./error_helpers.cjs");
const { validateTargetRepo, parseAllowedRepos, getDefaultTargetRepo } = require("./repo_helpers.cjs");

const fs = require("fs");

/**
 * Load tools from tools.json file
 * @param {Object} server - The MCP server instance for logging
 * @returns {Array} Array of tool definitions
 */
function loadTools(server) {
  const toolsPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH || `${process.env.RUNNER_TEMP}/gh-aw/safeoutputs/tools.json`;

  server.debug(`Reading tools from file: ${toolsPath}`);

  if (!fs.existsSync(toolsPath)) {
    server.debug(`Tools file does not exist at: ${toolsPath}`);
    server.debug(`Using empty tools array`);
    return [];
  }

  try {
    server.debug(`Tools file exists at: ${toolsPath}`);
    const toolsFileContent = fs.readFileSync(toolsPath, "utf8");
    server.debug(`Tools file content length: ${toolsFileContent.length} characters`);
    server.debug(`Tools file read successfully, attempting to parse JSON`);
    const tools = JSON.parse(toolsFileContent);
    server.debug(`Successfully parsed ${tools.length} tools from file`);

    // Log details about dispatch_workflow tools for debugging
    const dispatchWorkflowTools = tools.filter(t => t._workflow_name);
    if (dispatchWorkflowTools.length > 0) {
      server.debug(`  Found ${dispatchWorkflowTools.length} dispatch_workflow tools:`);
      dispatchWorkflowTools.forEach(t => {
        server.debug(`    - ${t.name} (workflow: ${t._workflow_name})`);
      });
    }

    // Log details about dispatch_repository tools for debugging
    const dispatchRepositoryTools = tools.filter(t => t._dispatch_repository_tool);
    if (dispatchRepositoryTools.length > 0) {
      server.debug(`  Found ${dispatchRepositoryTools.length} dispatch_repository tools:`);
      dispatchRepositoryTools.forEach(t => {
        server.debug(`    - ${t.name} (tool: ${t._dispatch_repository_tool})`);
      });
    }

    // Log details about call_workflow tools for debugging
    const callWorkflowTools = tools.filter(t => t._call_workflow_name);
    if (callWorkflowTools.length > 0) {
      server.debug(`  Found ${callWorkflowTools.length} call_workflow tools:`);
      callWorkflowTools.forEach(t => {
        server.debug(`    - ${t.name} (workflow: ${t._call_workflow_name})`);
      });
    }

    return tools;
  } catch (error) {
    server.debug(`Error reading tools file: ${getErrorMessage(error)}`);
    server.debug(`Falling back to empty tools array`);
    return [];
  }
}

/**
 * Attach handlers to tools
 * @param {Array} tools - Array of tool definitions
 * @param {Object} handlers - Object containing handler functions
 * @returns {Array} Tools with handlers attached
 */
function attachHandlers(tools, handlers) {
  const handlerMap = {
    create_pull_request: handlers.createPullRequestHandler,
    push_to_pull_request_branch: handlers.pushToPullRequestBranchHandler,
    push_repo_memory: handlers.pushRepoMemoryHandler,
    upload_asset: handlers.uploadAssetHandler,
    create_project: handlers.createProjectHandler,
    add_comment: handlers.addCommentHandler,
  };

  tools.forEach(tool => {
    const handler = handlerMap[tool.name];
    if (handler) {
      tool.handler = handler;
    }

    // Check if this is a dispatch_workflow tool (dynamic tool with workflow metadata)
    if (tool._workflow_name) {
      // Create a custom handler that wraps args in inputs and adds workflow_name
      const workflowName = tool._workflow_name;
      tool.handler = args => {
        // Wrap args in inputs property to match dispatch_workflow schema
        return handlers.defaultHandler("dispatch_workflow")({
          inputs: args,
          workflow_name: workflowName,
        });
      };
    }

    // Check if this is a dispatch_repository tool (dynamic tool with dispatch_repository metadata)
    if (tool._dispatch_repository_tool) {
      const toolKey = tool._dispatch_repository_tool;
      tool.handler = args => {
        return handlers.defaultHandler("dispatch_repository")({
          inputs: args,
          tool_name: toolKey,
        });
      };
    }

    // Check if this is a call_workflow tool (dynamic tool with call workflow metadata)
    if (tool._call_workflow_name) {
      // Create a custom handler that wraps args in inputs and adds workflow_name
      const workflowName = tool._call_workflow_name;
      tool.handler = args => {
        // Wrap args in inputs property to match call_workflow schema
        return handlers.defaultHandler("call_workflow")({
          inputs: args,
          workflow_name: workflowName,
        });
      };
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
    // Check if this is a regular tool matching a config key
    if (Object.keys(config).find(configKey => normalizeTool(configKey) === tool.name)) {
      let toolToRegister = tool;
      // Enrich create_pull_request tool description when target-repo is configured
      if (tool.name === "create_pull_request" && config.create_pull_request) {
        const targetRepo = config.create_pull_request["target-repo"];
        if (targetRepo) {
          // Validate the configured target-repo against the allowed-repos list
          const allowedRepos = parseAllowedRepos(config.create_pull_request.allowed_repos);
          if (allowedRepos.size > 0) {
            const defaultRepo = getDefaultTargetRepo(config.create_pull_request);
            const validation = validateTargetRepo(targetRepo, defaultRepo, allowedRepos);
            if (!validation.valid) {
              server.debug(`WARNING: SEC-005: ${validation.error}`);
            }
          }
          toolToRegister = JSON.parse(JSON.stringify(tool));
          toolToRegister.description += ` Note: This workflow is configured to create pull requests in '${targetRepo}'. You do not need to specify the repo parameter.`;
          if (toolToRegister.inputSchema && toolToRegister.inputSchema.properties && toolToRegister.inputSchema.properties.repo) {
            toolToRegister.inputSchema.properties.repo.description += ` Configured default: '${targetRepo}'.`;
          }
        }
      }
      registerTool(server, toolToRegister);
      return;
    }

    // Check if this is a dispatch_workflow tool (has _workflow_name metadata)
    // These tools are dynamically generated with workflow-specific names
    if (tool._workflow_name) {
      server.debug(`Found dispatch_workflow tool: ${tool.name} (_workflow_name: ${tool._workflow_name})`);
      if (config.dispatch_workflow) {
        server.debug(`  dispatch_workflow config exists, registering tool`);
        registerTool(server, tool);
        return;
      } else {
        // Note: Using server.debug() with "WARNING:" prefix since MCP server only provides
        // debug and debugError methods. The prefix helps identify severity in logs.
        server.debug(`  WARNING: dispatch_workflow config is missing or falsy - tool will NOT be registered`);
        server.debug(`  Config keys: ${Object.keys(config).join(", ")}`);
        server.debug(`  config.dispatch_workflow value: ${JSON.stringify(config.dispatch_workflow)}`);
      }
    }

    // Check if this is a dispatch_repository tool (has _dispatch_repository_tool metadata)
    // These tools are dynamically generated with tool-specific names
    if (tool._dispatch_repository_tool) {
      server.debug(`Found dispatch_repository tool: ${tool.name} (_dispatch_repository_tool: ${tool._dispatch_repository_tool})`);
      if (config.dispatch_repository) {
        server.debug(`  dispatch_repository config exists, registering tool`);
        registerTool(server, tool);
        return;
      } else {
        server.debug(`  WARNING: dispatch_repository config is missing or falsy - tool will NOT be registered`);
        server.debug(`  Config keys: ${Object.keys(config).join(", ")}`);
        server.debug(`  config.dispatch_repository value: ${JSON.stringify(config.dispatch_repository)}`);
      }
    }

    // Check if this is a call_workflow tool (has _call_workflow_name metadata)
    // These tools are dynamically generated with workflow-specific names
    if (tool._call_workflow_name) {
      server.debug(`Found call_workflow tool: ${tool.name} (_call_workflow_name: ${tool._call_workflow_name})`);
      if (config.call_workflow) {
        server.debug(`  call_workflow config exists, registering tool`);
        registerTool(server, tool);
        return;
      } else {
        server.debug(`  WARNING: call_workflow config is missing or falsy - tool will NOT be registered`);
        server.debug(`  Config keys: ${Object.keys(config).join(", ")}`);
        server.debug(`  config.call_workflow value: ${JSON.stringify(config.call_workflow)}`);
      }
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
    if (server.tools[normalizedKey] || tools.find(t => t.name === normalizedKey)) {
      return;
    }

    const jobConfig = config[configKey];

    // Create a dynamic tool for this safe-job
    const dynamicTool = {
      name: normalizedKey,
      description: jobConfig?.description ?? `Custom safe-job: ${configKey}`,
      inputSchema: {
        type: "object",
        properties: {},
        additionalProperties: true, // Allow any properties for flexibility
      },
      handler: args => {
        // Create a generic safe-job output entry
        const entry = { type: normalizedKey, ...args };

        // Write the entry to the output file in JSONL format
        // CRITICAL: Use JSON.stringify WITHOUT formatting parameters for JSONL format
        // Each entry must be on a single line, followed by a newline character
        fs.appendFileSync(outputFile, `${JSON.stringify(entry)}\n`);

        // Use output from safe-job config if available
        const outputText = jobConfig?.output ?? `Safe-job '${configKey}' executed successfully with arguments: ${JSON.stringify(args)}`;

        return {
          content: [{ type: "text", text: JSON.stringify({ result: outputText }) }],
        };
      },
    };

    // Add input schema based on job configuration if available
    if (jobConfig?.inputs) {
      dynamicTool.inputSchema.properties = {};
      dynamicTool.inputSchema.required = [];

      Object.keys(jobConfig.inputs).forEach(inputName => {
        const inputDef = jobConfig.inputs[inputName];

        // Convert GitHub Actions choice type to JSON Schema string type
        // GitHub Actions uses "choice" type with "options" array
        // JSON Schema requires "string" type with "enum" array
        let jsonSchemaType = inputDef.type || "string";
        if (jsonSchemaType === "choice") {
          jsonSchemaType = "string";
        }

        const propSchema = {
          type: jsonSchemaType,
          description: inputDef.description || `Input parameter: ${inputName}`,
        };

        if (Array.isArray(inputDef.options)) {
          propSchema.enum = inputDef.options;
        }

        dynamicTool.inputSchema.properties[inputName] = propSchema;

        if (inputDef.required) {
          dynamicTool.inputSchema.required.push(inputName);
        }
      });
    }

    registerTool(server, dynamicTool);
  });
}

module.exports = {
  loadTools,
  attachHandlers,
  registerPredefinedTools,
  registerDynamicTools,
};
