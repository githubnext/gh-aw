import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noGitCommandLoggingRule } from "./no-git-command-logging";

const tester = new RuleTester({ languageOptions: { ecmaVersion: 2022, sourceType: "commonjs" } });

describe("no-git-command-logging", () => {
  it("requires silent Git execution without changing other options", () => {
    tester.run("no-git-command-logging", noGitCommandLoggingRule, {
      valid: [
        `exec.exec("node", ["script.cjs"]);`,
        `regex.exec("git");`,
        `exec.exec("git", ["config"], { silent: true });`,
        `exec.getExecOutput("git", ["diff"], { ...opts, silent: true });`,
        `const options = { silent: true, cwd }; exec.exec("git", args, options);`,
        `const base = { silent: true }; const options = { ...base, cwd }; exec.exec("git", args, options);`,
        `exec.exec("git", args, { silent: true, ...{} });`,
        `exec.exec("git", args, { "silent": true });`,
        `exec["exec"]("git", args, { silent: true });`,
        `const { gitLoggingOptions } = require("./git_logging.cjs"); exec.exec("git", args, gitLoggingOptions(options));`,
        `const { gitLoggingOptions: logOptions } = require("./git_logging.cjs"); exec.getExecOutput("git", args, logOptions());`,
      ],
      invalid: [
        {
          code: `const gitLoggingOptions = () => ({}); exec.exec("git", args, gitLoggingOptions());`,
          output: `const gitLoggingOptions = () => ({}); exec.exec("git", args, { ...(gitLoggingOptions()), silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `exec.exec("git", ["status"]);`,
          output: `exec.exec("git", ["status"], { silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: "exec.getExecOutput(`git`, args, { cwd });",
          output: "exec.getExecOutput(`git`, args, { cwd, silent: true });",
          errors: [{ messageId: "silent" }],
        },
        {
          code: `exec.exec("git", args, { silent: false });`,
          output: `exec.exec("git", args, { silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `exec.exec("git", args, { silent: true, ...opts });`,
          output: `exec.exec("git", args, { silent: true, ...opts, silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `const options = { cwd }; exec.exec("git", args, options);`,
          output: `const options = { cwd }; exec.exec("git", args, { ...(options), silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `function run(execApi, opts) { return execApi.getExecOutput("git", args, opts); }`,
          output: `function run(execApi, opts) { return execApi.getExecOutput("git", args, { ...(opts), silent: true }); }`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `const command = "/usr/bin/git"; exec.exec(command, args);`,
          output: `const command = "/usr/bin/git"; exec.exec(command, args, { silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `const { getExecOutput: run } = require("@actions/exec"); run("git", args);`,
          output: `const { getExecOutput: run } = require("@actions/exec"); run("git", args, { silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `const run = exec.getExecOutput; run("git", args);`,
          output: `const run = exec.getExecOutput; run("git", args, { silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `execGetExecOutput("git", args, { cwd });`,
          output: `execGetExecOutput("git", args, { cwd, silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `exec.exec("git", args, ...execArgs);`,
          output: null,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `let opts = { silent: true }; opts = {}; exec.exec("git", args, opts);`,
          output: `let opts = { silent: true }; opts = {}; exec.exec("git", args, { ...(opts), silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `const opts = { silent: true }; opts.silent = false; exec.exec("git", args, opts);`,
          output: `const opts = { silent: true }; opts.silent = false; exec.exec("git", args, { ...(opts), silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `const silent = false; exec.exec("git", args, { silent });`,
          output: `const silent = false; exec.exec("git", args, { silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `exec.exec("git", args, { get silent() { return false; } });`,
          output: `exec.exec("git", args, { silent: true });`,
          errors: [{ messageId: "silent" }],
        },
        {
          code: `exec.exec("git", args, { silent() { return false; } });`,
          output: `exec.exec("git", args, { silent: true });`,
          errors: [{ messageId: "silent" }],
        },
      ],
    });
  });

  it("rejects inherited and file-directed child_process output", () => {
    tester.run("no-git-command-logging", noGitCommandLoggingRule, {
      valid: [
        `const { spawnSync } = require("child_process"); spawnSync("git", args);`,
        `const cp = require("node:child_process"); cp.spawnSync("git", args, { stdio: "pipe" });`,
        `const cp = require("child_process"); cp.execSync("git status", { stdio: ["ignore", "pipe", "ignore"] });`,
        `const cp = require("child_process"); cp.spawnSync("git", args, { stdio: "ignore" });`,
        `const cp = require("child_process"); cp.spawnSync("git", args, { stdio: [0, "pipe", "pipe", ...extraDescriptors] });`,
        `const cp = require("child_process"); cp.exec("git status", (error, stdout, stderr) => {});`,
        `const cp = require("child_process"); cp.execFile("git", args, (error, stdout, stderr) => {});`,
        `const cp = require("child_process"); cp.execFile("git", { stdio: "pipe" }, (error, stdout, stderr) => {});`,
        `const cp = require("child_process"); const callback = () => {}; cp.execFile("git", args, callback);`,
        `const cp = require("child_process"); function callback() {} cp.exec("git status", callback);`,
        `const cp = require("child_process"); cp.spawnSync("node", args, { stdio: "inherit" });`,
      ],
      invalid: [
        { code: `const cp = require("child_process"); cp.spawnSync("git", args, { stdio: "inherit" });`, errors: [{ messageId: "stdio" }] },
        { code: `const cp = require("child_process"); const run = cp.spawnSync; run("git", args, { stdio: "inherit" });`, errors: [{ messageId: "stdio" }] },
        { code: `const cp = require("child_process"); const spawnSync = "unrelated"; const run = cp.spawnSync; run("git", args, { stdio: "inherit" });`, errors: [{ messageId: "stdio" }] },
        { code: `const { execSync: run } = require("node:child_process"); run("git diff", { stdio: [0, 1, 2] });`, errors: [{ messageId: "stdio" }] },
        { code: `const { spawnSync } = require("child_process"); spawnSync("git", args, { stdio: ["ignore", fileFd, "pipe"] });`, errors: [{ messageId: "stdio" }] },
        { code: `const cp = require("child_process"); cp.spawnSync("git", args, { stdio: "pipe", ...options });`, errors: [{ messageId: "stdio" }] },
        { code: `const cp = require("child_process"); cp.spawnSync("git", args, options);`, errors: [{ messageId: "stdio" }] },
        { code: `const cp = require("child_process"); cp.execFileSync("git", { stdio: "inherit" });`, errors: [{ messageId: "stdio" }] },
        { code: `const cp = require("child_process"); cp.execFile("git", { stdio: "inherit" }, () => {});`, errors: [{ messageId: "stdio" }] },
        { code: `const cp = require("child_process"); cp.spawnSync("git", args, { stdio: [...fileDescriptors] });`, errors: [{ messageId: "stdio" }] },
        { code: `const cp = require("child_process"); cp.spawnSync("git", args, { stdio: [0, ...fileDescriptors] });`, errors: [{ messageId: "stdio" }] },
        { code: `const cp = require("child_process"); cp.spawnSync("git", args, { stdio: [0, "pipe", ...fileDescriptors] });`, errors: [{ messageId: "stdio" }] },
      ],
    });
  });

  it("allows redacted core logging and rejects raw/file-backed logging", () => {
    tester.run("no-git-command-logging", noGitCommandLoggingRule, {
      valid: [
        `const { stdout, exitCode } = exec.getExecOutput("git", args, { silent: true }); core.info(stdout.length); core.info(exitCode);`,
        `const result = exec.getExecOutput("node", args); core.info(result.stdout);`,
        `const stdout = "fixed diagnostic"; core.info(stdout);`,
        `const result = exec.getExecOutput("git", args, { silent: true }); core.info(result.exitCode);`,
        `const patch = execGitSync(args); fs.writeFileSync(patchPath, patch);`,
        `const result = exec.getExecOutput("git", args, { silent: true }); core.info(result.stdout); core.debug(result.stderr);`,
        `const { stdout: text } = exec.getExecOutput("git", args, { silent: true }); core.debug(text);`,
        `const { stderr } = exec.getExecOutput("git", args, { silent: true }); core.error("Failed: " + stderr);`,
        `const result = exec.getExecOutput("git", args, { silent: true }); const logger = core; logger.info(result.stdout);`,
        `const coreAlias = require("@actions/core"); const output = execGitSync(args); coreAlias.debug(output);`,
        `const result = exec.getExecOutput("git", args, { silent: true }); throw new Error(result.stderr);`,
      ],
      invalid: [
        { code: `const result = exec.getExecOutput("git", args, { silent: true }); server.debug(result.stdout);`, errors: [{ messageId: "output" }] },
        { code: `const { stdout: text } = exec.getExecOutput("git", args, { silent: true }); server.debug(text);`, errors: [{ messageId: "output" }] },
        { code: `const { stderr } = exec.getExecOutput("git", args, { silent: true }); server.debug("Failed: " + stderr);`, errors: [{ messageId: "output" }] },
        { code: `async function run() { const result = await exec.getExecOutput("git", args, { silent: true }); server.debug(result.stderr); }`, errors: [{ messageId: "output" }] },
        { code: `const output = execGitSync(args); const text = output.trim(); debugLog(\`Git output: \${text}\`);`, errors: [{ messageId: "output" }] },
        { code: `const cp = require("child_process"); const result = cp.spawnSync("git", args); logger.debug(result.stdout.toString());`, errors: [{ messageId: "output" }] },
        { code: `const cp = require("child_process"); const run = cp.spawnSync; const result = run("git", args); logger.debug(result.stdout.toString());`, errors: [{ messageId: "output" }] },
        { code: `const result = exec.getExecOutput("git", args, { silent: true }); const error = new Error(result.stderr); server.debug(error.message);`, errors: [{ messageId: "output" }] },
        { code: `const output = execGitSync(args); fs.appendFileSync("debug.log", output);`, errors: [{ messageId: "output" }] },
        { code: `const output = execGitSync(args); fs.writeFileSync(logPath, output);`, errors: [{ messageId: "output" }] },
      ],
    });
  });
});
