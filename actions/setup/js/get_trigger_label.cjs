// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Emit a unified `command_name` output (and a `label_name` alias) for the triggering command.
 *
 * This step is used when `label_command` is configured with `remove_label: false`.
 * It resolves the command name for both label-command and slash-command triggers so that
 * downstream jobs can reference a single `needs.activation.outputs.command_name` regardless
 * of which trigger type fired the workflow.
 *
 * Resolution order:
 *   1. Labeled events  — `command_name` = the triggering label name.
 *   2. Other events    — `command_name` = GH_AW_MATCHED_COMMAND env var (slash-command name),
 *                        falling back to an empty string if the var is absent.
 *   3. workflow_dispatch — same fallback logic as (2); normally produces an empty string.
 *
 * Outputs:
 *   label_name   — the triggering label name, or "" for non-labeled events.
 *   command_name — unified command name usable by both label_command and slash_command workflows.
 */
function main() {
  // Optional: pre-computed matched slash-command name from check_command_position.cjs
  const matchedCommand = process.env.GH_AW_MATCHED_COMMAND ?? "";

  const eventName = context.eventName;

  if (eventName === "workflow_dispatch") {
    core.info("Event is workflow_dispatch – no label trigger.");
    core.setOutput("label_name", "");
    core.setOutput("command_name", matchedCommand);
    return;
  }

  // For labeled events (issues, pull_request, discussion) use the label name.
  const labelName = context.payload?.label?.name ?? "";
  if (labelName) {
    core.info(`Trigger label: '${labelName}'`);
    core.setOutput("label_name", labelName);
    core.setOutput("command_name", labelName);
    return;
  }

  // Non-labeled events (e.g. slash-command comments) — fall back to the matched command.
  core.info(`Event '${eventName}' has no label payload – using matched command '${matchedCommand}'.`);
  core.setOutput("label_name", "");
  core.setOutput("command_name", matchedCommand);
}

module.exports = { main };
