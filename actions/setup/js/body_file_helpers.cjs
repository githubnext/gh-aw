// @ts-check

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

const { getErrorMessage } = require("./error_helpers.cjs");
const { lstatGuard } = require("./symlink_guard.cjs");

const SAFE_BODY_FILE_DIRNAME = "gh-aw-safe";
const MAX_BODY_FILE_BYTES = 65536;

function getSafeBodyFileRoot() {
  const runnerTemp = process.env.RUNNER_TEMP || "/tmp";
  return path.resolve(runnerTemp, SAFE_BODY_FILE_DIRNAME);
}

function normalizeAuditPath(filePath) {
  const allowlistedRoot = getSafeBodyFileRoot();
  const relative = path.relative(allowlistedRoot, filePath);
  return relative && relative !== "." ? `${SAFE_BODY_FILE_DIRNAME}/${relative.split(path.sep).join("/")}` : SAFE_BODY_FILE_DIRNAME;
}

function ensurePathHasNoSymlinks(candidatePath, allowlistedRoot) {
  const relative = path.relative(allowlistedRoot, candidatePath);
  const segments = relative.split(path.sep).filter(Boolean);
  let currentPath = allowlistedRoot;

  if (lstatGuard(currentPath) === null) {
    throw new Error(`Rejected body_file '${normalizeAuditPath(candidatePath)}': allowlisted root is a symbolic link`);
  }

  for (const segment of segments) {
    currentPath = path.join(currentPath, segment);
    if (lstatGuard(currentPath) === null) {
      throw new Error(`Rejected body_file '${normalizeAuditPath(candidatePath)}': symbolic links are not allowed`);
    }
  }
}

function resolveBodyFilePath(bodyFile) {
  const rawValue = typeof bodyFile === "string" ? bodyFile.trim() : "";
  if (!rawValue) {
    throw new Error("body_file must be a non-empty string");
  }

  const allowlistedRoot = getSafeBodyFileRoot();
  const candidatePath = path.isAbsolute(rawValue) ? path.resolve(rawValue) : path.resolve(process.env.RUNNER_TEMP || "/tmp", rawValue);
  const relative = path.relative(allowlistedRoot, candidatePath);
  if (relative.startsWith("..") || path.isAbsolute(relative)) {
    throw new Error(`Rejected body_file '${rawValue}': path must stay under ${SAFE_BODY_FILE_DIRNAME}/`);
  }
  if (!fs.existsSync(candidatePath)) {
    throw new Error(`Rejected body_file '${rawValue}': file does not exist`);
  }

  ensurePathHasNoSymlinks(candidatePath, allowlistedRoot);

  let stat;
  try {
    stat = fs.statSync(candidatePath);
  } catch (error) {
    throw new Error(`Rejected body_file '${rawValue}': ${getErrorMessage(error)}`);
  }
  if (!stat.isFile()) {
    throw new Error(`Rejected body_file '${rawValue}': path is not a regular file`);
  }
  if (stat.size > MAX_BODY_FILE_BYTES) {
    throw new Error(`Rejected body_file '${rawValue}': file is too large (${stat.size} bytes > ${MAX_BODY_FILE_BYTES} bytes)`);
  }

  return candidatePath;
}

function readBodyFileSnapshot(bodyFile, expectedSha256) {
  const resolvedPath = resolveBodyFilePath(bodyFile);
  const openFlags = fs.constants.O_RDONLY | (fs.constants.O_NOFOLLOW || 0);
  let fd;
  try {
    fd = fs.openSync(resolvedPath, openFlags);
  } catch (error) {
    throw new Error(`Rejected body_file '${bodyFile}': ${getErrorMessage(error)}`);
  }
  try {
    const stat = fs.fstatSync(fd);
    if (!stat.isFile()) {
      throw new Error(`Rejected body_file '${bodyFile}': path is not a regular file`);
    }
    if (stat.size > MAX_BODY_FILE_BYTES) {
      throw new Error(`Rejected body_file '${bodyFile}': file is too large (${stat.size} bytes > ${MAX_BODY_FILE_BYTES} bytes)`);
    }
    let fileBytes;
    try {
      fileBytes = fs.readFileSync(fd);
    } catch (error) {
      throw new Error(`Rejected body_file '${bodyFile}': ${getErrorMessage(error)}`);
    }
    const digest = crypto.createHash("sha256").update(fileBytes).digest("hex");
    if (digest !== expectedSha256) {
      throw new Error(`Rejected body_file '${bodyFile}': body_sha256 mismatch`);
    }
    if (fileBytes.includes(0)) {
      throw new Error(`Rejected body_file '${bodyFile}': file must be UTF-8 text`);
    }

    let content;
    try {
      content = new TextDecoder("utf-8", { fatal: true }).decode(fileBytes);
    } catch {
      throw new Error(`Rejected body_file '${bodyFile}': file must be UTF-8 text`);
    }

    return {
      content,
      metadata: {
        path: normalizeAuditPath(resolvedPath),
        sha256: digest,
        bytes: fileBytes.length,
      },
    };
  } finally {
    fs.closeSync(fd);
  }
}

module.exports = {
  SAFE_BODY_FILE_DIRNAME,
  MAX_BODY_FILE_BYTES,
  getSafeBodyFileRoot,
  readBodyFileSnapshot,
  resolveBodyFilePath,
};
