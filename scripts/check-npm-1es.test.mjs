import assert from "node:assert/strict";
import { execFile as execFileCallback } from "node:child_process";
import { access, mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { test } from "node:test";
import { promisify } from "node:util";
import { cacheArguments, checkPackages, collectPackages, main, registry } from "./check-npm-1es.mjs";

const execFile = promisify(execFileCallback);

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
        "node_modules/local": { resolved: "packages/local-only", link: true },
        "packages/local-only": { name: "local-only", version: "1.0.0", dependencies: { public: "^1.0.0" } },
        "packages/local-only/node_modules/nested": { version: "2.0.0" },
        "node_modules/private": { version: "1.0.0", resolved: "https://private.example/pkg.tgz" },
        "node_modules/public": { version: "1.0.0", resolved: `${registry}public/-/public-1.0.0.tgz` },
      }),
    ],
  ]);
  assert.deepEqual([...packages.keys()], ["nested@2.0.0", "public@1.0.0"]);
  assert.deepEqual(external, ["package-lock.json: node_modules/git", "package-lock.json: node_modules/local", "package-lock.json: packages/local-only", "package-lock.json: node_modules/private"]);
});

test("excludes local link targets even under node_modules", () => {
  const { packages, external } = collectPackages([
    [
      "package-lock.json",
      lock({
        "node_modules/alias": { resolved: "node_modules/local", link: true },
        "node_modules/local": { name: "local", version: "1.0.0" },
        "packages/unlinked": { name: "unlinked", version: "1.0.0" },
      }),
    ],
  ]);
  assert.equal(packages.size, 0);
  assert.equal(external.length, 3);
});

test("checks registry entries without resolved or integrity fields", () => {
  const { packages, external } = collectPackages([["docs/package-lock.json", lock({ "node_modules/@antfu/install-pkg": { version: "1.1.0" } })]]);
  assert.deepEqual([...packages], [["@antfu/install-pkg@1.1.0", ["docs/package-lock.json: node_modules/@antfu/install-pkg"]]]);
  assert.deepEqual(external, []);
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

function npmFailure(code) {
  return Object.assign(new Error("sensitive npm output"), { code: 1, stderr: `npm error code ${code}\nsensitive npm output` });
}

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

async function repository(t, files) {
  const cwd = await mkdtemp(join(tmpdir(), "gh-aw-npm-1es-test-"));
  t.after(() => rm(cwd, { recursive: true, force: true }));
  await execFile("git", ["init", "--quiet"], { cwd });
  for (const [file, contents] of Object.entries(files)) {
    await mkdir(dirname(join(cwd, file)), { recursive: true });
    await writeFile(join(cwd, file), JSON.stringify(contents));
  }
  if (Object.keys(files).length) await execFile("git", ["add", "--", ...Object.keys(files)], { cwd });
  return cwd;
}

test("main rejects repositories without tracked lockfiles", async t => {
  const cwd = await repository(t, {});
  await writeFile(join(cwd, "package-lock.json"), JSON.stringify(lock({ "node_modules/pkg": { version: "1.0.0" } })));
  await assert.rejects(main({ cwd }), /No tracked npm lockfiles found/);
});

test("main reports external locations and rejects lockfiles without registry dependencies", async t => {
  const cwd = await repository(t, {
    "package-lock.json": lock({
      "node_modules/local": { link: true, resolved: "packages/local" },
      "packages/local": { name: "local", version: "1.0.0" },
    }),
  });
  const messages = [];
  await assert.rejects(main({ cwd, log: message => messages.push(message) }), /No npm registry dependencies found/);
  assert.deepEqual(messages, ["Not an npm registry dependency (requires separate access): package-lock.json: node_modules/local", "Not an npm registry dependency (requires separate access): package-lock.json: packages/local"]);
});

test("main discovers tracked root and nested lockfiles, deduplicates pins and cleans its cache", async t => {
  const cwd = await repository(t, {
    "package-lock.json": lock({ "node_modules/pkg": { version: "1.0.0" } }),
    "nested project/package-lock.json": lock({
      "node_modules/pkg": { version: "1.0.0" },
      "node_modules/other": { version: "2.0.0" },
    }),
  });
  const checked = [];
  const messages = [];
  let cache;
  await main({
    cwd,
    log: message => messages.push(message),
    run: async (command, args, options) => {
      if (command === "git") return execFile(command, args, options);
      assert.equal(command, "npm");
      assert.deepEqual(args.slice(0, 2), ["cache", "add"]);
      if (cache) assert.equal(options.cwd, cache);
      cache = options.cwd;
      await access(cache);
      checked.push(args[2]);
    },
  });
  assert.deepEqual(checked.sort(), ["other@2.0.0", "pkg@1.0.0"]);
  assert.match(messages[0], /Checking 2 exact npm versions from 2 lockfiles/);
  assert.equal(messages.at(-1), "All 2 npm registry dependencies are available in the 1ES feed.");
  await assert.rejects(access(cache), { code: "ENOENT" });
});

test("main cleans the cache on download failure without looking up replacements", async t => {
  const cwd = await repository(t, {
    "package-lock.json": lock({ "node_modules/pkg": { version: "1.0.0" } }),
    "docs/package-lock.json": lock({ "node_modules/pkg": { version: "1.0.0" } }),
  });
  let cache;
  let downloads = 0;
  const messages = [];
  await assert.rejects(
    main({
      cwd,
      log: message => messages.push(message),
      run: async (command, args, options) => {
        if (command === "git") return execFile(command, args, options);
        assert.equal(command, "npm");
        assert.deepEqual(args.slice(0, 3), ["cache", "add", "pkg@1.0.0"]);
        cache = options.cwd;
        await access(cache);
        downloads++;
        throw npmFailure("ETARGET");
      },
    }),
    error => {
      assert.match(error.message, /pkg@1\.0\.0.*failed \(ETARGET\)/);
      assert.match(error.message, /package-lock\.json: node_modules\/pkg/);
      assert.match(error.message, /docs\/package-lock\.json: node_modules\/pkg/);
      assert.doesNotMatch(error.message, /sensitive npm output|compatible release/);
      return true;
    }
  );
  assert.equal(downloads, 1);
  assert.equal(messages.length, 1);
  await assert.rejects(access(cache), { code: "ENOENT" });
});
