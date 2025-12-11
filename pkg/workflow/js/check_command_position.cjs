// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check if command is the first word in the triggering text
 * This prevents accidental command triggers from words appearing later in content
 */
async function main() {
  const command = process.env.GH_AW_COMMAND;

  if (!command) {
    core.setFailed("Configuration error: GH_AW_COMMAND not specified.");
    return;
  }

  // Get the triggering text based on event type
  let text = "";
  const eventName = context.eventName;

  try {
    if (eventName === "issues") {
      text = context.payload.issue?.body || "";
    } else if (eventName === "pull_request") {
      text = context.payload.pull_request?.body || "";
    } else if (eventName === "issue_comment") {
      text = context.payload.comment?.body || "";
    } else if (eventName === "pull_request_review_comment") {
      text = context.payload.comment?.body || "";
    } else if (eventName === "discussion") {
      text = context.payload.discussion?.body || "";
    } else if (eventName === "discussion_comment") {
      text = context.payload.comment?.body || "";
    } else {
      // For non-comment events, pass the check
      core.info(`Event ${eventName} does not require command position check`);
      core.setOutput("command_position_ok", "true");
      return;
    }

    // Expected command format: /command
    const expectedCommand = `/${command}`;

    // If text is empty or doesn't contain the command at all, pass the check
    if (!text || !text.includes(expectedCommand)) {
      core.info(`No command '${expectedCommand}' found in text, passing check`);
      core.setOutput("command_position_ok", "true");
      return;
    }

    // Normalize whitespace and get the first word
    const trimmedText = text.trim();
    const firstWord = trimmedText.split(/\s+/)[0];

    core.info(`Checking command position for: ${expectedCommand}`);
    core.info(`First word in text: ${firstWord}`);

    if (firstWord === expectedCommand) {
      core.info(`✓ Command '${expectedCommand}' is at the start of the text`);
      core.setOutput("command_position_ok", "true");
    } else {
      core.warning(`⚠️ Command '${expectedCommand}' is not the first word (found: '${firstWord}'). Workflow will be skipped.`);
      core.setOutput("command_position_ok", "false");
    }
  } catch (error) {
    core.setFailed(error instanceof Error ? error.message : String(error));
  }
}

await main();
