import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";
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
    expect(() => next.captureOptional(patch)).toThrow(TransportIntegrityError);
    next.dispose();
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
