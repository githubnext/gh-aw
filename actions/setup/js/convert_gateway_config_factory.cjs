// @ts-check
"use strict";

const { runGatewayConversion } = require("./convert_gateway_config_shared.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {{
 *   gatewayOutput: string;
 *   domain: string;
 *   port: string;
 *   urlPrefix: string;
 *   cliServers: Set<string>;
 *   servers: Record<string, Record<string, unknown>>;
 *   extraEnv: Record<string, string>;
 * }} GatewayContext
 * @typedef {{
 *   format: string;
 *   engine: string;
 *   outputPath?: string | ((context: GatewayContext) => string);
 *   preRunOutputPath?: () => string;
 *   contextOptions?: { extraRequiredEnv?: string[] };
 *   getTargetDomain?: (context: GatewayContext) => string;
 *   getUrlPrefix?: (context: GatewayContext) => string;
 *   getUrlPrefixLog?: (context: GatewayContext) => string | undefined;
 *   transformEntry?: (entry: Record<string, unknown>, urlPrefix: string, context: GatewayContext, name: string) => Record<string, unknown>;
 *   transformServer?: (name: string, entry: Record<string, unknown>, urlPrefix: string, context: GatewayContext) => Record<string, unknown>;
 *   serialize: (servers: Record<string, Record<string, unknown>>, context: GatewayContext, urlPrefix: string) => string;
 *   setFailedOnError?: boolean;
 * }} GatewayConversionProfile
 */

/**
 * @param {GatewayConversionProfile} profile
 */
function buildGatewayConversionOptions(profile) {
  const outputPath = profile.preRunOutputPath ? profile.preRunOutputPath() : profile.outputPath;
  if (!outputPath) {
    throw new Error(`Gateway conversion profile ${profile.engine} is missing an output path`);
  }
  const transformServer =
    profile.transformServer ||
    ((name, entry, urlPrefix, context) => {
      if (!profile.transformEntry) {
        return entry;
      }
      return profile.transformEntry(entry, urlPrefix, context, name);
    });
  const getUrlPrefix = profile.getUrlPrefix;
  const getUrlPrefixLog = profile.getUrlPrefixLog;

  return {
    format: profile.format,
    engine: profile.engine,
    outputPath,
    contextOptions: profile.contextOptions,
    getTargetDomain: profile.getTargetDomain,
    getUrlPrefix:
      getUrlPrefixLog || !getUrlPrefix
        ? context => {
            const message = getUrlPrefixLog?.(context);
            if (message) {
              core.info(message);
            }
            return getUrlPrefix ? getUrlPrefix(context) : context.urlPrefix;
          }
        : getUrlPrefix,
    transformServer,
    serialize: profile.serialize,
  };
}

/**
 * @param {GatewayConversionProfile} profile
 */
function runGatewayProfile(profile) {
  let options;
  try {
    options = buildGatewayConversionOptions(profile);
  } catch (err) {
    if (profile.setFailedOnError) {
      core.setFailed(`ERROR: ${getErrorMessage(err)}`);
      return undefined;
    }
    throw err;
  }
  return runGatewayConversion(options);
}

module.exports = { buildGatewayConversionOptions, runGatewayProfile };
