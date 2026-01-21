// Quick test to verify update_issue behavior with invalid target
const { main } = require("./update_issue.cjs");

// Mock global objects
global.core = {
  info: (...args) => console.log("INFO:", ...args),
  warning: (...args) => console.log("WARNING:", ...args),
  error: (...args) => console.log("ERROR:", ...args),
  debug: () => {},
  setOutput: () => {},
};

global.github = {
  rest: {
    issues: {
      get: async () => ({ data: { body: "old body", number: 230 } }),
      update: async (params) => {
        console.log("UPDATE CALLED:", params);
        return { data: { number: params.issue_number, html_url: "https://github.com/test/test/issues/230", title: "Test" } };
      },
    },
  },
};

global.context = {
  eventName: "issues",
  repo: { owner: "test", repo: "test" },
  serverUrl: "https://github.com",
  runId: 12345,
  payload: { issue: { number: 100 } },
};

async function test() {
  console.log("\n=== Testing update_issue with target='event' ===\n");
  
  const handler = await main({ target: "event", allow_body: true, max: 10 });
  const message = {
    type: "update_issue",
    body: "New body content",
  };
  
  const result = await handler(message, {});
  
  console.log("\n=== Result ===");
  console.log(JSON.stringify(result, null, 2));
}

test().catch(console.error);
