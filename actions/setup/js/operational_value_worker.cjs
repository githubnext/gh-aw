// @ts-check

const { spawn } = require("child_process");

/**
 * Own the Bash process group so deadlines also stop descendants holding pipes.
 * @param {string} scriptPath
 * @param {string} sessionPath
 * @param {number} timeout
 * @returns {Promise<string>}
 */
function runEvaluatorProcess(scriptPath, sessionPath, timeout) {
  return new Promise((resolve, reject) => {
    const child = spawn("/bin/bash", ["--noprofile", "--norc", "-euo", "pipefail", scriptPath, sessionPath], {
      stdio: ["ignore", "pipe", "pipe"],
      detached: process.platform !== "win32",
      env: process.env,
    });
    const chunks = [];
    let bytes = 0;
    /** @type {Error | undefined} */
    let failure;
    const terminate = () => {
      if (!child.pid) return;
      if (process.platform === "win32") {
        child.kill("SIGKILL");
        return;
      }
      try {
        process.kill(-child.pid, "SIGKILL");
      } catch (error) {
        if (!(error instanceof Error) || !("code" in error) || error.code !== "ESRCH") {
          failure ??= new Error("cannot terminate operational-value evaluator process group", { cause: error });
        }
      }
    };
    const timer = setTimeout(() => {
      failure = new Error("operational-value evaluator exceeded its time limit");
      terminate();
    }, timeout);
    /** @param {Buffer} data */
    const collect = data => {
      bytes += data.length;
      if (bytes > 1024 * 1024) {
        failure ??= new Error("operational-value evaluator exceeded its output limit");
        terminate();
        return false;
      }
      return true;
    };
    child.stdout.on("data", data => {
      if (collect(data)) chunks.push(data);
    });
    child.stderr.on("data", collect);
    child.on("error", error => {
      failure ??= new Error("cannot start operational-value evaluator", { cause: error });
    });
    child.on("exit", terminate);
    child.on("close", (code, signal) => {
      clearTimeout(timer);
      if (failure) reject(failure);
      else if (signal || code !== 0) reject(new Error("operational-value evaluator exited unsuccessfully"));
      else resolve(Buffer.concat(chunks).toString("utf8"));
    });
  });
}

if (require.main === module) {
  const [scriptPath, sessionPath, timeout] = process.argv.slice(2);
  runEvaluatorProcess(scriptPath, sessionPath, Number(timeout)).then(
    output => process.stdout.write(output),
    error => {
      process.stderr.write(error.message);
      process.exitCode = 1;
    }
  );
}

module.exports = { runEvaluatorProcess };
