// @ts-check
"use strict";

require("./shim.cjs");

const { runGatewayProfile } = require("./convert_gateway_config_factory.cjs");
const { gatewayConversionProfiles, transformClaudeEntry } = require("./convert_gateway_config_profiles.cjs");

function main() {
  return runGatewayProfile(gatewayConversionProfiles.claude);
}

if (require.main === module) {
  main();
}

module.exports = { transformClaudeEntry, main };
