#!/usr/bin/env node

import { execFile as execFileCallback } from "node:child_process";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { promisify } from "node:util";

const execFile = promisify(execFileCallback);
export const registry = "https://ms-feed-25.pkgs.visualstudio.com/1es-public/_packaging/npm-public/npm/registry/";
const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const exactVersion = /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;

function resolveDependency(packages, parent, name) {
  while (true) {
    const path = `${parent ? `${parent}/` : ""}node_modules/${name}`;
    if (packages[path]) return path;
    if (!parent) return undefined;
    parent = parent.slice(0, parent.lastIndexOf("node_modules/")).replace(/\/$/, "");
  }
}

export function collectPackages(lockfiles) {
  const packages = new Map();
  const constraints = new Map();
  const external = [];
  for (const [file, lock] of lockfiles) {
    if (![2, 3].includes(lock.lockfileVersion) || !lock.packages) {
      throw new Error(`${file}: expected an npm v2 or v3 lockfile with packages`);
    }
    const specs = new Map();
    for (const [path, entry] of Object.entries(lock.packages)) {
      if (!path) continue;
      const location = `${file}: ${path}`;
      if (entry.link || (entry.resolved && !entry.resolved.startsWith("https://registry.npmjs.org/") && !entry.resolved.startsWith(registry))) {
        external.push(location);
        continue;
      }
      const name = entry.name || path.split("node_modules/").at(-1);
      if (!name || !exactVersion.test(entry.version || "")) {
        throw new Error(`${location}: missing package name or exact version`);
      }
      const spec = `${name}@${entry.version}`;
      specs.set(path, spec);
      if (!packages.has(spec)) packages.set(spec, []);
      packages.get(spec).push(location);
    }
    for (const [parent, entry] of Object.entries(lock.packages)) {
      const requirements = { ...entry.dependencies, ...entry.optionalDependencies, ...(!parent && entry.devDependencies) };
      // Peer constraints must also hold for the installed version.
      for (const requirementsByKind of [requirements, entry.peerDependencies || {}]) {
        for (const [name, requirement] of Object.entries(requirementsByKind)) {
          const spec = specs.get(resolveDependency(lock.packages, parent, name));
          if (!spec) continue;
          const range = requirement.startsWith("npm:") ? requirement.slice(requirement.lastIndexOf("@") + 1) : requirement;
          if (!constraints.has(spec)) constraints.set(spec, new Set());
          constraints.get(spec).add(range);
        }
      }
    }
  }
  return { packages, external, constraints };
}

function registryArguments(spec, cache) {
  const args = [`--registry=${registry}`, `--cache=${cache}`, "--ignore-scripts", "--prefer-online", "--fetch-retries=0", "--fetch-timeout=15000", "--loglevel=error"];
  if (spec.startsWith("@")) args.push(`--${spec.split("/")[0]}:registry=${registry}`);
  return args;
}

export function cacheArguments(spec, cache) {
  return ["cache", "add", spec, ...registryArguments(spec, cache)];
}

function npmErrorCode(error) {
  return error.stderr?.match(/npm (?:error|ERR!) code ([A-Z0-9_]+)/)?.[1] || error.code || "npm error";
}

export function compareVersions(a, b) {
  const [aCore, aPre] = a.split("+")[0].split(/-(.*)/);
  const [bCore, bPre] = b.split("+")[0].split(/-(.*)/);
  const core = aCore.split(".").map(Number);
  const other = bCore.split(".").map(Number);
  for (let i = 0; i < 3; i++) {
    if (core[i] !== other[i]) return core[i] - other[i];
  }
  if (aPre === bPre) return 0;
  if (aPre === undefined) return 1;
  if (bPre === undefined) return -1;
  const identifiers = aPre.split(".");
  const others = bPre.split(".");
  for (let i = 0; i < Math.max(identifiers.length, others.length); i++) {
    const left = identifiers[i];
    const right = others[i];
    if (left === right) continue;
    if (left === undefined) return -1;
    if (right === undefined) return 1;
    const leftNumeric = /^\d+$/.test(left);
    const rightNumeric = /^\d+$/.test(right);
    if (leftNumeric && rightNumeric) return Number(left) - Number(right);
    if (leftNumeric !== rightNumeric) return leftNumeric ? -1 : 1;
    return left < right ? -1 : 1;
  }
  return 0;
}

export async function findCompatibleRelease(spec, ranges, cache, run = execFile) {
  if (!ranges?.size) return "No recorded dependency ranges; a compatible replacement cannot be determined.";
  const name = spec.slice(0, spec.lastIndexOf("@"));
  let candidates;
  for (const range of ranges) {
    let stdout;
    try {
      ({ stdout } = await run("npm", ["view", `${name}@${range}`, "version", "--json", ...registryArguments(name, cache)], { cwd: cache, timeout: 60000 }));
    } catch (error) {
      const code = npmErrorCode(error);
      if (["E404", "ETARGET"].includes(code)) return `No compatible release in 1ES satisfies all recorded ranges: ${[...ranges].join(", ")}.`;
      throw new Error(`Compatible release lookup failed (${code}).`);
    }
    const result = stdout.trim() ? JSON.parse(stdout) : [];
    const versions = typeof result === "string" ? [result] : result;
    if (!Array.isArray(versions) || versions.some(version => typeof version !== "string" || !exactVersion.test(version))) {
      throw new Error("Compatible release lookup returned invalid version metadata.");
    }
    candidates = candidates === undefined ? versions : candidates.filter(version => versions.includes(version));
  }
  for (const version of [...new Set(candidates)].sort(compareVersions)) {
    const candidate = `${name}@${version}`;
    if (candidate === spec) continue;
    try {
      await run("npm", cacheArguments(candidate, cache), { cwd: cache, timeout: 60000 });
      return `First compatible release available in 1ES: ${candidate} (recorded ranges: ${[...ranges].join(", ")}). Review and regenerate the lockfile before using it.`;
    } catch (error) {
      const code = npmErrorCode(error);
      if (!["E404", "ETARGET"].includes(code)) throw new Error(`Compatible release tarball check failed (${code}).`);
    }
  }
  return `No downloadable compatible release in 1ES satisfies all recorded ranges: ${[...ranges].join(", ")}.`;
}

export async function checkPackages(packages, cache, run = execFile, constraints = new Map()) {
  const pending = [...packages];
  let next = 0;
  let failure;
  await Promise.all(
    Array.from({ length: Math.min(8, pending.length) }, async () => {
      while (!failure && next < pending.length) {
        const [spec, locations] = pending[next++];
        try {
          // npm cache add fetches tarballs without installing or filtering by platform.
          await run("npm", cacheArguments(spec, cache), { cwd: cache, timeout: 60000 });
        } catch (error) {
          failure ??= { spec, locations, code: npmErrorCode(error) };
        }
      }
    })
  );
  if (failure) {
    const { spec, locations, code } = failure;
    let suggestion = "Check feed access and permissions to cache this exact version from upstream.";
    if (["E404", "ETARGET"].includes(code)) {
      try {
        suggestion = await findCompatibleRelease(spec, constraints.get(spec), cache, run);
      } catch (error) {
        suggestion = error.message;
      }
    }
    throw new Error(`${spec}: 1ES metadata or tarball download failed (${code}).\n${locations.join("\n")}\n${suggestion}`);
  }
}

async function main() {
  const { stdout } = await execFile("git", ["ls-files", "-z", "--", "package-lock.json", "**/package-lock.json"], { cwd: root });
  const files = stdout.split("\0").filter(Boolean);
  if (!files.length) throw new Error("No tracked npm lockfiles found");
  const lockfiles = await Promise.all(files.map(async file => [file, JSON.parse(await readFile(join(root, file), "utf8"))]));
  const { packages, external, constraints } = collectPackages(lockfiles);
  for (const location of external) console.log(`Not an npm registry dependency (requires separate access): ${location}`);
  if (!packages.size) throw new Error("No npm registry dependencies found");
  console.log(`Checking ${packages.size} exact npm versions from ${files.length} lockfiles against ${registry}`);
  const cache = await mkdtemp(join(tmpdir(), "gh-aw-npm-1es-"));
  try {
    await checkPackages(packages, cache, execFile, constraints);
    console.log(`All ${packages.size} npm registry dependencies are available in the 1ES feed.`);
  } finally {
    await rm(cache, { recursive: true, force: true });
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main().catch(error => {
    console.error(`Error: ${error.message}`);
    process.exitCode = 1;
  });
}
