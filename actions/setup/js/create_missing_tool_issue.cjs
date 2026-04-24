// @ts-check
const { handlerRegistry } = require("./missing_issue_handler_registry.cjs");

/** @type {import('./types/handler-factory').HandlerFactoryFunction} */
const main = handlerRegistry.get("create_missing_tool_issue");
if (typeof main !== "function") throw new Error("create_missing_tool_issue handler not found in registry");

module.exports = { main };
