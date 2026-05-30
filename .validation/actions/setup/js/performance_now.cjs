// @ts-check

const { performance } = require("perf_hooks");

/**
 * Returns the current time as a high-resolution epoch timestamp in milliseconds.
 *
 * Uses `performance.timeOrigin + performance.now()` rather than `Date.now()` to
 * take advantage of the sub-millisecond precision available in Node.js 24+.
 *
 * @returns {number} Milliseconds since the Unix epoch (floating-point).
 */
function nowMs() {
  return performance.timeOrigin + performance.now();
}

module.exports = { nowMs };
