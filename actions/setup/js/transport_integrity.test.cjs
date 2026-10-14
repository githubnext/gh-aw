import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";
import crypto from "crypto";
import { spawnSync } from "child_process";
import { createTransportIntegrity, TransportIntegrityError, withTransportIntegrity } from "./transport_integrity.cjs";

describe("safe-output transport integrity", () => {
  let directory;
  let patch;
  let bundle;
  let transport;

  beforeEach(() => {
    directory = fs.mkdtempSync(path.join(process.cwd(), ".transport-integrity-test-"));
    patch = path.join(directory, "aw-test.patch");
    bundle = path.join(directory, "aw-test.bundle");
    fs.writeFileSync(patch, Buffer.from([0x61, 0xff, 0x62, 0x0a]));
    fs.writeFileSync(bundle, "bundle original");
    transport = createTransportIntegrity();
    transport.captureOptional(patch);
    transport.captureOptional(bundle);
  });

  afterEach(() => {
    transport.dispose();
    fs.rmSync(directory, { recursive: true, force: true });
  });

  it("validates and passes exactly the captured raw bytes through private read-only snapshots", async () => {
    const expected = transport.read(patch);
    let snapshotPath;
    const api = {
      exec: vi.fn(async (_command, args) => {
        snapshotPath = args[2];
        expect(snapshotPath).not.toBe(patch);
        expect(fs.readFileSync(snapshotPath)).toEqual(expected);
        expect(fs.statSync(snapshotPath).mode & 0o777).toBe(0o400);
        expect(fs.statSync(path.dirname(snapshotPath)).mode & 0o777).toBe(0o700);
        expect(path.dirname(path.dirname(snapshotPath))).toBe(os.tmpdir());
        return 0;
      }),
    };
    await transport.guardExec(api).exec("git", ["am", "--3way", patch]);
    transport.dispose();
    expect(fs.existsSync(snapshotPath)).toBe(false);
    expect(fs.readFileSync(patch)).toEqual(expected);
  });

  it.each(["same-size mutation", "replacement", "deletion", "symlink"])("rejects %s before git reads a patch", async attack => {
    if (attack === "same-size mutation") fs.writeFileSync(patch, Buffer.from([0x61, 0xfe, 0x62, 0x0a]));
    else {
      fs.renameSync(patch, `${patch}.original`);
      if (attack === "replacement") fs.copyFileSync(`${patch}.original`, patch);
      if (attack === "symlink") fs.symlinkSync(`${patch}.original`, patch);
    }
    const api = { exec: vi.fn() };
    await expect(transport.guardExec(api).exec("git", ["am", "--3way", patch])).rejects.toThrow(TransportIntegrityError);
    expect(api.exec).not.toHaveBeenCalled();
  });

  it("does not reopen the original even if it changes after the final pre-git check", async () => {
    const original = transport.read(patch);
    let applied;
    const api = {
      exec: vi.fn(async (_command, args) => {
        fs.writeFileSync(patch, "evil");
        applied = fs.readFileSync(args[2]);
        return 0;
      }),
    };
    await expect(transport.guardExec(api).exec("git", ["am", "--3way", patch])).rejects.toThrow(TransportIntegrityError);
    expect(applied).toEqual(original);
  });

  it("fails closed when a private snapshot cannot be created", async () => {
    const cause = new Error("disk full");
    const write = vi.spyOn(fs, "writeFileSync").mockImplementationOnce(() => {
      throw cause;
    });
    const api = { exec: vi.fn() };
    try {
      await expect(transport.guardExec(api).exec("git", ["am", "--3way", patch])).rejects.toMatchObject({ name: "TransportIntegrityError", cause });
      await expect(transport.guardExec(api).exec("git", ["am", "--3way", patch])).rejects.toThrow(TransportIntegrityError);
      expect(api.exec).not.toHaveBeenCalled();
    } finally {
      write.mockRestore();
    }
  });

  it.each(["fetch", "list-heads", "verify"])("freezes bundle bytes for %s and rejects tampering before retries", async operation => {
    const api = {
      getExecOutput: vi.fn(async (_command, args) => {
        const stablePath = args.find(arg => arg.endsWith(".bundle"));
        expect(stablePath).not.toBe(bundle);
        expect(fs.readFileSync(stablePath, "utf8")).toBe("bundle original");
        return { exitCode: 1, stdout: "", stderr: "retry needed" };
      }),
    };
    const exec = transport.guardExec(api);
    const args = operation === "fetch" ? ["fetch", bundle, "HEAD:refs/test"] : ["bundle", operation, bundle];
    await exec.getExecOutput("git", args);
    fs.writeFileSync(bundle, "bundle tampered");
    await expect(exec.getExecOutput("git", args)).rejects.toThrow(TransportIntegrityError);
    expect(api.getExecOutput).toHaveBeenCalledTimes(1);
  });

  it("reuses the identical bundle snapshot across head discovery and fetch retries", async () => {
    const paths = [];
    const api = {
      getExecOutput: vi.fn(async (_command, args) => {
        const snapshot = args.find(arg => arg.endsWith(".bundle"));
        paths.push(snapshot);
        expect(fs.readFileSync(snapshot, "utf8")).toBe("bundle original");
        return { exitCode: 0, stdout: "", stderr: "" };
      }),
    };
    const exec = transport.guardExec(api);
    await exec.getExecOutput("git", ["bundle", "verify", bundle]);
    await exec.getExecOutput("git", ["fetch", bundle, "refs/heads/feature:refs/test"]);
    await exec.getExecOutput("git", ["bundle", "list-heads", bundle]);
    await exec.getExecOutput("git", ["fetch", bundle, "HEAD:refs/test"]);
    expect(new Set(paths).size).toBe(1);
  });

  it("fails closed if a resolved artifact disappears before capture", () => {
    fs.unlinkSync(patch);
    const next = createTransportIntegrity();
    expect(() => next.captureOptional(patch)).toThrow(expect.objectContaining({ name: "TransportIntegrityError", operation: "capture", cause: expect.objectContaining({ code: "ENOENT" }) }));
    next.dispose();
  });

  it.skipIf(process.platform === "win32").each(["capture", "verify", "snapshot"])("rejects FIFO input without blocking during %s", operation => {
    const filename = path.join(directory, "fifo.patch");
    const result = spawnSync(
      process.execPath,
      [
        "-e",
        `
          const fs = require("fs");
          const { execFileSync } = require("child_process");
          require("os").tmpdir = () => require("path").dirname(process.argv[2]);
          async function main() {
            const guard = require(process.argv[1]).createTransportIntegrity();
            let filename = process.argv[2];
            if (process.argv[3] !== "capture") {
              fs.writeFileSync(filename, "original");
              guard.captureOptional(filename);
              if (process.argv[3] === "snapshot") {
                await guard.guardExec({exec: async (_command, args) => {
                  filename = args[2];
                  return 0;
                }}).exec("git", ["am", "--3way", filename]);
              }
              fs.unlinkSync(filename);
            }
            execFileSync("mkfifo", [filename]);
            try {
              if (process.argv[3] === "capture") guard.captureOptional(filename);
              else guard.assertUnchanged();
              process.exitCode = 1;
            } catch (error) {
              if (error.code !== "ERR_TRANSPORT_INTEGRITY" || error.reason !== "not a regular file") throw error;
            } finally {
              guard.dispose();
            }
          }
          main().catch(error => { console.error(error); process.exitCode = 1; });
        `,
        path.resolve("transport_integrity.cjs"),
        filename,
        operation,
      ],
      { timeout: 2000, encoding: "utf8" }
    );
    expect(result.error).toBeUndefined();
    expect(result.stderr).toBe("");
    expect(result.status).toBe(0);
  });

  it.each(["openSync", "readFileSync"])("preserves the filesystem cause and operation when %s fails", method => {
    const cause = Object.assign(new Error("private diagnostic detail"), { code: method === "openSync" ? "EACCES" : "EIO" });
    const spy = vi.spyOn(fs, method).mockImplementationOnce(() => {
      throw cause;
    });
    try {
      expect(() => transport.assertUnchanged()).toThrow(expect.objectContaining({ cause, operation: "verify", reason: "unable to read artifact" }));
      expect(() => transport.assertUnchanged()).toThrow(cause.code);
      expect(() => transport.assertUnchanged()).not.toThrow("private diagnostic detail");
    } finally {
      spy.mockRestore();
    }
  });

  it("distinguishes identity replacement from a same-size content change", () => {
    fs.writeFileSync(patch, Buffer.from([0x61, 0xfe, 0x62, 0x0a]));
    expect(() => transport.assertUnchanged()).toThrow(expect.objectContaining({ reason: "content changed" }));
    const next = createTransportIntegrity();
    try {
      next.captureOptional(patch);
      fs.renameSync(patch, `${patch}.old`);
      fs.copyFileSync(`${patch}.old`, patch);
      expect(() => next.assertUnchanged()).toThrow(expect.objectContaining({ reason: "file identity changed" }));
    } finally {
      next.dispose();
    }
  });

  it("does not read or hash full artifacts for unrelated local commands, but verifies bytes before publishing", async () => {
    transport.dispose();
    fs.writeFileSync(bundle, Buffer.alloc(8 * 1024 * 1024));
    transport = createTransportIntegrity();
    transport.captureOptional(patch);
    transport.captureOptional(bundle);
    const api = { exec: vi.fn(async () => 0) };
    const exec = transport.guardExec(api);
    await exec.exec("git", ["bundle", "verify", bundle]);
    const read = vi.spyOn(fs, "readFileSync");
    const hash = vi.spyOn(crypto, "createHash");
    try {
      for (let i = 0; i < 100; i++) await exec.exec("git", ["rev-parse", "HEAD"]);
      expect(read).not.toHaveBeenCalled();
      expect(hash).not.toHaveBeenCalled();
      transport.assertUnchanged();
      expect(read).toHaveBeenCalledTimes(3);
      expect(hash).toHaveBeenCalledTimes(3);
      fs.writeFileSync(patch, Buffer.from([0x61, 0xfe, 0x62, 0x0a]));
      await expect(exec.exec("git", ["push", "origin", "feature"])).rejects.toThrow(TransportIntegrityError);
      expect(api.exec.mock.calls.some(([, args]) => args[0] === "push")).toBe(false);
    } finally {
      read.mockRestore();
      hash.mockRestore();
    }
  });

  it("rechecks private snapshot bytes before application", async () => {
    let snapshot;
    const api = {
      exec: vi.fn(async (_command, args) => {
        snapshot = args[2];
        return 0;
      }),
    };
    const exec = transport.guardExec(api);
    await exec.exec("git", ["bundle", "verify", bundle]);
    fs.chmodSync(snapshot, 0o600);
    fs.writeFileSync(snapshot, "bundle modified");
    await expect(exec.exec("git", ["fetch", bundle, "HEAD:refs/test"])).rejects.toThrow(TransportIntegrityError);
    expect(api.exec).toHaveBeenCalledTimes(1);
  });

  it("reports structured integrity diagnostics and logs a safe error", async () => {
    vi.stubGlobal("core", { error: vi.fn() });
    try {
      const handler = withTransportIntegrity(async (_message, _ids, guard) => {
        guard.captureOptional(`${patch}.missing`);
      });
      const result = await handler({}, {});
      expect(result).toMatchObject({ success: false, error_type: "ERR_TRANSPORT_INTEGRITY" });
      expect(result.error).toContain("capture: unable to read artifact (ENOENT)");
      expect(core.error).toHaveBeenCalledWith(result.error);
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it("preserves integrity diagnostics when a handler wraps the original error", async () => {
    const handler = withTransportIntegrity(async (_message, _ids, guard) => {
      try {
        guard.captureOptional(`${patch}.missing`);
      } catch (error) {
        throw new Error("Failed to read patch", { cause: error });
      }
    });
    await expect(handler({}, {})).resolves.toMatchObject({ success: false, error_type: "ERR_TRANSPORT_INTEGRITY", error: expect.stringContaining("ENOENT") });
  });

  it("only permits registered local cleanup after failure, never remote writes or arbitrary ref deletion", async () => {
    const api = {
      exec: vi.fn(async (_command, args) => {
        if (args[0] === "am" && args[1] === "--3way") {
          fs.unlinkSync(patch);
          throw new Error("conflict");
        }
        return 0;
      }),
    };
    const handler = withTransportIntegrity(async (_message, _ids, guard) => {
      guard.captureOptional(patch);
      const exec = guard.guardExec(api);
      await expect(exec.exec("git", ["am", "--3way", patch])).rejects.toThrow(TransportIntegrityError);
      for (const args of [
        ["push", "origin", "feature"],
        ["update-ref", "-d", "refs/heads/main"],
        ["update-ref", "-d", "refs/bundles/not-registered"],
      ]) {
        await expect(exec.exec("git", args)).rejects.toThrow(TransportIntegrityError);
      }
    });
    await expect(handler({}, {})).resolves.toMatchObject({ success: false });
    expect(api.exec.mock.calls.map(([, args]) => args.slice(0, 2))).toEqual([
      ["am", "--3way"],
      ["am", "--abort"],
    ]);
  });

  it("surfaces cleanup errors and still removes private snapshots", async () => {
    let snapshotPath;
    const handler = withTransportIntegrity(async (_message, _ids, guard) => {
      guard.captureOptional(patch);
      await guard
        .guardExec({
          exec: async (_command, args) => {
            if (args[1] === "--abort") throw new Error("abort failed");
            snapshotPath = args[2];
            fs.unlinkSync(patch);
            throw new Error("conflict");
          },
        })
        .exec("git", ["am", "--3way", patch]);
    });
    await expect(handler({}, {})).rejects.toMatchObject({ message: "Failed to clean up safe-output Git state", errors: [expect.objectContaining({ message: "abort failed" })] });
    expect(fs.existsSync(path.dirname(snapshotPath))).toBe(false);
  });

  it("pins cleanup to the original checkout even when caller options change", async () => {
    const options = { cwd: directory };
    const api = {
      exec: vi.fn(async (_command, args) => {
        if (args[1] === "--3way") {
          fs.unlinkSync(patch);
          throw new Error("conflict");
        }
        return 0;
      }),
    };
    const handler = withTransportIntegrity(async (_message, _ids, guard) => {
      guard.captureOptional(patch);
      try {
        await guard.guardExec(api).exec("git", ["am", "--3way", patch], options);
      } finally {
        options.cwd = path.dirname(directory);
      }
    });
    await expect(handler({}, {})).resolves.toMatchObject({ success: false });
    expect(api.exec).toHaveBeenCalledWith("git", ["am", "--abort"], expect.objectContaining({ cwd: directory }));
  });

  it.each(["symlink", "directory"])("reports a %s capture as an integrity error", kind => {
    fs.unlinkSync(patch);
    if (kind === "symlink") fs.symlinkSync(bundle, patch);
    else fs.mkdirSync(patch);
    const next = createTransportIntegrity();
    expect(() => next.captureOptional(patch)).toThrow(TransportIntegrityError);
    next.dispose();
  });

  it("allows trusted transformations without changing or re-blessing the original baseline", async () => {
    transport.dispose();
    fs.writeFileSync(patch, "Subject: [PATCH] Update\n+#aw_example\n");
    transport = createTransportIntegrity();
    transport.captureOptional(patch);
    transport.transform(patch, bytes => Buffer.from(bytes.toString().replace("#aw_example", "#123")));
    transport.transform(patch, bytes => Buffer.from(bytes.toString().replace("Update", "Update [suffix]")));
    const api = {
      exec: vi.fn(async (_command, args) => {
        expect(fs.readFileSync(args[2], "utf8")).toBe("Subject: [PATCH] Update [suffix]\n+#123\n");
        return 0;
      }),
    };
    await transport.guardExec(api).exec("git", ["am", "--3way", patch]);
    expect(fs.readFileSync(patch, "utf8")).toBe("Subject: [PATCH] Update\n+#aw_example\n");
    fs.writeFileSync(patch, "Subject: [PATCH] Update\n+#aw_changed\n");
    expect(() => transport.transform(patch, bytes => bytes)).toThrow(TransportIntegrityError);
  });

  it.each(["return", "throw", "tamper"])("cleans up snapshots on handler %s", async outcome => {
    let snapshotPath;
    const handler = withTransportIntegrity(async (_message, _ids, guard) => {
      guard.captureOptional(patch);
      await guard
        .guardExec({
          exec: async (_command, args) => {
            snapshotPath = args[2];
            return 0;
          },
        })
        .exec("git", ["am", "--3way", patch]);
      if (outcome === "throw") throw new Error("handler failure");
      if (outcome === "tamper") {
        fs.unlinkSync(patch);
        guard.assertUnchanged();
      }
      return { success: true };
    });
    if (outcome === "throw") await expect(handler({}, {})).rejects.toThrow("handler failure");
    else {
      const result = await handler({}, {});
      expect(result.success).toBe(outcome === "return");
      if (outcome === "tamper") expect(result.error).toContain("SECURITY:");
    }
    expect(fs.existsSync(path.dirname(snapshotPath))).toBe(false);
  });
});
