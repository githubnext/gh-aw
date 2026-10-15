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

export function collectPackages(lockfiles) {
  const packages = new Map();
  const external = [];
  for (const [file, lock] of lockfiles) {
    if (![2, 3].includes(lock.lockfileVersion) || !lock.packages) {
      throw new Error(`${file}: expected an npm v2 or v3 lockfile with packages`);
    }
    for (const [path, entry] of Object.entries(lock.packages)) {
      if (!path) continue;
      const location = `${file}: ${path}`;
      if (entry.link || (entry.resolved && !entry.resolved.startsWith("https://registry.npmjs.org/") && !entry.resolved.startsWith(registry))) {
        external.push(location);
        continue;
      }
      const name = entry.name || path.split("node_modules/").at(-1);
      if (!name || !/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(entry.version || "")) {
        throw new Error(`${location}: missing package name or exact version`);
      }
      const spec = `${name}@${entry.version}`;
      if (!packages.has(spec)) packages.set(spec, []);
      packages.get(spec).push(location);
    }
  }
  return { packages, external };
}

export function cacheArguments(spec, cache) {
  const args = ["cache", "add", spec, `--registry=${registry}`, `--cache=${cache}`, "--ignore-scripts", "--prefer-online", "--fetch-retries=0", "--fetch-timeout=15000", "--loglevel=error"];
  if (spec.startsWith("@")) args.push(`--${spec.split("/")[0]}:registry=${registry}`);
  return args;
}

export async function checkPackages(packages, cache, run = execFile) {
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
          const code = error.stderr?.match(/npm (?:error|ERR!) code ([A-Z0-9_]+)/)?.[1] || error.code || "npm error";
          failure ??= new Error(`${spec}: 1ES metadata or tarball download failed (${code}).\n${locations.join("\n")}\nCheck feed access and permissions to cache this exact version from upstream.`);
        }
      }
    })
  );
  if (failure) throw failure;
}

async function main() {
  const { stdout } = await execFile("git", ["ls-files", "-z", "--", "package-lock.json", "**/package-lock.json"], { cwd: root });
  const files = stdout.split("\0").filter(Boolean);
  if (!files.length) throw new Error("No tracked npm lockfiles found");
  const lockfiles = await Promise.all(files.map(async file => [file, JSON.parse(await readFile(join(root, file), "utf8"))]));
  const { packages, external } = collectPackages(lockfiles);
  for (const location of external) console.log(`Not an npm registry dependency (requires separate access): ${location}`);
  if (!packages.size) throw new Error("No npm registry dependencies found");
  console.log(`Checking ${packages.size} exact npm versions from ${files.length} lockfiles against ${registry}`);
  const cache = await mkdtemp(join(tmpdir(), "gh-aw-npm-1es-"));
  try {
    await checkPackages(packages, cache);
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
