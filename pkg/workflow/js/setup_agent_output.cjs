function main() {
  const fs = require("fs");
  const crypto = require("crypto");

  // Generate a random filename for the output file
  const randomId = crypto.randomBytes(8).toString("hex");
  const outputFile = `/tmp/aw_output_${randomId}.txt`;

  // Ensure the /tmp directory exists 
  fs.mkdirSync("/tmp", { recursive: true });

  // We don't create the file, as the name is sufficiently random
  // and some engines (Claude) fails first Write to the file
  // if it exists and has not been read.

  // Set the environment variable for subsequent steps
  core.exportVariable("GITHUB_AW_SAFE_OUTPUTS", outputFile);

  // Also set as step output for reference
  core.setOutput("output_file", outputFile);
}

main();
