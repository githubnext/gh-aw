// @ts-check
"use strict";

require("./shim.cjs");

const { runGatewayProfile } = require("./convert_gateway_config_factory.cjs");
const { gatewayConversionProfiles, toCodexTomlSection } = require("./convert_gateway_config_profiles.cjs");

function main() {
  return runGatewayProfile(gatewayConversionProfiles.codex);
}

if (require.main === module) {
  main();
}

module.exports = { toCodexTomlSection, main };
