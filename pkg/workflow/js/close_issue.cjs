// @ts-check
/// <reference types="@actions/github-script" />

const { processCloseEntityItems, createEntityCallbacks, ISSUE_CONFIG } = require("./close_entity_helpers.cjs");

async function main() {
  const callbacks = createEntityCallbacks(ISSUE_CONFIG);
  return processCloseEntityItems(ISSUE_CONFIG, callbacks);
}

await main();
