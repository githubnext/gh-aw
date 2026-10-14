// @ts-check

const fs = require("fs");

const RESOURCES = ["root", "runtime", "workspace", "runner_temp"];
const SPACE_BANDS = ["exhausted", "critical", "low", "healthy", "unavailable"];
const INODE_BANDS = ["exhausted", "low", "healthy", "unavailable"];
const OPERATIONS = ["setup", "interval", "shutdown"];
const ERROR_CODES = new Set(["EACCES", "EPERM", "ENOENT", "ELOOP", "ENOSPC", "EIO", "EMFILE", "ENFILE", "EROFS", "EEXIST", "EINVAL", "ENOTDIR", "ENOSYS", "ENOTSUP"]);

/** @param {unknown} error @returns {string} */
function diskErrorCode(error) {
  return error && typeof error === "object" && "code" in error && ERROR_CODES.has(String(error.code)) ? String(error.code) : "UNAVAILABLE";
}

/** @param {string} operation @param {unknown} [error] */
function diskWarning(operation, error) {
  // Keep annotations on stdout so CLI callers can discard unsanitized Node
  // bootstrap diagnostics on stderr without losing monitor warnings.
  process.stdout.write(`::warning::disk_usage_logger: ${operation} (${diskErrorCode(error)})\n`);
}

/** @param {string} content @returns {string} */
function sanitizeDiskDiagnostics(content) {
  const lines = content.split(/\r?\n/).filter(line => line.trim());
  let discarded = lines.length > 4096;
  const output = lines
    .slice(0, 4096)
    .map(line => {
      const match = /^::warning::disk_usage_logger: [a-z0-9 _-]+ \(([A-Z]+)\)$/.exec(line);
      const code = match && (ERROR_CODES.has(match[1]) || match[1] === "UNAVAILABLE") ? match[1] : "UNAVAILABLE";
      if (!match) discarded = true;
      return `::warning::disk_usage_logger: monitor diagnostic (${code})\n`;
    })
    .join("");
  if (discarded) diskWarning("discarded unsafe monitor diagnostics");
  return output;
}

/** @param {number} available */
function spaceBand(available) {
  return available === 0 ? "exhausted" : available < 1024 ** 3 ? "critical" : available < 5 * 1024 ** 3 ? "low" : "healthy";
}

/** @param {number} total @param {number} free */
function inodeBand(total, free) {
  return !total ? "unavailable" : !free ? "exhausted" : free / total < 0.01 ? "low" : "healthy";
}

/** Strip legacy precise and opaque fields, including untrusted labels.
 * @param {any} record
 * @returns {Record<string, string> | undefined}
 */
function normalizeDiskUsage(record) {
  if (!record || typeof record !== "object" || !RESOURCES.includes(record.resource) || !OPERATIONS.includes(record.operation)) return undefined;
  const data = {
    source: "statfs",
    operation: record.operation,
    resource: record.resource,
    availableSpaceBand: SPACE_BANDS.includes(record.availableSpaceBand) ? record.availableSpaceBand : "unavailable",
    inodeHeadroom: INODE_BANDS.includes(record.inodeHeadroom) ? record.inodeHeadroom : "unavailable",
  };
  return data;
}

/** @param {unknown} value @returns {string | undefined} */
function diskTimestamp(value) {
  return typeof value === "string" && /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{3})?Z$/.test(value) && Number.isFinite(Date.parse(value)) ? value : undefined;
}

/** Open a directory without following its final component. Child operations must
 * use the descriptor anchor, not the original pathname. Linux /proc is required;
 * other platforms fail closed rather than falling back to racy path checks.
 * @param {string} directory
 * @returns {{fd: number, anchor: string}}
 */
function openDiskDirectory(directory) {
  if (fs.constants.O_NOFOLLOW === undefined || fs.constants.O_DIRECTORY === undefined) throw new Error("safe disk opens unavailable");
  const fd = fs.openSync(directory, fs.constants.O_RDONLY | fs.constants.O_DIRECTORY | fs.constants.O_NOFOLLOW);
  try {
    if (!fs.fstatSync(fd).isDirectory()) throw new Error("not a directory");
    const anchor = `/proc/self/fd/${fd}`;
    if (!fs.statSync(anchor).isDirectory()) throw new Error("directory anchors unavailable");
    return { fd, anchor };
  } catch (error) {
    closeDiskFile(fd);
    throw error;
  }
}

/** @param {string} file @param {number} flags @returns {number} */
function openDiskFile(file, flags) {
  const fd = fs.openSync(file, flags | fs.constants.O_NOFOLLOW | fs.constants.O_NONBLOCK, 0o600);
  try {
    const stat = fs.fstatSync(fd);
    if (!stat.isFile() || stat.nlink !== 1) throw new Error("unsafe disk telemetry file");
    return fd;
  } catch (error) {
    closeDiskFile(fd);
    throw error;
  }
}

/** @param {number | undefined} fd */
function closeDiskFile(fd) {
  if (fd === undefined) return;
  try {
    fs.closeSync(fd);
  } catch (error) {
    diskWarning("failed to close monitor file", error);
  }
}

/** @param {string} file */
function removeDiskControl(file) {
  try {
    fs.unlinkSync(file);
  } catch (error) {
    if (diskErrorCode(error) !== "ENOENT") diskWarning("failed to remove monitor control", error);
  }
}

module.exports = { RESOURCES, spaceBand, inodeBand, normalizeDiskUsage, diskTimestamp, diskWarning, diskErrorCode, sanitizeDiskDiagnostics, openDiskDirectory, openDiskFile, closeDiskFile, removeDiskControl };
