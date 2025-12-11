// @ts-check
/// <reference types="@actions/github-script" />

const { processCloseEntityItems, createEntityCallbacks, PULL_REQUEST_CONFIG } = require("./close_entity_helpers.cjs");

async function main() {
  const callbacks = createEntityCallbacks(PULL_REQUEST_CONFIG);
  return processCloseEntityItems(PULL_REQUEST_CONFIG, callbacks);
}

await main();
