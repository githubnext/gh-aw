async function setupAgentOutputMain() {
  // Generate a unique filename using 16 random hex characters
  const crypto = require("crypto");
  const fs = require("fs");
  const randomSuffix = crypto.randomBytes(8).toString("hex");
  const outputFile = `/tmp/aw_output_${randomSuffix}.txt`;
  fs.mkdirSync("/tmp", { recursive: true });
  core.exportVariable("GITHUB_AW_SAFE_OUTPUTS", outputFile);
  core.setOutput("output_file", outputFile);
}

(async () => {
  await setupAgentOutputMain();
})();
