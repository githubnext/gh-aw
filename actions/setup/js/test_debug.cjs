const { processMessages } = require("./safe_output_handler_manager.cjs");

// Mock core
global.core = {
  info: (msg) => console.log("[INFO]", msg),
  debug: (msg) => console.log("[DEBUG]", msg),
  warning: (msg) => console.log("[WARN]", msg),
  error: (msg) => console.log("[ERROR]", msg),
  setOutput: () => {},
  setFailed: () => {},
};

const messages = [
  { type: "create_issue", body: "Related to #aw_tempid111111", title: "First Issue" },
  { type: "create_discussion", body: "See #aw_tempid111111 for details", title: "Discussion" },
  { type: "create_issue", temporary_id: "aw_tempid111111", body: "The referenced issue", title: "Referenced Issue" },
];

let callCount = 0;
const mockCreateIssueHandler = async () => {
  callCount++;
  if (callCount === 1) return { repo: "owner/repo", number: 100 };
  if (callCount === 2) return { repo: "owner/repo", number: 102, temporaryId: "aw_tempid111111" };
};

const mockCreateDiscussionHandler = async () => ({ repo: "owner/repo", number: 101 });

const handlers = new Map([
  ["create_issue", mockCreateIssueHandler],
  ["create_discussion", mockCreateDiscussionHandler],
]);

processMessages(handlers, messages).then(result => {
  console.log("\n=== RESULT ===");
  console.log("Success:", result.success);
  console.log("Results count:", result.results.length);
  console.log("Temp ID map:", result.temporaryIdMap);
  console.log("Pending updates:", result.pendingUpdates.length);
  result.pendingUpdates.forEach((u, i) => {
    console.log(`  Update ${i+1}:`, u.type, "for issue", u.issue_number || u.discussion_number);
  });
});
