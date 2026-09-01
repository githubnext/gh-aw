// @ts-check
const { createAzureDevOpsWorkItemHandler } = require("./azure_devops_work_items.cjs");
async function main(config = {}) {
  return createAzureDevOpsWorkItemHandler("update_work_item", config);
}
module.exports = { main };
