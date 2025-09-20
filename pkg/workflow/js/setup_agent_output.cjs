const fs = require("fs");
function main() {
  const outputFile = `/tmp/aw_output.txt`;
  fs.mkdirSync("/tmp", { recursive: true });
  core.exportVariable("GITHUB_AW_SAFE_OUTPUTS", outputFile);
  core.setOutput("output_file", outputFile);
}
main();
