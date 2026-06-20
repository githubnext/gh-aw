// @ts-check
/// <reference types="@actions/github-script" />

const { ERR_API, ERR_CONFIG, ERR_VALIDATION } = require("./error_codes.cjs");
const { writeDenialSummary } = require("./pre_activation_summary.cjs");
const { matchesCommandName, resolveMatchedCommand } = require("./slash_command_matcher.cjs");

/**
 * Check if command is the first word in the triggering text
 * This prevents accidental command triggers from words appearing later in content
 * Supports multiple command names - checks if any of them match
 */
async function main() {
  const commandsJSON = process.env.GH_AW_COMMANDS;

  const { getErrorMessage } = require("./error_helpers.cjs");

  if (!commandsJSON) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_COMMANDS not specified.`);
    return;
  }

  // Parse commands from JSON array
  let commands = [];
  try {
    commands = JSON.parse(commandsJSON);
    if (!Array.isArray(commands)) {
      core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_COMMANDS must be an array.`);
      return;
    }
  } catch (error) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: Failed to parse GH_AW_COMMANDS: ${getErrorMessage(error)}`);
    return;
  }

  if (commands.length === 0) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: No commands specified.`);
    return;
  }

  // Get the triggering text based on event type
  let text = "";
  const eventName = context.eventName;

  try {
    // For labeled events (label-command triggers), skip the command position check.
    // The label name itself is the trigger, not a slash command in the body.
    // Label name matching is enforced by the workflow-level `if:` condition
    // (e.g. github.event.label.name == 'cloclo'), so no additional filtering
    // is needed here regardless of which labels are configured.
    if (context.payload?.action === "labeled") {
      core.info(`Event ${eventName} with action 'labeled' does not require command position check`);
      core.setOutput("command_position_ok", "true");
      core.setOutput("matched_command", "");
      return;
    }

    if ((eventName === "pull_request" && ["ready_for_review", "review_requested"].includes(context.payload?.action ?? "")) || (eventName === "pull_request_review" && context.payload?.action === "submitted")) {
      core.info(`Event ${eventName} with action '${context.payload?.action ?? ""}' does not require command position check`);
      core.setOutput("command_position_ok", "true");
      core.setOutput("matched_command", "");
      return;
    }

    if (eventName === "issues") {
      text = context.payload.issue?.body || "";
    } else if (eventName === "pull_request") {
      text = context.payload.pull_request?.body || "";
    } else if (eventName === "issue_comment") {
      text = context.payload.comment?.body || "";
    } else if (eventName === "pull_request_review_comment") {
      text = context.payload.comment?.body || "";
    } else if (eventName === "pull_request_review") {
      text = context.payload.review?.body || "";
    } else if (eventName === "discussion") {
      text = context.payload.discussion?.body || "";
    } else if (eventName === "discussion_comment") {
      text = context.payload.comment?.body || "";
    } else if (eventName === "workflow_dispatch") {
      const rawAwContext = context.payload?.inputs?.aw_context ?? "";
      let inboundCommandName = "";
      if (typeof rawAwContext === "string" && rawAwContext.trim() !== "") {
        try {
          const parsed = JSON.parse(rawAwContext);
          if (parsed && typeof parsed === "object" && typeof parsed.command_name === "string") {
            inboundCommandName = parsed.command_name.trim();
          }
        } catch {
          // ignore malformed aw_context and fall back to manual workflow_dispatch behavior
        }
      }

      if (inboundCommandName) {
        if (commands.some(command => matchesCommandName(command, inboundCommandName))) {
          core.info(`✓ command_name '${inboundCommandName}' resolved from workflow_dispatch aw_context`);
          core.setOutput("command_position_ok", "true");
          core.setOutput("matched_command", inboundCommandName);
        } else {
          core.warning(`⚠️ command_name '${inboundCommandName}' from aw_context is not in allowed commands list.`);
          core.setOutput("command_position_ok", "false");
          core.setOutput("matched_command", "");
          await writeDenialSummary(`Workflow dispatch aw_context.command_name '${inboundCommandName}' is not one of the configured commands.`, "Ensure the centralized slash-command trigger dispatches only configured commands.");
        }
        return;
      }

      // Manual workflow_dispatch without aw_context.command_name is still allowed.
      core.info("workflow_dispatch without aw_context.command_name; skipping command position check");
      core.setOutput("command_position_ok", "true");
      core.setOutput("matched_command", "");
      return;
    } else {
      // For non-comment events, pass the check
      core.info(`Event ${eventName} does not require command position check`);
      core.setOutput("command_position_ok", "true");
      core.setOutput("matched_command", "");
      return;
    }

    // Normalize whitespace and resolve the matched slash command at the start of the text.
    const trimmedText = text.trim();
    const matchedCommand = resolveMatchedCommand(trimmedText, commands);
    const firstWord = trimmedText.split(/\s+/)[0];

    core.info(`Checking command position. First word in text: ${firstWord}`);
    core.info(`Looking for commands: ${commands.map(c => `/${c}`).join(", ")}`);

    if (matchedCommand) {
      core.info(`✓ Command '/${matchedCommand}' matched at the start of the text`);
      core.setOutput("command_position_ok", "true");
      core.setOutput("matched_command", matchedCommand);
    } else {
      const expectedCommands = commands.map(c => `/${c}`).join(", ");
      core.warning(`⚠️ None of the commands [${expectedCommands}] matched the first word (found: '${firstWord}'). Workflow will be skipped.`);
      core.setOutput("command_position_ok", "false");
      core.setOutput("matched_command", "");
      await writeDenialSummary(
        `The trigger comment did not start with a required command. Expected one of: ${expectedCommands}. Found: \`${firstWord}\`.`,
        "Make sure the trigger comment starts with the required command defined in `on.command:` in the workflow frontmatter."
      );
    }
  } catch (error) {
    core.setFailed(`${ERR_API}: ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
