// @ts-check
const { handlerRegistry } = require("./missing_issue_handler_registry.cjs");

const main = /** @type {import('./types/handler-factory').HandlerFactoryFunction} */ handlerRegistry.get("create_report_incomplete_issue");
if (typeof main !== "function") throw new Error("create_report_incomplete_issue handler not found in registry");

module.exports = { main };
