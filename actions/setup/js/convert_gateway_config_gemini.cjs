// @ts-check
"use strict";

require("./shim.cjs");

const { rewriteUrl } = require("./convert_gateway_config_shared.cjs");
const { runGatewayProfile } = require("./convert_gateway_config_factory.cjs");
const { gatewayConversionProfiles, transformGeminiEntry } = require("./convert_gateway_config_profiles.cjs");

function main() {
  return runGatewayProfile(gatewayConversionProfiles.gemini);
}

if (require.main === module) {
  main();
}

module.exports = { rewriteUrl, transformGeminiEntry, main };
