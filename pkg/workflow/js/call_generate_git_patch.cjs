// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Call the safe outputs MCP server to generate a git patch
 * This script replaces the generate_git_patch.sh shell script
 */

const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");

async function main() {
  try {
    // Get MCP config path
    const mcpConfigPath = process.env.GH_AW_MCP_CONFIG || "/tmp/gh-aw/.copilot/mcp-config.json";
    const safeOutputsOutputFile = process.env.GH_AW_SAFE_OUTPUTS || "/tmp/gh-aw/safeoutputs/outputs.jsonl";

    // Check if MCP config exists
    if (!fs.existsSync(mcpConfigPath)) {
      core.warning("MCP config not found, cannot generate git patch via MCP server");
      return;
    }

    // Read MCP config to find the safe outputs server
    const mcpConfig = JSON.parse(fs.readFileSync(mcpConfigPath, "utf8"));
    const safeOutputsServer = mcpConfig.mcpServers?.safeoutputs;

    if (!safeOutputsServer) {
      core.warning("Safe outputs MCP server not configured");
      return;
    }

    core.info("Calling safe outputs MCP server to generate git patch...");

    // The safe outputs MCP server is already running as part of the workflow
    // We need to invoke it via the MCP client protocol
    // For now, we'll use the node module directly if it's a local server

    if (safeOutputsServer.type === "local" && safeOutputsServer.command === "node") {
      const serverScript = safeOutputsServer.args[0];
      core.info(`Safe outputs server script: ${serverScript}`);

      // Create a JSON-RPC request to call the generate_git_patch tool
      const request = {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: {
          name: "generate_git_patch",
          arguments: {},
        },
      };

      // Write request to a temp file
      const requestFile = "/tmp/gh-aw/mcp-request.jsonl";
      fs.writeFileSync(requestFile, JSON.stringify(request) + "\n");

      // Call the MCP server
      try {
        const response = execSync(`cat ${requestFile} | node ${serverScript}`, {
          encoding: "utf8",
          maxBuffer: 10 * 1024 * 1024, // 10MB
        });

        core.info(`MCP server response: ${response}`);

        // Parse the response
        const lines = response
          .split("\n")
          .filter(line => line.trim())
          .filter(line => !line.startsWith("[safeoutputs]")); // Filter out debug messages

        for (const line of lines) {
          try {
            const msg = JSON.parse(line);
            if (msg.result) {
              const resultText = msg.result.content?.[0]?.text;
              if (resultText) {
                const result = JSON.parse(resultText);
                core.info(`Patch generation result: ${result.result}`);

                if (result.result === "success") {
                  core.info(`Patch created at ${result.path}`);
                  core.info(`Branch: ${result.branch}, Base: ${result.base}`);
                  core.info(`Size: ${result.size_kb} KB, Lines: ${result.lines}, Commits: ${result.commits}`);
                } else if (result.result === "no_branch") {
                  core.info(result.message);
                } else if (result.result === "no_changes") {
                  core.info(result.message);
                }
              }
            } else if (msg.error) {
              core.error(`MCP server error: ${msg.error.message}`);
            }
          } catch (parseError) {
            // Skip non-JSON lines
            continue;
          }
        }
      } catch (execError) {
        core.error(`Failed to call MCP server: ${execError instanceof Error ? execError.message : String(execError)}`);
      }
    } else {
      core.warning("Safe outputs server is not a local Node.js server, falling back to direct implementation");

      // Fallback: implement the logic directly here (same as the handler)
      await generatePatchDirectly();
    }
  } catch (error) {
    core.error(`Error generating git patch: ${error instanceof Error ? error.message : String(error)}`);
  }
}

/**
 * Fallback implementation that directly generates the patch
 * This is the same logic as in the safe_outputs_mcp_server.cjs handler
 */
async function generatePatchDirectly() {
  const cwd = process.env.GITHUB_WORKSPACE || process.cwd();
  const patchPath = "/tmp/gh-aw/aw.patch";
  const defaultBranch = process.env.DEFAULT_BRANCH || "main";
  const outputFile = process.env.GH_AW_SAFE_OUTPUTS || "/tmp/gh-aw/safeoutputs/outputs.jsonl";

  try {
    // Read safe outputs to find branch name
    let branchName = null;
    if (fs.existsSync(outputFile)) {
      core.info("Reading safe outputs to find branch name...");
      const lines = fs.readFileSync(outputFile, "utf8").split("\n");

      for (const line of lines) {
        if (!line.trim()) continue;

        try {
          const entry = JSON.parse(line);
          // Check for create_pull_request or push_to_pull_request_branch
          if (entry.type === "create_pull_request" || entry.type === "push_to_pull_request_branch") {
            if (entry.branch) {
              branchName = entry.branch;
              core.info(`Found branch name from ${entry.type}: ${branchName}`);
              break;
            }
          }
        } catch (parseError) {
          // Skip invalid JSON lines
          continue;
        }
      }
    }

    if (!branchName) {
      core.info("No branch found in safe outputs, no patch generation");
      return;
    }

    core.info(`Looking for branch: ${branchName}`);

    // Check if the branch exists
    try {
      execSync(`git show-ref --verify --quiet "refs/heads/${branchName}"`, { cwd, encoding: "utf8", stdio: "pipe" });
      core.info(`Branch ${branchName} exists`);
    } catch (error) {
      core.info(`Branch ${branchName} does not exist`);
      return;
    }

    // Determine base ref for patch generation
    let baseRef;
    try {
      // Check if origin/<branchName> exists
      execSync(`git show-ref --verify --quiet "refs/remotes/origin/${branchName}"`, { cwd, encoding: "utf8", stdio: "pipe" });
      baseRef = `origin/${branchName}`;
      core.info(`Using origin/${branchName} as base for patch generation`);
    } catch (error) {
      // Use merge-base with default branch
      core.info(`origin/${branchName} does not exist, using merge-base with ${defaultBranch}`);

      // Fetch the default branch
      try {
        execSync(`git fetch origin ${defaultBranch}`, { cwd, encoding: "utf8", stdio: "pipe" });
      } catch (fetchError) {
        core.info(`Failed to fetch ${defaultBranch}: ${fetchError instanceof Error ? fetchError.message : String(fetchError)}`);
      }

      // Find merge base
      try {
        baseRef = execSync(`git merge-base "origin/${defaultBranch}" "${branchName}"`, {
          cwd,
          encoding: "utf8",
        }).trim();
        core.info(`Using merge-base as base: ${baseRef}`);
      } catch (mergeBaseError) {
        throw new Error(`Failed to find merge-base: ${mergeBaseError instanceof Error ? mergeBaseError.message : String(mergeBaseError)}`);
      }
    }

    // Count commits to be included
    let commitCount = 0;
    try {
      const countOutput = execSync(`git rev-list --count "${baseRef}..${branchName}"`, {
        cwd,
        encoding: "utf8",
      }).trim();
      commitCount = parseInt(countOutput, 10);
      core.info(`Number of commits to include: ${commitCount}`);
    } catch (countError) {
      core.info(`Failed to count commits: ${countError instanceof Error ? countError.message : String(countError)}`);
    }

    // Generate the patch
    core.info(`Generating patch: git format-patch ${baseRef}..${branchName} --stdout`);
    const patchContent = execSync(`git format-patch "${baseRef}..${branchName}" --stdout`, {
      cwd,
      encoding: "utf8",
      maxBuffer: 10 * 1024 * 1024, // 10MB buffer
    });

    // Check if patch has content
    if (!patchContent || !patchContent.trim()) {
      core.info("Generated patch is empty - no changes between base and branch");
      return;
    }

    // Ensure output directory exists
    const patchDir = path.dirname(patchPath);
    if (!fs.existsSync(patchDir)) {
      fs.mkdirSync(patchDir, { recursive: true });
    }

    // Write patch to file
    fs.writeFileSync(patchPath, patchContent, "utf8");

    const patchSizeKB = Math.ceil(Buffer.byteLength(patchContent, "utf8") / 1024);
    const patchLines = patchContent.split("\n").length;

    core.info(`Patch file created: ${patchPath}`);
    core.info(`Patch size: ${patchSizeKB} KB, ${patchLines} lines, ${commitCount} commits`);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to generate patch: ${errorMessage}`);
    throw error;
  }
}

main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
