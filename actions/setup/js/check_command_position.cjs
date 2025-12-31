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

  // Map event types to their payload paths
  const eventTextPaths = {
    issues: () => context.payload.issue?.body,
    pull_request: () => context.payload.pull_request?.body,
    issue_comment: () => context.payload.comment?.body,
    pull_request_review_comment: () => context.payload.comment?.body,
    discussion: () => context.payload.discussion?.body,
    discussion_comment: () => context.payload.comment?.body,
  };

  const eventName = context.eventName;
  const textGetter = eventTextPaths[eventName];

  // For non-comment events, pass the check
  if (!textGetter) {
    core.info(`Event ${eventName} does not require command position check`);
    core.setOutput("command_position_ok", "true");
    return;
  }

  const text = textGetter() || "";
  const expectedCommand = `/${command}`;

  // If text is empty or doesn't contain the command at all, pass the check
  if (!text || !text.includes(expectedCommand)) {
    core.info(`No command '${expectedCommand}' found in text, passing check`);
    core.setOutput("command_position_ok", "true");
    return;
  }

  // Normalize whitespace and get the first word
  const firstWord = text.trim().split(/\s+/)[0];

  core.info(`Checking command position for: ${expectedCommand}`);
  core.info(`First word in text: ${firstWord}`);

  if (firstWord === expectedCommand) {
    core.info(`✓ Command '${expectedCommand}' is at the start of the text`);
    core.setOutput("command_position_ok", "true");
  } else {
    core.warning(
      `⚠️ Command '${expectedCommand}' is not the first word (found: '${firstWord}'). Workflow will be skipped.`
    );
    core.setOutput("command_position_ok", "false");
  }
}

module.exports = { main };
