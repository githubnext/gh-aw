// @ts-check

const { StringDecoder } = require("node:string_decoder");

/**
 * Keep automatic exec streams silent while forwarding Git output to redacted
 * Actions logs. Preserve caller listeners and decode split UTF-8 characters.
 * @param {import("@actions/exec").ExecOptions} [options]
 * @param {boolean} [logStdout] Keep credential-query results silent when false.
 * @returns {import("@actions/exec").ExecOptions}
 */
function gitLoggingOptions(options = {}, logStdout = true) {
  const stdoutDecoder = new StringDecoder("utf8");
  const stderrDecoder = new StringDecoder("utf8");
  const listeners = options.listeners || {};
  return {
    ...options,
    silent: true,
    listeners: {
      ...listeners,
      stdout: data => {
        const text = stdoutDecoder.write(data);
        if (text && logStdout) core.info(text);
        listeners.stdout?.(data);
      },
      stderr: data => {
        const text = stderrDecoder.write(data);
        if (text) core.info(text);
        listeners.stderr?.(data);
      },
    },
  };
}

module.exports = { gitLoggingOptions };
