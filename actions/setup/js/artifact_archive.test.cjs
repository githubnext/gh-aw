import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import path from "path";
import { fileURLToPath, pathToFileURL } from "url";
import { createTestArchive } from "./test_artifact_archive.js";

const require = createRequire(import.meta.url);
const { inspectArchive, extractArtifactArchive, downloadArtifactValidationInput, hasCheckedArtifactImport } = require("./artifact_archive.cjs");
const { prepareArtifactValidation, downloadApprovedArtifact } = require("./artifact_validation.cjs");
let root, destination;
beforeEach(() => {
  root = fs.mkdtempSync(path.join(process.cwd(), ".artifact-archive-"));
  destination = path.join(root, "input");
});
afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  fs.rmSync(root, { recursive: true, force: true });
});

describe("checked artifact archive import", () => {
  it.each(["agent_output.json", "nested/download/agent_output.json"])("preserves actual ZIP layout and bytes (%s)", name => {
    const archive = createTestArchive([
      { name, content: "output", deflate: true, descriptor: true },
      { name: "nested/aw-owner-repo-feature.patch", content: "patch" },
    ]);
    extractArtifactArchive(archive, destination);
    expect(fs.readFileSync(path.join(destination, ...name.split("/")), "utf8")).toBe("output");
    expect(fs.readFileSync(path.join(destination, "nested", "aw-owner-repo-feature.patch"), "utf8")).toBe("patch");
    expect(hasCheckedArtifactImport(destination)).toBe(true);
  });
  it.each(["../outside", "nested/../../outside", "/absolute", "C:\\absolute", "\\\\server\\share", "a//b", "a/./b", "a\\..\\b", "a:stream", "nul.txt"])("rejects unsafe entry %s before any extraction writes", name => {
    const archive = createTestArchive([
      { name: "benign.txt", content: "benign" },
      { name, content: "other" },
    ]);
    expect(() => extractArtifactArchive(archive, destination)).toThrow("CTR-033: unsafe archive entry path");
    expect(fs.existsSync(destination)).toBe(false);
    expect(fs.existsSync(path.join(root, "outside"))).toBe(false);
  });
  it.each([
    ["duplicate.patch", "duplicate.patch"],
    ["UPPER.patch", "upper.patch"],
    ["é.patch", "e\u0301.patch"],
    ["folder", "folder/file"],
    ["folder/file", "folder"],
    ["FOLDER/", "folder/file"],
  ])("rejects collisions %s + %s without overwriting any file", (one, two) => {
    const archive = createTestArchive([
      { name: one, content: one.endsWith("/") ? "" : "one" },
      { name: two, content: "two" },
    ]);
    expect(() => extractArtifactArchive(archive, destination)).toThrow("CTR-033:");
    expect(fs.existsSync(destination)).toBe(false);
  });
  it("rejects symbolic links before extraction", () => {
    const archive = createTestArchive([{ name: "link", content: "../outside", mode: 0o120777 }]);
    expect(() => extractArtifactArchive(archive, destination)).toThrow("CTR-033: archive links or special file types are forbidden");
    expect(fs.existsSync(destination)).toBe(false);
  });
  it("rejects conflicting local and directory filenames", () => {
    const archive = createTestArchive([{ name: "safe", localName: "../outside", content: "data" }]);
    expect(() => extractArtifactArchive(archive, destination)).toThrow("CTR-033: archive local filename or bounds mismatch");
    expect(fs.existsSync(destination)).toBe(false);
  });
  it("rejects existing extraction destinations instead of overlaying files", () => {
    fs.mkdirSync(destination);
    fs.writeFileSync(path.join(destination, "original"), "original");
    expect(() => extractArtifactArchive(createTestArchive([{ name: "original", content: "replacement" }]), destination)).toThrow("CTR-033: archive destination must be new");
    expect(fs.readFileSync(path.join(destination, "original"), "utf8")).toBe("original");
  });
  it("refuses staging extracted by another downloader instead of trusting post-extraction checks", async () => {
    fs.mkdirSync(destination);
    fs.writeFileSync(path.join(destination, "agent_output.json"), JSON.stringify({ items: [] }));
    await expect(prepareArtifactValidation({ stagingRoot: destination, preparedRoot: path.join(root, "prepared"), approvedRoot: path.join(root, "approved") })).rejects.toThrow("CTR-033: artifact-validation requires checked archive import");
  });
  it("rejects destination junctions without writes outside staging", () => {
    const outside = path.join(root, "outside");
    fs.mkdirSync(outside);
    fs.symlinkSync(outside, path.join(root, "junction"), "junction");
    expect(() => extractArtifactArchive(createTestArchive([{ name: "file", content: "data" }]), path.join(root, "junction", "input"))).toThrow("CTR-033: unsafe archive destination");
    expect(fs.existsSync(path.join(outside, "input"))).toBe(false);
  });
  it("rejects corrupted bytes and removes only its own new extraction directory", () => {
    const archive = createTestArchive([{ name: "file", content: "data" }]);
    archive[34] ^= 1;
    expect(() => extractArtifactArchive(archive, destination)).toThrow("CTR-033: artifact archive extraction failed");
    expect(fs.existsSync(destination)).toBe(false);
  });
  it("accepts the artifact toolkit's normal archiver format", async () => {
    const archiver = require("archiver");
    const archive = archiver("zip", { zlib: { level: 6 } });
    const chunks = [];
    const completed = new Promise((resolve, reject) => {
      archive.on("data", chunk => chunks.push(chunk));
      archive.on("end", resolve);
      archive.on("error", reject);
    });
    archive.append(Buffer.from("output"), { name: "nested/agent_output.json" });
    await archive.finalize();
    await completed;
    extractArtifactArchive(Buffer.concat(chunks), destination);
    expect(fs.readFileSync(path.join(destination, "nested", "agent_output.json"), "utf8")).toBe("output");
  });
  it("accepts a ZIP generated by @actions/artifact createZipUploadStream with nested files and data descriptors", async () => {
    const uploadDirectory = path.join(path.dirname(fileURLToPath(import.meta.resolve("@actions/artifact"))), "internal", "upload");
    const { createZipUploadStream } = await import(pathToFileURL(path.join(uploadDirectory, "zip.js")).href);
    const { getUploadZipSpecification } = await import(pathToFileURL(path.join(uploadDirectory, "upload-zip-specification.js")).href);
    const producer = path.join(root, "producer");
    const files = new Map([
      ["README.txt", Buffer.from("synthetic artifact fixture\n")],
      [path.join("download", "agent_output.json"), Buffer.from('{"items":[]}\n')],
      [path.join("download", "patches", "aw-owner-repo-feature.patch"), Buffer.from("synthetic patch bytes\n")],
    ]);
    for (const [relative, bytes] of files) {
      const filename = path.join(producer, relative);
      fs.mkdirSync(path.dirname(filename), { recursive: true });
      fs.writeFileSync(filename, bytes);
    }
    const fetchMock = vi.fn().mockRejectedValue(new Error("archive compatibility test must not access a backend"));
    vi.stubGlobal("fetch", fetchMock);
    const specification = getUploadZipSpecification(
      [...files.keys()].map(relative => path.join(producer, relative)),
      producer
    );
    const stream = await createZipUploadStream(specification);
    const chunks = [];
    for await (const chunk of stream) chunks.push(Buffer.from(chunk));
    const archive = Buffer.concat(chunks);
    const entries = inspectArchive(archive);
    expect(entries.some(entry => !entry.directory && (archive.readUInt16LE(entry.offset + 6) & 8) !== 0)).toBe(true);
    expect(entries.map(entry => entry.relative).sort()).toEqual([...files.keys()].map(relative => relative.split(path.sep).join("/")).sort());
    extractArtifactArchive(archive, destination);
    for (const [relative, bytes] of files) expect(fs.readFileSync(path.join(destination, relative))).toEqual(bytes);
    expect(hasCheckedArtifactImport(destination)).toBe(true);
    expect(fetchMock).not.toHaveBeenCalled();
  });
  it("downloads raw bytes through same-run artifact transport without publication tokens", async () => {
    const payload = Buffer.from(JSON.stringify({ scp: "Actions.Results:run:reader" })).toString("base64url");
    vi.stubEnv("ACTIONS_RUNTIME_TOKEN", `header.${payload}.signature`);
    vi.stubEnv("ACTIONS_RESULTS_URL", "https://results.example/");
    const archive = createTestArchive([{ name: "nested/agent_output.json", content: "output" }]);
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (String(url).endsWith("/ListArtifacts")) {
        expect(JSON.parse(options.body)).toMatchObject({ workflowRunBackendId: "run", idFilter: "7" });
        return new Response(JSON.stringify({ artifacts: [{ workflowRunBackendId: "run", workflowJobRunBackendId: "producer", databaseId: "7", name: "agent" }] }), { status: 200 });
      }
      if (String(url).endsWith("/GetSignedArtifactURL")) return new Response(JSON.stringify({ signedUrl: "https://storage.example/archive.zip" }), { status: 200 });
      expect(options.headers).toBeUndefined();
      return new Response(archive, { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);
    await downloadArtifactValidationInput({ artifactId: "7", stagingRoot: destination });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fs.readFileSync(path.join(destination, "nested", "agent_output.json"), "utf8")).toBe("output");
  });
  it("requires a trusted artifact ID for approved-package import rather than selecting by name", async () => {
    vi.stubEnv("GH_AW_ARTIFACT_APPROVAL_ARTIFACT_ID", "");
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    await expect(downloadApprovedArtifact({ stagingRoot: destination, artifactName: "approved-artifacts" })).rejects.toThrow("CTR-033: trusted approved artifact ID");
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
