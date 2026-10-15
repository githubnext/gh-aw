import assert from "node:assert/strict";
import { test } from "node:test";
import { cacheArguments, checkPackages, collectPackages, registry } from "./check-npm-1es.mjs";

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
