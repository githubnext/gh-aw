// @ts-check
"use strict";

require("./shim.cjs");

const { rewriteUrl } = require("./convert_gateway_config_shared.cjs");
const { runGatewayProfile } = require("./convert_gateway_config_factory.cjs");
const { gatewayConversionProfiles, resolveCopilotConfigOutputPath, transformCopilotEntry } = require("./convert_gateway_config_profiles.cjs");

function main() {
  return runGatewayProfile(gatewayConversionProfiles.copilot);
}

if (require.main === module) {
  main();
}

module.exports = { rewriteUrl, transformCopilotEntry, resolveCopilotConfigOutputPath, main };
