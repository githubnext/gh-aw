const fs = require("fs");
const path = require("path");

async function main() {
  // Initialize outputs to empty strings to ensure they're always set
  core.setOutput("task_number", "");
  core.setOutput("task_url", "");

  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";
  const agentOutputFile = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!agentOutputFile) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  // Read agent output from file
  let outputContent;
  try {
    outputContent = fs.readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    core.setFailed(`Error reading agent output file: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }
  core.info(`Agent output content length: ${outputContent.length}`);
  
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }
  
  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }
  
  const createAgentTaskItems = validatedOutput.items.filter(item => item.type === "create_agent_task");
  if (createAgentTaskItems.length === 0) {
    core.info("No create-agent-task items found in agent output");
    return;
  }
  
  core.info(`Found ${createAgentTaskItems.length} create-agent-task item(s)`);
  
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Create Agent Tasks Preview\n\n";
    summaryContent += "The following agent tasks would be created if staged mode was disabled:\n\n";
    
    for (const [index, item] of createAgentTaskItems.entries()) {
      summaryContent += `### Task ${index + 1}\n\n`;
      summaryContent += `**Description:**\n${item.body || "No description provided"}\n\n`;
      
      const baseBranch = process.env.GITHUB_AW_AGENT_TASK_BASE || "main";
      summaryContent += `**Base Branch:** ${baseBranch}\n\n`;
      
      const targetRepo = process.env.GITHUB_AW_TARGET_REPO || process.env.GITHUB_REPOSITORY || "unknown";
      summaryContent += `**Target Repository:** ${targetRepo}\n\n`;
      
      summaryContent += "---\n\n";
    }
    
    core.summary.addRaw(summaryContent);
    await core.summary.write();
    return;
  }

  // Get base branch from environment or use current branch
  const baseBranch = process.env.GITHUB_AW_AGENT_TASK_BASE || process.env.GITHUB_REF_NAME || "main";
  const targetRepo = process.env.GITHUB_AW_TARGET_REPO;
  
  // Process the first agent task item (max is 1)
  const taskItem = createAgentTaskItems[0];
  const taskDescription = taskItem.body;
  
  if (!taskDescription || taskDescription.trim() === "") {
    core.setFailed("Agent task description is empty");
    return;
  }
  
  try {
    // Write task description to a temporary file
    const tmpDir = "/tmp/gh-aw";
    if (!fs.existsSync(tmpDir)) {
      fs.mkdirSync(tmpDir, { recursive: true });
    }
    
    const taskFile = path.join(tmpDir, "agent-task-description.md");
    fs.writeFileSync(taskFile, taskDescription, "utf8");
    core.info(`Task description written to ${taskFile}`);
    
    // Build gh agent-task create command
    const ghArgs = ["agent-task", "create", "--from-file", taskFile, "--base", baseBranch];
    
    if (targetRepo) {
      ghArgs.push("--repo", targetRepo);
    }
    
    core.info(`Creating agent task with command: gh ${ghArgs.join(" ")}`);
    
    // Execute gh agent-task create command
    let taskOutput;
    try {
      taskOutput = await exec.getExecOutput("gh", ghArgs, {
        silent: false,
        ignoreReturnCode: false
      });
    } catch (execError) {
      core.setFailed(`Failed to create agent task: ${execError instanceof Error ? execError.message : String(execError)}`);
      return;
    }
    
    // Parse the output to extract task number and URL
    // Expected output format from gh agent-task create is typically:
    // https://github.com/owner/repo/issues/123
    const output = taskOutput.stdout.trim();
    core.info(`Agent task created: ${output}`);
    
    // Extract task number from URL
    const urlMatch = output.match(/github\.com\/[^/]+\/[^/]+\/issues\/(\d+)/);
    if (urlMatch) {
      const taskNumber = urlMatch[1];
      core.setOutput("task_number", taskNumber);
      core.setOutput("task_url", output);
      
      core.summary.addRaw(`## ✅ Agent Task Created\n\n`);
      core.summary.addRaw(`**Task:** [#${taskNumber}](${output})\n\n`);
      core.summary.addRaw(`**Base Branch:** ${baseBranch}\n\n`);
      await core.summary.write();
      
      core.info(`✅ Successfully created agent task #${taskNumber}`);
    } else {
      core.warning(`Could not parse task number from output: ${output}`);
      core.setOutput("task_url", output);
    }
    
  } catch (error) {
    core.setFailed(`Error creating agent task: ${error instanceof Error ? error.message : String(error)}`);
  }
}

main().catch((error) => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
