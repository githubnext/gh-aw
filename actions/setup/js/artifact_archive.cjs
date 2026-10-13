// @ts-check
"use strict";

const fs = require("fs");
const path = require("path");
const zlib = require("zlib");
const { artifactError } = require("./resolve_transport_paths.cjs");
const { DefaultArtifactClient } = require("./artifact_client.cjs");

const MAX_ARCHIVE = 128 * 1024 * 1024;
const MAX_FILE = 128 * 1024 * 1024;
const MAX_TOTAL = 512 * 1024 * 1024;
const checkedRoots = new Set();
const CRC_TABLE = Array.from({ length: 256 }, (_, index) => {
  let value = index;
  for (let bit = 0; bit < 8; bit++) value = value & 1 ? 0xedb88320 ^ (value >>> 1) : value >>> 1;
  return value >>> 0;
});

function crc32(bytes) {
  let value = 0xffffffff;
  for (const byte of bytes) value = CRC_TABLE[(value ^ byte) & 255] ^ (value >>> 8);
  return (value ^ 0xffffffff) >>> 0;
}

function decodeName(bytes, flags) {
  if (!(flags & 0x800) && bytes.some(byte => byte > 127)) throw artifactError("archive filename encoding is unsupported");
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    throw artifactError("archive filename encoding is invalid");
  }
}

function validateExtra(bytes) {
  for (let position = 0; position < bytes.length;) {
    if (position + 4 > bytes.length) throw artifactError("malformed archive extra field");
    const id = bytes.readUInt16LE(position);
    const length = bytes.readUInt16LE(position + 2);
    position += 4;
    if (position + length > bytes.length || id === 1 || id === 0x7075) throw artifactError("unsupported archive filename or ZIP64 extension");
    position += length;
  }
}

function safeName(name) {
  const normalized = name.replaceAll("\\", "/");
  if (/^[\\/]|^[A-Za-z]:|[\x00-\x1f\x7f:]/.test(normalized)) throw artifactError("unsafe archive entry path");
  const components = normalized.replace(/\/$/, "").split("/");
  if (components.length > 32 || components.some(component => !component || component === "." || component === ".." || /[. ]$/.test(component) || /^(con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\.|$)/i.test(component)))
    throw artifactError("unsafe archive entry path");
  return components.join("/");
}

/** Validate every ZIP entry before creating any destination file. */
function inspectArchive(bytes) {
  if (!Buffer.isBuffer(bytes) || bytes.length < 22 || bytes.length > MAX_ARCHIVE) throw artifactError("artifact archive exceeds its bounded ZIP contract");
  let end = -1;
  for (let position = bytes.length - 22; position >= Math.max(0, bytes.length - 65557); position--) {
    if (bytes.readUInt32LE(position) === 0x06054b50 && position + 22 + bytes.readUInt16LE(position + 20) === bytes.length) {
      end = position;
      break;
    }
  }
  if (end < 0 || bytes.readUInt16LE(end + 4) !== 0 || bytes.readUInt16LE(end + 6) !== 0) throw artifactError("unsupported or malformed artifact archive");
  const count = bytes.readUInt16LE(end + 10);
  const directorySize = bytes.readUInt32LE(end + 12);
  const directoryStart = bytes.readUInt32LE(end + 16);
  if (count < 1 || count > 4096 || bytes.readUInt16LE(end + 8) !== count || directoryStart + directorySize !== end) throw artifactError("unsupported archive directory");
  /** @type {any[]} */
  const entries = [];
  const paths = new Map();
  let total = 0;
  let position = directoryStart;
  for (let index = 0; index < count; index++) {
    if (position + 46 > end || bytes.readUInt32LE(position) !== 0x02014b50) throw artifactError("malformed archive directory entry");
    const flags = bytes.readUInt16LE(position + 8);
    const method = bytes.readUInt16LE(position + 10);
    const crc = bytes.readUInt32LE(position + 16);
    const compressed = bytes.readUInt32LE(position + 20);
    const size = bytes.readUInt32LE(position + 24);
    const nameLength = bytes.readUInt16LE(position + 28);
    const extraLength = bytes.readUInt16LE(position + 30);
    const commentLength = bytes.readUInt16LE(position + 32);
    const attributes = bytes.readUInt32LE(position + 38);
    const offset = bytes.readUInt32LE(position + 42);
    const next = position + 46 + nameLength + extraLength + commentLength;
    if (next > end || bytes.readUInt16LE(position + 34) !== 0 || flags & ~0x80e || ![0, 8].includes(method) || size > MAX_FILE || compressed > MAX_ARCHIVE || offset >= directoryStart)
      throw artifactError("unsupported or unbounded archive entry");
    const nameBytes = bytes.subarray(position + 46, position + 46 + nameLength);
    const name = decodeName(nameBytes, flags);
    const relative = safeName(name);
    const directory = /[\\/]$/.test(name);
    const mode = (attributes >>> 16) & 0xf000;
    if (![0, 0x8000, 0x4000].includes(mode) || (mode === 0x4000 && !directory) || (directory && (size !== 0 || mode === 0x8000))) throw artifactError("archive links or special file types are forbidden");
    validateExtra(bytes.subarray(position + 46 + nameLength, position + 46 + nameLength + extraLength));
    const components = relative.split("/");
    for (let component = 1; component < components.length; component++) {
      const parent = components.slice(0, component).join("/");
      const parentKey = parent.normalize("NFC").toLowerCase();
      const existing = paths.get(parentKey);
      if (existing && (!existing.directory || existing.relative !== parent)) throw artifactError("archive file and directory collision");
      if (!existing) paths.set(parentKey, { relative: parent, directory: true, implicit: true });
    }
    const key = relative.normalize("NFC").toLowerCase();
    const existing = paths.get(key);
    if (existing && !(existing.implicit && existing.relative === relative && directory)) throw artifactError("colliding archive entries");
    paths.set(key, { relative, directory, implicit: false });
    if (offset + 30 > directoryStart || bytes.readUInt32LE(offset) !== 0x04034b50 || bytes.readUInt16LE(offset + 6) !== flags || bytes.readUInt16LE(offset + 8) !== method) throw artifactError("archive local header mismatch");
    const localNameLength = bytes.readUInt16LE(offset + 26);
    const localExtraLength = bytes.readUInt16LE(offset + 28);
    const dataStart = offset + 30 + localNameLength + localExtraLength;
    const dataEnd = dataStart + compressed;
    if (dataEnd > directoryStart || !bytes.subarray(offset + 30, offset + 30 + localNameLength).equals(nameBytes)) throw artifactError("archive local filename or bounds mismatch");
    validateExtra(bytes.subarray(offset + 30 + localNameLength, dataStart));
    if (!(flags & 8) && (bytes.readUInt32LE(offset + 14) !== crc || bytes.readUInt32LE(offset + 18) !== compressed || bytes.readUInt32LE(offset + 22) !== size)) throw artifactError("archive local size or checksum mismatch");
    total += size;
    if (total > MAX_TOTAL) throw artifactError("archive exceeds extracted byte limit");
    entries.push({ relative, directory, method, crc, size, offset, dataStart, dataEnd });
    position = next;
  }
  if (position !== end) throw artifactError("archive directory size mismatch");
  const ordered = [...entries].sort((left, right) => left.offset - right.offset);
  for (let index = 1; index < ordered.length; index++) if (ordered[index].offset < ordered[index - 1].dataEnd) throw artifactError("overlapping archive entries");
  return entries;
}

function extractArtifactArchive(bytes, destination) {
  const entries = inspectArchive(bytes);
  const root = path.resolve(destination);
  if (fs.lstatSync(root, { throwIfNoEntry: false })) throw artifactError("archive destination must be new");
  const parts = path.relative(path.parse(root).root, path.dirname(root)).split(path.sep).filter(Boolean);
  let parent = path.parse(root).root;
  for (const part of parts) {
    parent = path.join(parent, part);
    const stat = fs.lstatSync(parent, { throwIfNoEntry: false });
    if (stat && (!stat.isDirectory() || stat.isSymbolicLink() || fs.realpathSync(parent) !== parent)) throw artifactError("unsafe archive destination");
    if (!stat) fs.mkdirSync(parent, { mode: 0o700 });
  }
  fs.mkdirSync(root, { mode: 0o700 });
  try {
    for (const entry of entries) {
      const filename = path.join(root, ...entry.relative.split("/"));
      if (!filename.startsWith(root + path.sep)) throw artifactError("archive entry escapes staging");
      if (entry.directory) {
        fs.mkdirSync(filename, { recursive: true, mode: 0o700 });
        continue;
      }
      const compressed = bytes.subarray(entry.dataStart, entry.dataEnd);
      const content = entry.method === 0 ? compressed : zlib.inflateRawSync(compressed, { maxOutputLength: Math.max(1, entry.size) });
      if (content.length !== entry.size || crc32(content) !== entry.crc) throw artifactError("archive entry checksum mismatch");
      fs.mkdirSync(path.dirname(filename), { recursive: true, mode: 0o700 });
      fs.writeFileSync(filename, content, { flag: "wx", mode: 0o600 });
    }
    checkedRoots.add(root);
    return root;
  } catch {
    fs.rmSync(root, { recursive: true, force: true });
    throw artifactError("artifact archive extraction failed");
  }
}

/** @param {any} [options] */
async function downloadArtifactValidationInput(options = {}) {
  const artifactId = options.artifactId || process.env.GH_AW_ARTIFACT_INPUT_ID;
  const artifactName = artifactId ? undefined : options.artifactName || process.env.GH_AW_ARTIFACT_INPUT_NAME || "agent";
  if (artifactId && !/^[1-9][0-9]{0,19}$/.test(String(artifactId))) throw artifactError("trusted artifact ID is invalid");
  try {
    const client = new DefaultArtifactClient();
    const url = await client.getArtifactArchiveURL({ artifactId, artifactName });
    const response = await fetch(url, { signal: AbortSignal.timeout(120000) });
    if (!response.ok || !response.body) throw artifactError("artifact archive download failed");
    if (Number(response.headers.get("content-length") || 0) > MAX_ARCHIVE) throw artifactError("artifact archive exceeds download limit");
    const reader = response.body.getReader();
    const chunks = [];
    let size = 0;
    try {
      for (;;) {
        const result = await reader.read();
        if (result.done) break;
        size += result.value.length;
        if (size > MAX_ARCHIVE) throw artifactError("artifact archive exceeds download limit");
        chunks.push(Buffer.from(result.value));
      }
    } finally {
      await reader.cancel();
    }
    return extractArtifactArchive(Buffer.concat(chunks), options.stagingRoot || process.env.GH_AW_ARTIFACT_STAGING_ROOT || "/tmp/gh-aw/artifact-validation/input");
  } catch (error) {
    if (error.code === "CTR-033") throw error;
    throw artifactError("safe artifact archive import failed");
  }
}

const hasCheckedArtifactImport = root => checkedRoots.has(path.resolve(root));

module.exports = { inspectArchive, extractArtifactArchive, downloadArtifactValidationInput, hasCheckedArtifactImport, crc32 };
