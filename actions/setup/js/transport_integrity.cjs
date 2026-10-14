// @ts-check

const fs = require("fs");
const path = require("path");
const os = require("os");
const crypto = require("crypto");
const READ_FLAGS = fs.constants.O_RDONLY | fs.constants.O_NOFOLLOW | fs.constants.O_NONBLOCK;

class TransportIntegrityError extends Error {
  /** @param {unknown} [cause] */
  constructor(filePath, cause = undefined, operation = "verify", reason = "artifact changed") {
    const systemCode = cause && typeof cause === "object" && "code" in cause && typeof cause.code === "string" && /^E[A-Z0-9]+$/.test(cause.code) ? ` (${cause.code})` : "";
    super(`SECURITY: Safe-output transport changed after capture: ${filePath}; ${operation}: ${reason}${systemCode}`, { cause });
    this.name = "TransportIntegrityError";
    this.code = "ERR_TRANSPORT_INTEGRITY";
    this.operation = operation;
    this.reason = reason;
  }
}

const digest = bytes => crypto.createHash("sha256").update(bytes).digest("hex");

/**
 * Capture validation inputs once. Git receives private, read-only snapshots, never
 * the agent-writable originals, so checking a hash cannot race with git's open().
 */
function createTransportIntegrity() {
  /** @typedef {{hash: string, dev: number, ino: number, size: number}} FileIdentity */
  /** @type {Map<string, FileIdentity & {bytes: Buffer, snapshot?: FileIdentity & {path: string}}>} */
  const inputs = new Map();
  /** @type {Map<string, () => Promise<unknown>>} */
  const cleanups = new Map();
  let directory = "";
  /** @type {TransportIntegrityError | null} */
  let failure = null;

  /** @returns {never} */
  function fail(filePath, cause = undefined, operation = "verify", reason = "unable to read artifact") {
    failure ||= new TransportIntegrityError(filePath, cause, operation, reason);
    throw failure;
  }

  function captureOptional(filePath) {
    if (!filePath || inputs.has(filePath)) return;
    try {
      const fd = fs.openSync(filePath, READ_FLAGS);
      try {
        const stat = fs.fstatSync(fd);
        if (!stat.isFile()) fail(filePath, undefined, "capture", "not a regular file");
        const bytes = fs.readFileSync(fd);
        inputs.set(filePath, { bytes, hash: digest(bytes), dev: stat.dev, ino: stat.ino, size: stat.size });
      } finally {
        fs.closeSync(fd);
      }
    } catch (error) {
      // A resolved canonical path was present. Losing access to it must not be
      // interpreted as an absent/empty patch (including allow-empty handlers).
      fail(filePath, error, "capture");
    }
  }

  /** @param {FileIdentity} identity */
  function verifyFile(filePath, identity, contents) {
    try {
      const fd = fs.openSync(filePath, READ_FLAGS);
      try {
        const stat = fs.fstatSync(fd);
        if (!stat.isFile()) fail(filePath, undefined, "verify", "not a regular file");
        if (stat.dev !== identity.dev || stat.ino !== identity.ino) fail(filePath, undefined, "verify", "file identity changed");
        if (stat.size !== identity.size || (contents && digest(fs.readFileSync(fd)) !== identity.hash)) fail(filePath, undefined, "verify", "content changed");
      } finally {
        fs.closeSync(fd);
      }
    } catch (error) {
      fail(filePath, error);
    }
  }

  function checkUnchanged(contents) {
    if (failure) throw failure;
    for (const [filePath, input] of inputs) {
      verifyFile(filePath, input, contents);
      if (input.snapshot) verifyFile(input.snapshot.path, input.snapshot, contents);
    }
  }

  function assertUnchanged() {
    checkUnchanged(true);
  }

  function read(filePath) {
    assertUnchanged();
    const input = inputs.get(filePath);
    if (!input?.bytes) throw new Error(`No captured transport file: ${filePath}`);
    return Buffer.from(input.bytes);
  }

  function snapshot(filePath) {
    const input = inputs.get(filePath);
    if (!input?.bytes) return filePath;
    try {
      if (!input.snapshot) {
        if (!directory) {
          // Keep transport files outside the checkout: git add -A recovery must
          // never stage the private validation inputs into the resulting commit.
          directory = fs.mkdtempSync(path.join(os.tmpdir(), ".gh-aw-transport-"));
          fs.chmodSync(directory, 0o700);
        }
        const snapshotPath = path.join(directory, path.basename(filePath));
        fs.writeFileSync(snapshotPath, input.bytes, { mode: 0o400, flag: "wx" });
        const stat = fs.statSync(snapshotPath);
        input.snapshot = { path: snapshotPath, hash: digest(input.bytes), dev: stat.dev, ino: stat.ino, size: stat.size };
      }
    } catch (error) {
      fail(filePath, error, "snapshot", "unable to create snapshot");
    }
    return input.snapshot.path;
  }

  // Only trusted transformations of previously captured validation bytes belong
  // here. The original baseline is deliberately never updated from disk.
  function transform(filePath, transformBytes) {
    const bytes = transformBytes(read(filePath));
    const input = inputs.get(filePath);
    if (!input) throw new Error(`No captured transport file: ${filePath}`);
    if (input.snapshot) {
      try {
        fs.unlinkSync(input.snapshot.path);
      } catch (error) {
        fail(filePath, error, "transform", "unable to remove snapshot");
      }
      input.snapshot = undefined;
    }
    input.bytes = Buffer.from(bytes);
  }

  function guardExec(execApi) {
    const guarded = { ...execApi };
    for (const method of ["exec", "getExecOutput"]) {
      guarded[method] = async (command, args = [], ...options) => {
        const cleanupKey = cleanupArgs => JSON.stringify([options[0]?.cwd || process.cwd(), cleanupArgs]);
        const key = cleanupKey(args);
        // Only cleanup registered by this guard may bypass a sticky failure.
        if (command === "git" && cleanups.has(key)) {
          const result = await execApi[method](command, args, ...options);
          if ((method === "exec" ? result : result.exitCode) === 0) cleanups.delete(key);
          return result;
        }
        const usesTransport = command === "git" && args.some(arg => inputs.has(arg));
        const verifyContents = usesTransport || (command === "git" && args[0] === "push");
        // Full byte checks remain mandatory at reads, application, publication,
        // and handler completion. Unrelated local commands only check identity.
        checkUnchanged(verifyContents);
        const stableArgs = command === "git" ? args.map(arg => (inputs.has(arg) ? snapshot(arg) : arg)) : args;
        const cleanupOptions = [{ ...options[0], cwd: options[0]?.cwd || process.cwd() }, ...options.slice(1)];
        let amCleanupKey;
        if (usesTransport && args[0] === "am" && args[1] === "--3way") {
          amCleanupKey = cleanupKey(["am", "--abort"]);
          cleanups.set(amCleanupKey, () => execApi.exec("git", ["am", "--abort"], ...cleanupOptions));
        }
        if (usesTransport && args[0] === "fetch") {
          for (const refspec of args.slice(2)) {
            const ref = refspec.split(":")[1];
            if (ref?.startsWith("refs/bundles/")) {
              const cleanup = ["update-ref", "-d", ref];
              cleanups.set(cleanupKey(cleanup), () => execApi.exec("git", cleanup, ...cleanupOptions));
            }
          }
        }
        try {
          const result = await execApi[method](command, stableArgs, ...options);
          if ((method === "exec" ? result : result.exitCode) === 0) {
            if (amCleanupKey) cleanups.delete(amCleanupKey);
            if (command === "git" && args[0] === "am" && args[1] === "--continue") cleanups.delete(cleanupKey(["am", "--abort"]));
          }
          return result;
        } finally {
          // Detect identity changes immediately; apply/push commands also
          // recheck bytes, including when the command itself failed.
          checkUnchanged(verifyContents);
        }
      };
    }
    return guarded;
  }

  async function cleanup() {
    const errors = [];
    for (const run of cleanups.values()) {
      try {
        const exitCode = await run();
        if (typeof exitCode === "number" && exitCode !== 0) throw new Error(`Git cleanup exited with code ${exitCode}`);
      } catch (error) {
        errors.push(error);
      }
    }
    cleanups.clear();
    if (errors.length) throw new AggregateError(errors, "Failed to clean up safe-output Git state", { cause: failure });
  }

  function dispose() {
    try {
      if (directory) fs.rmSync(directory, { recursive: true, force: true });
    } catch (error) {
      throw new Error("Failed to remove safe-output transport snapshots", { cause: error });
    }
  }

  return {
    captureOptional,
    read,
    transform,
    assertUnchanged,
    guardExec,
    cleanup,
    dispose,
    get failure() {
      return failure;
    },
  };
}

function withTransportIntegrity(handler) {
  return async (message, resolvedTemporaryIds) => {
    const transport = createTransportIntegrity();
    try {
      const result = await handler(message, resolvedTemporaryIds, transport);
      transport.assertUnchanged();
      return result;
    } catch (error) {
      const integrityError = transport.failure || (error instanceof TransportIntegrityError ? error : null);
      if (!integrityError) throw error;
      if (typeof core !== "undefined") core.error(integrityError.message);
      return { success: false, error: integrityError.message, error_type: integrityError.code };
    } finally {
      try {
        await transport.cleanup();
      } finally {
        transport.dispose();
      }
    }
  };
}

module.exports = { createTransportIntegrity, withTransportIntegrity, TransportIntegrityError };
