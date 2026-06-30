import { spawn, type SpawnOptions } from "node:child_process";
import { constants as fsConstants } from "node:fs";
import { access } from "node:fs/promises";
import { join } from "node:path";

const INSTALL_COMMAND = "gh extension install github/gh-aw";
const GH_INSTALL_URL = "https://cli.github.com";
const LOG = "[dashboard-cli]";

type ExecError = Error & {
  code?: string | number;
  syscall?: string;
  path?: string;
  stderr?: string;
  stdout?: string;
  output?: string;
};

type ExecCallback = (err: ExecError | null, stdout: string, stderr: string) => void;

type ExecFileLike = (file: string, args: string[], options: ExecOptions, callback: ExecCallback) => void;

type AccessLike = typeof access;

interface ExecOptions {
  env?: NodeJS.ProcessEnv;
  cwd?: string;
  maxBuffer?: number;
}

interface RunExecOptions {
  combineIO?: boolean;
  execFileFn?: ExecFileLike;
  env?: NodeJS.ProcessEnv;
}

interface RunnerOptions {
  getWorkspacePath: () => string;
  accessFn?: AccessLike;
  execFileFn?: ExecFileLike;
  platform?: NodeJS.Platform;
  env?: NodeJS.ProcessEnv;
  /** Pre-built memoized resolver; when provided, `findDevBinary` is never called directly. */
  resolveBin?: () => Promise<string | null>;
}

export interface GhAwStatus {
  available: boolean;
  source: "dev-binary" | "gh-extension" | "gh-not-found" | "missing" | "error";
  version: string;
  command: string;
  installCommand: string;
  installUrl?: string;
  message?: string;
}

export type GhAwRunner = ((args: string[]) => Promise<string>) & {
  getStatus: () => Promise<GhAwStatus>;
};

function combineOutput(stdout: string, stderr: string): string {
  return [stdout, stderr].filter(Boolean).join("\n").trim();
}

function spawnExecFile(file: string, args: string[], options: ExecOptions, callback: ExecCallback): void {
  const { env, cwd, maxBuffer = 10 * 1024 * 1024 } = options ?? {};
  const spawnOptions: SpawnOptions = { env, cwd, stdio: ["ignore", "pipe", "pipe"], windowsHide: true, detached: true };
  console.error(`${LOG} spawn file=${file} args=${JSON.stringify(args)} cwd=${cwd}`);
  const proc = spawn(file, args, spawnOptions);
  const stdoutChunks: Buffer[] = [];
  const stderrChunks: Buffer[] = [];
  let stdoutLen = 0;
  let stderrLen = 0;
  let overflowed = false;

  proc.stdout?.on("data", (chunk: Buffer) => {
    stdoutLen += chunk.length;
    if (stdoutLen > maxBuffer) {
      overflowed = true;
      return;
    }
    stdoutChunks.push(chunk);
  });

  proc.stderr?.on("data", (chunk: Buffer) => {
    stderrLen += chunk.length;
    if (stderrLen > maxBuffer) {
      overflowed = true;
      return;
    }
    stderrChunks.push(chunk);
  });

  proc.on("error", err => {
    console.error(`${LOG} spawn error file=${file} args=${JSON.stringify(args)}: ${(err as ExecError).message}`);
    callback(err as ExecError, "", "");
  });
  proc.on("close", code => {
    const stdout = Buffer.concat(stdoutChunks).toString("utf8");
    const stderr = Buffer.concat(stderrChunks).toString("utf8");
    if (code !== 0 || stderr) {
      console.error(`${LOG} spawn close file=${file} code=${code} stdout=${stdout.length}B stderr=${stderr.length}B${stderr ? ` stderr: ${stderr.slice(0, 300)}` : ""}`);
    }
    if (overflowed) {
      const err: ExecError = new Error("stdout/stderr maxBuffer exceeded");
      err.code = "ERR_CHILD_PROCESS_STDIO_MAXBUFFER";
      callback(err, stdout, stderr);
    } else if (code !== 0) {
      const err: ExecError = new Error(`Command failed with exit code ${code}`);
      err.code = code ?? 1;
      callback(err, stdout, stderr);
    } else {
      callback(null, stdout, stderr);
    }
  });
}

function execp(bin: string, args: string[], cwd: string, { combineIO = false, execFileFn = spawnExecFile, env = process.env }: RunExecOptions = {}): Promise<string> {
  return new Promise((resolve, reject) => {
    execFileFn(
      bin,
      args,
      {
        cwd,
        env: { ...env, CI: "1", NO_COLOR: "1", GH_NO_UPDATE_NOTIFIER: "1" },
        maxBuffer: 10 * 1024 * 1024,
      },
      (err, stdout, stderr) => {
        const output = combineOutput(stdout ?? "", stderr ?? "");
        if (err) {
          reject(Object.assign(err, { stderr: stderr ?? "", stdout: stdout ?? "", output }));
          return;
        }
        resolve(combineIO ? output : stdout);
      }
    );
  });
}

function parseVersionFromOutput(output: string): string {
  const trimmed = String(output ?? "").trim();
  if (!trimmed) return "";
  const match = trimmed.match(/gh(?:-aw| aw) version ([^\r\n]+)/i);
  return match?.[1]?.trim() ?? "";
}

function isMissingGh(error: unknown): boolean {
  const e = error as ExecError | undefined;
  return e?.code === "ENOENT" && e?.syscall === "spawn" && e?.path === "gh";
}

function isMissingGhAwExtension(error: unknown): boolean {
  const e = error as ExecError | undefined;
  const output = String(e?.output ?? e?.stderr ?? e?.message ?? "");
  return /extension not found:\s*aw/i.test(output) || /unknown command ["']aw["'] for ["']gh["']/i.test(output);
}

async function findDevBinary(cwd: string, accessFn: AccessLike = access, platform: NodeJS.Platform = process.platform): Promise<string | null> {
  const devBin = join(cwd, platform === "win32" ? "gh-aw.exe" : "gh-aw");
  try {
    await accessFn(devBin, fsConstants.X_OK);
    console.error(`${LOG} findDevBinary found: ${devBin}`);
    return devBin;
  } catch {
    console.error(`${LOG} findDevBinary not found at ${devBin}, falling back to gh extension`);
    return null;
  }
}

export function createGhAwRunner({ getWorkspacePath, accessFn = access, execFileFn = spawnExecFile, platform = process.platform, env = process.env, resolveBin }: RunnerOptions): (args: string[]) => Promise<string> {
  // Memoize per cwd so findDevBinary is called at most once per workspace path.
  const binCache = new Map<string, Promise<string | null>>();
  const _resolveBin =
    resolveBin ??
    (() => {
      const cwd = getWorkspacePath();
      if (!binCache.has(cwd)) {
        binCache.set(cwd, findDevBinary(cwd, accessFn, platform));
      }
      return binCache.get(cwd)!;
    });

  function runExec(bin: string, args: string[], cwd: string, options?: RunExecOptions): Promise<string> {
    return execp(bin, args, cwd, { ...options, execFileFn, env });
  }

  return async function runGhAw(args: string[]): Promise<string> {
    const cwd = getWorkspacePath();
    const devBin = await _resolveBin();
    if (devBin) {
      console.error(`${LOG} runGhAw using dev-binary: ${devBin} args=${JSON.stringify(args)} cwd=${cwd}`);
      return runExec(devBin, args, cwd);
    }
    console.error(`${LOG} runGhAw using gh extension args=${JSON.stringify(args)} cwd=${cwd}`);
    return runExec("gh", ["aw", ...args], cwd);
  };
}

export function createGhAwRunnerWithStatus(options: RunnerOptions): GhAwRunner {
  // One shared per-cwd memoized resolver so findDevBinary is called at most once,
  // even across concurrent runGhAw() calls and getStatus().
  const binCache = new Map<string, Promise<string | null>>();
  const resolveBin = (): Promise<string | null> => {
    const cwd = options.getWorkspacePath();
    if (!binCache.has(cwd)) {
      binCache.set(cwd, findDevBinary(cwd, options.accessFn ?? access, options.platform ?? process.platform));
    }
    return binCache.get(cwd)!;
  };

  const runGhAw = createGhAwRunner({ ...options, resolveBin }) as GhAwRunner;
  const getStatus = async (): Promise<GhAwStatus> => {
    const cwd = options.getWorkspacePath();
    const devBin = await resolveBin();

    if (devBin) {
      const output = await execp(devBin, ["version"], cwd, {
        combineIO: true,
        execFileFn: options.execFileFn ?? spawnExecFile,
        env: options.env ?? process.env,
      });
      const status: GhAwStatus = {
        available: true,
        source: "dev-binary",
        version: parseVersionFromOutput(output) || "unknown",
        command: `${devBin} version`,
        installCommand: INSTALL_COMMAND,
      };
      console.error(`${LOG} getStatus: available=${status.available} source=${status.source} version=${status.version} cwd=${cwd}`);
      return status;
    }

    try {
      const output = await execp("gh", ["aw", "version"], cwd, {
        combineIO: true,
        execFileFn: options.execFileFn ?? spawnExecFile,
        env: options.env ?? process.env,
      });
      const status: GhAwStatus = {
        available: true,
        source: "gh-extension",
        version: parseVersionFromOutput(output) || "unknown",
        command: "gh aw version",
        installCommand: INSTALL_COMMAND,
      };
      console.error(`${LOG} getStatus: available=${status.available} source=${status.source} version=${status.version} cwd=${cwd}`);
      return status;
    } catch (error) {
      if (isMissingGh(error)) {
        console.error(`${LOG} getStatus error: gh not found in PATH cwd=${cwd}`);
        return {
          available: false,
          source: "gh-not-found",
          version: "",
          command: "gh aw version",
          installCommand: INSTALL_COMMAND,
          installUrl: GH_INSTALL_URL,
          message: "Install the GitHub CLI to use this dashboard.",
        };
      }

      if (isMissingGhAwExtension(error)) {
        console.error(`${LOG} getStatus error: gh aw extension not installed cwd=${cwd}`);
        return {
          available: false,
          source: "missing",
          version: "",
          command: "gh aw version",
          installCommand: INSTALL_COMMAND,
          message: "gh aw is not installed. Install the GitHub CLI extension to use the dashboard outside a local dev build.",
        };
      }

      const e = error as ExecError | undefined;
      const message = String(e?.output ?? e?.stderr ?? e?.message ?? "Failed to detect gh aw.");
      console.error(`${LOG} getStatus error: ${message} cwd=${cwd}`);
      return {
        available: false,
        source: "error",
        version: "",
        command: "gh aw version",
        installCommand: INSTALL_COMMAND,
        message,
      };
    }
  };

  runGhAw.getStatus = getStatus;
  return runGhAw;
}
