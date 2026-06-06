const fs = require("fs");
const path = require("path");
const assert = require("assert");

describe("start_safe_outputs_server.sh", () => {
  it("checks safe_outputs_mcp_arguments.cjs before starting the server", () => {
    const scriptPath = path.join(__dirname, "../sh/start_safe_outputs_server.sh");
    const content = fs.readFileSync(scriptPath, "utf8");
    const requiredDepsBlock = content.match(/REQUIRED_DEPS=\(([\s\S]*?)\)/);

    assert.ok(requiredDepsBlock, "REQUIRED_DEPS block should exist");
    assert.ok(requiredDepsBlock[1].includes('"safe_outputs_mcp_arguments.cjs"'), "REQUIRED_DEPS should include safe_outputs_mcp_arguments.cjs");
  });
});
