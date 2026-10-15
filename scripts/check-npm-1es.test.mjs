import assert from "node:assert/strict";
import { test } from "node:test";
import { cacheArguments, checkPackages, collectPackages, compareVersions, findCompatibleRelease, registry } from "./check-npm-1es.mjs";

function lock(packages) {
  return { lockfileVersion: 3, packages: { "": { name: "test" }, ...packages } };
}

test("collects exact transitive, dev, optional, scoped and aliased versions across lockfiles", () => {
  const { packages, external } = collectPackages([
    [
      "a/package-lock.json",
      lock({
        "node_modules/a": { version: "1.2.3", dev: true },
        "node_modules/b/node_modules/a": { version: "1.2.3" },
        "node_modules/@scope/optional": { version: "2.0.0-beta.1", optional: true, os: ["win32"] },
        "node_modules/alias": { name: "original", version: "3.0.0" },
      }),
    ],
    ["b/package-lock.json", lock({ "node_modules/a": { version: "1.2.3" } })],
  ]);
  assert.deepEqual([...packages.keys()], ["a@1.2.3", "@scope/optional@2.0.0-beta.1", "original@3.0.0"]);
  assert.equal(packages.get("a@1.2.3").length, 3);
  assert.deepEqual(external, []);
});

test("reports Git, local and private registry dependencies separately", () => {
  const { packages, external } = collectPackages([
    [
      "package-lock.json",
      lock({
        "node_modules/git": { version: "1.0.0", resolved: "git+ssh://git@github.com/example/pkg.git#abc" },
        "node_modules/local": { resolved: "../local", link: true },
        "node_modules/private": { version: "1.0.0", resolved: "https://private.example/pkg.tgz" },
        "node_modules/public": { version: "1.0.0", resolved: `${registry}public/-/public-1.0.0.tgz` },
      }),
    ],
  ]);
  assert.deepEqual([...packages.keys()], ["public@1.0.0"]);
  assert.equal(external.length, 3);
});

test("rejects unsupported lockfiles and missing or non-exact versions", () => {
  assert.throws(() => collectPackages([["old.json", { lockfileVersion: 1 }]]), /old.json.*v2 or v3/);
  for (const version of [undefined, "latest", "^1.2.3"]) {
    assert.throws(() => collectPackages([["bad.json", lock({ "node_modules/pkg": { version } })]]), /bad.json.*exact version/);
  }
});

test("supports v2 lockfiles", () => {
  const input = lock({ "node_modules/pkg": { version: "1.0.0" } });
  input.lockfileVersion = 2;
  assert.ok(collectPackages([["package-lock.json", input]]).packages.has("pkg@1.0.0"));
});

test("collects declared ranges using nested module resolution, aliases and peers", () => {
  const input = lock({
    "node_modules/a": { version: "1.2.3", dependencies: { shared: "^2.0.0", alias: "npm:original@~3.0.0" }, peerDependencies: { peer: ">=4 <5" } },
    "node_modules/b": { version: "1.0.0", dependencies: { shared: "^1.0.0" } },
    "node_modules/a/node_modules/shared": { version: "2.1.0" },
    "node_modules/shared": { version: "1.1.0" },
    "node_modules/alias": { name: "original", version: "3.0.2" },
    "node_modules/peer": { version: "4.1.0" },
  });
  input.packages[""].devDependencies = { a: "^1.2.0", shared: "~1.1.0" };
  const { constraints } = collectPackages([["package-lock.json", input]]);
  assert.deepEqual([...constraints.get("a@1.2.3")], ["^1.2.0"]);
  assert.deepEqual([...constraints.get("shared@1.1.0")], ["~1.1.0", "^1.0.0"]);
  assert.deepEqual([...constraints.get("shared@2.1.0")], ["^2.0.0"]);
  assert.deepEqual([...constraints.get("original@3.0.2")], ["~3.0.0"]);
  assert.deepEqual([...constraints.get("peer@4.1.0")], [">=4 <5"]);
});

test("sorts releases by semver precedence, including prereleases and numeric identifiers", () => {
  const ordered = ["0.9.0", "1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta", "1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.2.0", "1.10.0", "2.0.0"];
  assert.deepEqual([...ordered].reverse().sort(compareVersions), ordered);
  assert.equal(compareVersions("1.0.0+one", "1.0.0+two"), 0);
});

function npmFailure(code) {
  return Object.assign(new Error("sensitive npm output"), { code: 1, stderr: `npm error code ${code}\nsensitive npm output` });
}

test("finds the lowest downloadable release satisfying every recorded range", async () => {
  const checked = [];
  const ranges = new Set(["^1.0.0", ">=1.2.0"]);
  const result = await findCompatibleRelease("@scope/pkg@1.4.0", ranges, "/cache", async (command, args) => {
    assert.equal(command, "npm");
    assert.ok(args.includes(`--@scope:registry=${registry}`));
    if (args[0] === "view") {
      assert.deepEqual(args.slice(2, 4), ["version", "--json"]);
      return { stdout: JSON.stringify(args[1].endsWith("^1.0.0") ? ["1.10.0", "1.0.0", "1.3.0", "1.2.0"] : ["1.10.0", "1.3.0", "1.2.0"]) };
    }
    checked.push(args[2]);
    if (args[2] === "@scope/pkg@1.2.0") throw npmFailure("E404");
    return {};
  });
  assert.deepEqual(checked, ["@scope/pkg@1.2.0", "@scope/pkg@1.3.0"]);
  assert.match(result, /First compatible release.*@scope\/pkg@1\.3\.0/);
});

test("handles a single matching release and never retries the failed pin", async () => {
  let downloads = 0;
  const result = await findCompatibleRelease("pkg@1.2.3", new Set(["1.2.3"]), "/cache", async (command, args) => {
    if (args[0] === "view") return { stdout: '"1.2.3"' };
    downloads++;
  });
  assert.match(result, /No downloadable compatible release/);
  assert.equal(downloads, 0);
  assert.match(
    await findCompatibleRelease("pkg@1.2.3", new Set(["^1.0.0"]), "/cache", async (command, args) => {
      if (args[0] === "view") return { stdout: '"1.2.4"' };
    }),
    /pkg@1\.2\.4/
  );
});

test("does not invent compatibility for unreferenced pins or widen unsatisfied ranges", async () => {
  assert.match(await findCompatibleRelease("pkg@1.0.0", undefined, "/cache"), /cannot be determined/);
  for (const stdout of ["", "[]"]) {
    assert.match(await findCompatibleRelease("pkg@1.0.0", new Set(["^1.0.0"]), "/cache", async () => ({ stdout })), /No downloadable compatible release/);
  }
  for (const code of ["ETARGET", "E404"]) {
    assert.match(
      await findCompatibleRelease("pkg@1.0.0", new Set(["^1.0.0"]), "/cache", async () => {
        throw npmFailure(code);
      }),
      /No compatible release.*\^1\.0\.0/
    );
  }
});

test("surfaces feed access errors instead of trying more releases", async () => {
  for (const code of ["E401", "E403", "ENOTFOUND", "ETIMEDOUT"]) {
    await assert.rejects(
      findCompatibleRelease("pkg@1.0.0", new Set(["^1.0.0"]), "/cache", async () => {
        throw npmFailure(code);
      }),
      new RegExp(`lookup failed \\(${code}\\)`)
    );
  }
  await assert.rejects(
    findCompatibleRelease("pkg@1.0.0", new Set(["^1.0.0"]), "/cache", async (command, args) => {
      if (args[0] === "view") return { stdout: '["1.0.1", "1.0.2"]' };
      throw npmFailure("E403");
    }),
    /tarball check failed \(E403\)/
  );
});

test("rejects invalid version metadata", async () => {
  for (const stdout of ['{"error":"unavailable"}', '["latest"]']) {
    await assert.rejects(
      findCompatibleRelease("pkg@1.0.0", new Set(["^1.0.0"]), "/cache", async () => ({ stdout })),
      /invalid version metadata/
    );
  }
});

test("forces the feed including scoped overrides and disables scripts and retries", () => {
  const args = cacheArguments("@scope/pkg@1.2.3", "/temporary/cache");
  assert.deepEqual(args.slice(0, 4), ["cache", "add", "@scope/pkg@1.2.3", `--registry=${registry}`]);
  for (const flag of ["--ignore-scripts", "--prefer-online", "--fetch-retries=0", "--fetch-timeout=15000", "--cache=/temporary/cache", `--@scope:registry=${registry}`]) {
    assert.ok(args.includes(flag), flag);
  }
});

test("checks all versions once with bounded concurrency and no installation", async () => {
  const packages = new Map(Array.from({ length: 20 }, (_, i) => [`pkg@1.0.${i}`, ["package-lock.json"]]));
  const checked = [];
  let active = 0;
  let peak = 0;
  await checkPackages(packages, "/cache", async (command, args, options) => {
    assert.equal(command, "npm");
    assert.equal(options.cwd, "/cache");
    assert.equal(options.timeout, 60000);
    assert.deepEqual(args.slice(0, 2), ["cache", "add"]);
    checked.push(args[2]);
    peak = Math.max(peak, ++active);
    await new Promise(resolve => setImmediate(resolve));
    active--;
  });
  assert.equal(checked.length, packages.size);
  assert.equal(new Set(checked).size, packages.size);
  assert.equal(peak, 8);
});

test("fails on unavailable packages and waits for active checks before cleanup", async () => {
  const packages = new Map(Array.from({ length: 20 }, (_, i) => [`pkg@1.0.${i}`, ["docs/package-lock.json"]]));
  let calls = 0;
  let active = 0;
  await assert.rejects(
    checkPackages(packages, "/cache", async () => {
      calls++;
      active++;
      await new Promise(resolve => setImmediate(resolve));
      active--;
      throw Object.assign(new Error("sensitive npm output"), { code: 1, stderr: "npm error code ETARGET\nsensitive npm output" });
    }),
    error => {
      assert.match(error.message, /pkg@1\.0\.0.*1ES.*failed/);
      assert.match(error.message, /docs\/package-lock.json/);
      assert.match(error.message, /ETARGET/);
      assert.doesNotMatch(error.message, /sensitive npm output/);
      return true;
    }
  );
  assert.equal(calls, 8);
  assert.equal(active, 0);
});

test("adds compatible release diagnostics while preserving failure of the pinned dependency", async () => {
  const packages = new Map([["pkg@1.0.0", ["package-lock.json"]]]);
  await assert.rejects(
    checkPackages(
      packages,
      "/cache",
      async (command, args) => {
        if (args[0] === "view") return { stdout: '["1.0.1"]' };
        if (args[2] === "pkg@1.0.0") throw npmFailure("ETARGET");
      },
      new Map([["pkg@1.0.0", new Set(["^1.0.0"])]])
    ),
    /pkg@1\.0\.0.*failed[\s\S]*First compatible release.*pkg@1\.0\.1/
  );
});
