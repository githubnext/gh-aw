// @ts-check

const fs = require("fs");
const path = require("path");
const os = require("os");
const crypto = require("crypto");

class TransportIntegrityError extends Error {
  constructor(filePath, cause = undefined) {
    super(`SECURITY: Safe-output transport changed after capture: ${filePath}`, { cause });
    this.name = "TransportIntegrityError";
  }
}

const digest = bytes => crypto.createHash("sha256").update(bytes).digest("hex");

/**
 * Capture validation inputs once. Git receives private, read-only snapshots, never
 * the agent-writable originals, so checking a hash cannot race with git's open().
 */
function createTransportIntegrity() {
  /** @type {Map<string, { bytes: Buffer, hash: string, dev: number, ino: number, snapshot?: string }>} */
  const inputs = new Map();
  let directory = "";
  /** @type {TransportIntegrityError | null} */
  let failure = null;

  function fail(filePath, cause = undefined) {
    failure ||= new TransportIntegrityError(filePath, cause);
    throw failure;
  }

  function captureOptional(filePath) {
    if (!filePath || inputs.has(filePath)) return;
    try {
      const fd = fs.openSync(filePath, fs.constants.O_RDONLY | fs.constants.O_NOFOLLOW);
      try {
        const stat = fs.fstatSync(fd);
        if (!stat.isFile()) fail(filePath);
        const bytes = fs.readFileSync(fd);
        inputs.set(filePath, { bytes, hash: digest(bytes), dev: stat.dev, ino: stat.ino });
      } finally {
        fs.closeSync(fd);
      }
    } catch {
      // A resolved canonical path was present. Losing access to it must not be
      // interpreted as an absent/empty patch (including allow-empty handlers).
      fail(filePath);
    }
  }

  function assertUnchanged() {
    if (failure) throw failure;
    for (const [filePath, input] of inputs) {
      try {
        const fd = fs.openSync(filePath, fs.constants.O_RDONLY | fs.constants.O_NOFOLLOW);
        try {
          const stat = fs.fstatSync(fd);
          if (!stat.isFile() || stat.dev !== input.dev || stat.ino !== input.ino || digest(fs.readFileSync(fd)) !== input.hash) fail(filePath);
        } finally {
          fs.closeSync(fd);
        }
        if (input.snapshot && digest(fs.readFileSync(input.snapshot)) !== digest(input.bytes)) fail(filePath);
      } catch {
        fail(filePath);
      }
    }
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
        input.snapshot = path.join(directory, path.basename(filePath));
        fs.writeFileSync(input.snapshot, input.bytes, { mode: 0o400, flag: "wx" });
      }
    } catch (error) {
      fail(filePath, error);
    }
    return input.snapshot;
  }

  // Only trusted transformations of previously captured validation bytes belong
  // here. The original baseline is deliberately never updated from disk.
  function transform(filePath, transformBytes) {
    const bytes = transformBytes(read(filePath));
    const input = inputs.get(filePath);
    if (!input) throw new Error(`No captured transport file: ${filePath}`);
    if (input.snapshot) {
      try {
        fs.unlinkSync(input.snapshot);
      } catch (error) {
        fail(filePath, error);
      }
      input.snapshot = undefined;
    }
    input.bytes = Buffer.from(bytes);
  }

  function guardExec(execApi) {
    const guarded = { ...execApi };
    for (const method of ["exec", "getExecOutput"]) {
      guarded[method] = async (command, args = [], ...options) => {
        assertUnchanged();
        const stableArgs = command === "git" ? args.map(arg => (inputs.has(arg) ? snapshot(arg) : arg)) : args;
        try {
          return await execApi[method](command, stableArgs, ...options);
        } finally {
          // Also detect changes made while git was running, before any push/API
          // side effects, including when the command itself failed.
          assertUnchanged();
        }
      };
    }
    return guarded;
  }

  function dispose() {
    try {
      if (directory) fs.rmSync(directory, { recursive: true, force: true });
    } catch (error) {
      throw new Error("Failed to remove safe-output transport snapshots", { cause: error });
    }
  }

  return { captureOptional, read, transform, assertUnchanged, guardExec, dispose };
}

function withTransportIntegrity(handler) {
  return async (message, resolvedTemporaryIds) => {
    const transport = createTransportIntegrity();
    try {
      const result = await handler(message, resolvedTemporaryIds, transport);
      transport.assertUnchanged();
      return result;
    } catch (error) {
      if (!(error instanceof TransportIntegrityError)) throw error;
      return { success: false, error: error.message };
    } finally {
      transport.dispose();
    }
  };
}

module.exports = { createTransportIntegrity, withTransportIntegrity, TransportIntegrityError };
