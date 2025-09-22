const fs = require("fs");
const crypto = require("crypto");

function main() {
  // Generate a unique filename using 16 random hex characters
  const randomSuffix = crypto.randomBytes(8).toString("hex");
  const outputFile = `/tmp/aw_output_${randomSuffix}.txt`;
  fs.mkdirSync("/tmp", { recursive: true });
  core.exportVariable("GITHUB_AW_SAFE_OUTPUTS", outputFile);
  core.setOutput("output_file", outputFile);
}
main();
