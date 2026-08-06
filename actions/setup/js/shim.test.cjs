import { describe, expect, it } from "vitest";
import { spawnSync } from "child_process";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

describe("core shim", () => {
  it("emits escaped add-mask workflow commands for secrets", () => {
    const result = spawnSync(process.execPath, ["-e", 'require("./shim.cjs"); core.setSecret("a%b\\nc\\r");'], {
      cwd: __dirname,
      encoding: "utf8",
    });

    expect(result.status).toBe(0);
    expect(result.stdout).toBe("");
    expect(result.stderr).toBe("::add-mask::a%25b%0Ac%0D\n");
  });

  it("adds setSecret to an existing partial core object", () => {
    const result = spawnSync(process.execPath, ["-e", 'global.core = {}; require("./shim.cjs"); core.setSecret("secret");'], {
      cwd: __dirname,
      encoding: "utf8",
    });

    expect(result.status).toBe(0);
    expect(result.stderr).toBe("::add-mask::secret\n");
  });
});
