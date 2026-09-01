// @ts-check
const { createAzureDevOpsWorkItemHandler } = require("./azure_devops_work_items.cjs");
async function main(config = {}) {
  return createAzureDevOpsWorkItemHandler("link_work_items", config);
}
module.exports = { main };
