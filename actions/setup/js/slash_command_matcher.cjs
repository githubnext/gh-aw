// @ts-check

/**
 * Extracts the slash command name from the start of the given body text.
 * Returns an empty string if the text does not begin with a valid slash command.
 * A valid slash command starts with '/' followed by a name of one or more characters
 * from [a-zA-Z0-9], [-], [_], and [.].
 * @param {string} text
 * @returns {string}
 */
function parseSlashCommand(text) {
  const match = /^\/([a-zA-Z0-9][a-zA-Z0-9._-]*)(?=$|\s)/.exec(String(text).trim());
  return match ? match[1] : "";
}

const CATCH_ALL_COMMAND = "*";

/**
 * @param {string} commandName
 * @returns {boolean}
 */
function isWildcardCommandName(commandName) {
  return typeof commandName === "string" && commandName.endsWith("*");
}

/**
 * @param {string} commandName
 * @returns {boolean}
 */
function isCatchAllCommandName(commandName) {
  return commandName === CATCH_ALL_COMMAND;
}

/**
 * @param {string} configuredCommand
 * @returns {string}
 */
function wildcardCommandPrefix(configuredCommand) {
  return isWildcardCommandName(configuredCommand) ? configuredCommand.slice(0, -1) : "";
}

/**
 * @param {string} configuredCommand
 * @param {string} actualCommand
 * @returns {boolean}
 */
function matchesCommandName(configuredCommand, actualCommand) {
  if (typeof configuredCommand !== "string" || typeof actualCommand !== "string") {
    return false;
  }

  if (isCatchAllCommandName(configuredCommand)) {
    return actualCommand !== "";
  }

  if (isWildcardCommandName(configuredCommand)) {
    const prefix = wildcardCommandPrefix(configuredCommand);
    return prefix !== "" && actualCommand.startsWith(prefix);
  }

  return configuredCommand === actualCommand;
}

/**
 * @param {string} text
 * @param {string[]} configuredCommands
 * @returns {string}
 */
function resolveMatchedCommand(text, configuredCommands) {
  const actualCommand = parseSlashCommand(text);
  if (!actualCommand) {
    return "";
  }

  for (const configuredCommand of configuredCommands) {
    if (matchesCommandName(configuredCommand, actualCommand)) {
      return actualCommand;
    }
  }

  return "";
}

module.exports = {
  CATCH_ALL_COMMAND,
  isCatchAllCommandName,
  isWildcardCommandName,
  matchesCommandName,
  parseSlashCommand,
  resolveMatchedCommand,
  wildcardCommandPrefix,
};
