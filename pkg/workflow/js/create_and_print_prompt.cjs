/** @type {typeof import("fs")} */
const fs = require("fs");

async function main() {
  // Environment validation - fail early if required variables are missing
  const promptPath = process.env.GITHUB_AW_PROMPT;
  if (!promptPath) {
    throw new Error("GITHUB_AW_PROMPT environment variable is required");
  }

  const promptContent = process.env.GITHUB_AW_PROMPT_CONTENT;
  if (!promptContent) {
    throw new Error("GITHUB_AW_PROMPT_CONTENT environment variable is required");
  }

  try {
    // Create the directory structure if it doesn't exist
    const promptDir = require("path").dirname(promptPath);
    fs.mkdirSync(promptDir, { recursive: true });

    // Write the prompt content to the file
    fs.writeFileSync(promptPath, promptContent, "utf8");
    
    core.info(`Prompt written to: ${promptPath}`);
    core.debug(`Prompt content length: ${promptContent.length} characters`);

    // Set the prompt as an environment variable using JSON.stringify for proper escaping
    const promptContentJson = JSON.stringify(promptContent);
    core.exportVariable("GITHUB_AW_PROMPT_JSON", promptContentJson);

    // Also set as step output for reference
    core.setOutput("prompt_file", promptPath);
    core.setOutput("prompt_content_json", promptContentJson);

    // Print prompt to GitHub step summary using core.summary API
    await core.summary
      .addRaw("## Generated Prompt\n\n")
      .addRaw("```markdown\n")
      .addRaw(promptContent)
      .addRaw("\n```")
      .write();

    core.info("Prompt successfully written to step summary");

  } catch (error) {
    core.setFailed(error instanceof Error ? error.message : String(error));
  }
}

await main();