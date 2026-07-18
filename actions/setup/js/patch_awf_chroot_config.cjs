// @ts-check

const fs = require("fs");
const os = require("os");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Patch the AWF config file with chroot settings for ARC/DinD runners.
 *
 * @param {Object} [options]
 * @param {string} [options.runnerTemp]
 * @param {string} [options.binariesSourcePath]
 * @param {string} [options.identityHome]
 * @returns {string} The patched JSON content
 */
function patchAWFChrootConfig(options = {}) {
  const runnerTemp = options.runnerTemp || process.env.RUNNER_TEMP;
  if (!runnerTemp) {
    throw new Error("RUNNER_TEMP is required");
  }

  const binariesSourcePath = options.binariesSourcePath || process.env.GH_AW_CHROOT_BINARIES_SOURCE_PATH || "/tmp/gh-aw";
  const identityHome = options.identityHome || process.env.GH_AW_CHROOT_IDENTITY_HOME || "/tmp/gh-aw/home";
  const configPath = path.join(runnerTemp, "gh-aw", "awf-config.json");
  const artifactConfigPath = path.join(binariesSourcePath, "awf-config.json");
  let config;
  try {
    config = JSON.parse(fs.readFileSync(configPath, "utf8"));
  } catch (err) {
    throw new Error("Failed to parse AWF config file " + configPath + ": " + getErrorMessage(err), { cause: err });
  }
  const userInfo = os.userInfo();

  config.chroot = {
    binariesSourcePath,
    identity: {
      user: userInfo.username,
      uid: userInfo.uid,
      gid: userInfo.gid,
      home: identityHome,
    },
  };

  const output = `${JSON.stringify(config)}\n`;
  try {
    fs.writeFileSync(configPath, output);
    fs.writeFileSync(artifactConfigPath, output);
  } catch (err) {
    throw new Error(`Failed to write chroot config: ${String(err)}`, { cause: err });
  }
  return output;
}

if (require.main === module) {
  try {
    patchAWFChrootConfig();
  } catch (error) {
    const message = getErrorMessage(error);
    throw new Error(`chroot config patch failed: ${message}`);
  }
}

module.exports = { patchAWFChrootConfig };
