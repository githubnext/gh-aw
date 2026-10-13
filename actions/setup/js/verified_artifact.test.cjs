// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import crypto from "crypto";
import { execFileSync, spawnSync } from "child_process";
import { createRequire } from "module";
import { createTestArchive } from "./test_artifact_archive.js";

const require = createRequire(import.meta.url);
const { prepare, finalize, approvedTransport, assertApprovedTree, readApproval, protectArtifactGitExecution } = require("./verified_artifact.cjs");
const { resolveTransportPaths } = require("./resolve_transport_paths.cjs");
const { prepareArtifactValidation, finalizeArtifactValidation, initializeArtifactPublication, downloadArtifactValidationInput } = require("./artifact_validation.cjs");
const { extractArtifactArchive } = require("./artifact_archive.cjs");
let root, checkout, staging, approved, outputPath, configPath, message, config, options, patch;
const hash = bytes => crypto.createHash("sha256").update(bytes).digest("hex");
const git = (...args) => execFileSync("git", args, { cwd: checkout, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim();
const writeOutput = () => fs.writeFileSync(outputPath, JSON.stringify({ items: [message] }));
const checkedInput = () => {
  const entries = [];
  const collect = directory => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      const filename = path.join(directory, entry.name);
      if (entry.isDirectory()) collect(filename);
      else entries.push({ name: path.relative(staging, filename).split(path.sep).join("/"), content: fs.readFileSync(filename) });
    }
  };
  collect(staging);
  const archive = createTestArchive(entries);
  fs.rmSync(staging, { recursive: true, force: true });
  extractArtifactArchive(archive, staging);
};
const seal = async () => {
  const manifest = await prepare(options);
  const prepareDigest = hash(fs.readFileSync(path.join(approved + ".prepared", "manifest.json")));
  finalize({ ...options, prepareDigest });
  const manifestPath = path.join(approved, "manifest.json");
  process.env.GH_AW_ARTIFACT_APPROVAL_MANIFEST = manifestPath;
  process.env.GH_AW_ARTIFACT_APPROVAL_DIGEST = hash(fs.readFileSync(manifestPath));
  process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED = "true";
  process.env.GH_AW_AGENT_OUTPUT = path.join(approved, "agent_output.json");
  process.env.GH_AW_ARTIFACT_POLICY_FILE = configPath;
  return manifest;
};

beforeEach(() => {
  vi.stubEnv("GITHUB_REPOSITORY", "owner/repo");
  vi.stubEnv("GITHUB_RUN_ID", "123");
  vi.stubEnv("GITHUB_RUN_ATTEMPT", "1");
  vi.stubEnv("GITHUB_WORKFLOW_REF", "owner/repo/.github/workflows/test.yml@refs/heads/main");
  vi.stubEnv("GITHUB_SHA", "a".repeat(40));
  vi.stubEnv("GH_AW_WORKFLOW_ID", "artifact-test");
  for (const key of [
    "GH_AW_ARTIFACT_APPROVAL_REQUIRED",
    "GH_AW_ARTIFACT_APPROVAL_ROOT",
    "GH_AW_ARTIFACT_APPROVAL_MANIFEST",
    "GH_AW_ARTIFACT_APPROVAL_DIGEST",
    "GH_AW_ARTIFACT_PREPARED_ROOT",
    "GH_AW_ARTIFACT_APPROVED_ROOT",
    "GH_AW_ARTIFACT_PREPARE_DIGEST",
    "GH_AW_AGENT_OUTPUT",
    "GH_AW_ARTIFACT_POLICY_FILE",
    "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
    "GH_AW_WORK_QUEUE_ENABLED",
    "GH_AW_ARTIFACT_STAGING_ROOT",
    "GH_AW_PR_HEAD_BASE_SHA",
  ])
    vi.stubEnv(key, "");
  root = fs.mkdtempSync(path.join(process.cwd(), ".artifact-approval-"));
  checkout = path.join(root, "checkout");
  staging = path.join(root, "staging");
  approved = path.join(root, "approved");
  fs.mkdirSync(checkout);
  fs.mkdirSync(staging);
  git("init", "-b", "main");
  git("config", "user.email", "test@example.com");
  git("config", "user.name", "Test");
  fs.mkdirSync(path.join(checkout, "src"));
  fs.writeFileSync(path.join(checkout, "src", "allowed.txt"), "old\n");
  git("add", ".");
  git("commit", "-m", "base");
  git("checkout", "-b", "feature");
  fs.writeFileSync(path.join(checkout, "src", "allowed.txt"), "new\n");
  git("commit", "-am", "change");
  patch = execFileSync("git", ["format-patch", "--stdout", "main..HEAD"], { cwd: checkout });
  git("checkout", "main");
  fs.writeFileSync(path.join(staging, "aw-owner-repo-feature.patch"), patch);
  outputPath = path.join(root, "output.json");
  configPath = path.join(root, "policy.json");
  message = { type: "create_pull_request", branch: "feature", repo: "repo", title: "change" };
  config = { create_pull_request: { allowed_files: ["src/**"], base_branch: "main", patch_format: "am" } };
  fs.writeFileSync(configPath, JSON.stringify(config));
  writeOutput();
  options = { stagingRoot: staging, approvedRoot: approved, outputPath, configPath, checkout };
});
afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  fs.rmSync(root, { recursive: true, force: true });
});

describe("verified artifact approval", () => {
  it.each([false, true])("imports raw agent ZIP before exposing the validator contract (explicit download: %s)", async explicitDownload => {
    const archive = createTestArchive([
      { name: "nested/agent_output.json", content: fs.readFileSync(outputPath) },
      { name: "nested/aw-owner-repo-feature.patch", content: patch },
    ]);
    fs.rmSync(staging, { recursive: true, force: true });
    const token = Buffer.from(JSON.stringify({ scp: "Actions.Results:run:reader" })).toString("base64url");
    vi.stubEnv("ACTIONS_RUNTIME_TOKEN", `header.${token}.signature`);
    vi.stubEnv("ACTIONS_RESULTS_URL", "https://results.example/");
    vi.stubEnv("GH_AW_ARTIFACT_INPUT_ID", "");
    vi.stubEnv("GH_AW_ARTIFACT_INPUT_NAME", "validation-input");
    vi.stubGlobal(
      "fetch",
      vi.fn(async (url, request = {}) => {
        if (String(url).endsWith("/ListArtifacts")) {
          expect(JSON.parse(request.body)).toMatchObject({ nameFilter: "validation-input" });
          return new Response(JSON.stringify({ artifacts: [{ workflowRunBackendId: "run", workflowJobRunBackendId: "producer", databaseId: "7", name: "validation-input" }] }), { status: 200 });
        }
        if (String(url).endsWith("/GetSignedArtifactURL")) return new Response(JSON.stringify({ signedUrl: "https://storage.example/archive.zip" }), { status: 200 });
        return new Response(archive, { status: 200 });
      })
    );
    const exported = {};
    const coreApi = {
      exportVariable: (key, value) => {
        exported[key] = value;
      },
      setOutput: vi.fn(),
    };
    const invocation = { ...options, preparedRoot: path.join(root, "prepared"), core: coreApi, github: {}, context: { repo: { owner: "owner", repo: "repo" }, payload: {} } };
    if (explicitDownload) await downloadArtifactValidationInput(invocation);
    const descriptor = await prepareArtifactValidation(invocation);
    expect(descriptor.entries[0].identity.repo).toBe("owner/repo");
    expect(exported.GH_AW_ARTIFACT_MANIFEST).toBe(path.join(root, "prepared", "manifest.json"));
    expect(exported.GH_AW_AGENT_OUTPUT).toBe(path.join(root, "prepared", "agent_output.json"));
    vi.unstubAllGlobals();
  });
  it("supports first-class nested import, explicit prepared root and root-only publisher paths", async () => {
    const nested = path.join(staging, "download", "nested");
    fs.mkdirSync(nested, { recursive: true });
    fs.copyFileSync(outputPath, path.join(nested, "agent_output.json"));
    fs.renameSync(path.join(staging, "aw-owner-repo-feature.patch"), path.join(nested, "aw-owner-repo-feature.patch"));
    checkedInput();
    const preparedRoot = path.join(root, "prepared");
    const outputs = {};
    const previousCore = global.core;
    global.core = {
      setOutput: (key, value) => {
        outputs[key] = value;
      },
      exportVariable: (key, value) => {
        process.env[key] = value;
      },
      info: vi.fn(),
      debug: vi.fn(),
      warning: vi.fn(),
    };
    try {
      const coreApi = global.core;
      global.core = undefined;
      const descriptor = await prepareArtifactValidation({ ...options, preparedRoot, core: coreApi, context: { repo: { owner: "owner", repo: "repo" }, payload: {} } });
      global.core = coreApi;
      expect(descriptor.stateRoot).toBe(preparedRoot);
      expect(descriptor.entries[0].tree).toBeTruthy();
      expect(process.env.GH_AW_RESOLVED_ARTIFACT_MANIFEST).toBe(path.join(preparedRoot, "manifest.json"));
      expect(process.env.GH_AW_ARTIFACT_MANIFEST).toBe(path.join(preparedRoot, "manifest.json"));
      finalizeArtifactValidation({ ...options, preparedRoot, prepareDigest: outputs.prepare_digest, core: coreApi });
      process.env.GH_AW_ARTIFACT_APPROVAL_ROOT = approved;
      process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED = "true";
      process.env.GH_AW_ARTIFACT_APPROVAL_DIGEST = outputs.approval_digest;
      process.env.GH_AW_ARTIFACT_POLICY_FILE = configPath;
      process.env.GH_AW_AGENT_OUTPUT = outputPath;
      process.env.GH_AW_ARTIFACT_APPROVAL_MANIFEST = outputPath;
      const { loadAgentOutput } = require("./load_agent_output.cjs");
      expect(loadAgentOutput().items[0].branch).toBe("feature");
      expect(process.env.GH_AW_AGENT_OUTPUT).toBe(path.join(approved, "agent_output.json"));
      const paths = resolveTransportPaths(message, "owner/repo", { config: config.create_pull_request });
      expect(fs.readFileSync(paths.patchPath)).toEqual(patch);
      fs.appendFileSync(path.join(approved, "agent_output.json"), " ");
      expect(() => loadAgentOutput()).toThrow("CTR-033: approved output digest mismatch");
    } finally {
      global.core = previousCore;
    }
  });
  it("rejects duplicated nested agent output before creating a descriptor", async () => {
    for (const directory of ["one", "two"]) {
      fs.mkdirSync(path.join(staging, directory));
      fs.copyFileSync(outputPath, path.join(staging, directory, "agent_output.json"));
    }
    checkedInput();
    await expect(prepareArtifactValidation({ ...options, preparedRoot: path.join(root, "prepared") })).rejects.toThrow("CTR-033: downloaded agent output is missing or ambiguous");
  });
  it("explicitly rejects unsupported first-class cross-repository validation", async () => {
    message.repo = "other";
    config.create_pull_request.allowed_repos = ["owner/other"];
    fs.writeFileSync(configPath, JSON.stringify(config));
    writeOutput();
    fs.copyFileSync(outputPath, path.join(staging, "agent_output.json"));
    fs.renameSync(path.join(staging, "aw-owner-repo-feature.patch"), path.join(staging, "aw-owner-other-feature.patch"));
    checkedInput();
    await expect(prepareArtifactValidation({ ...options, preparedRoot: path.join(root, "prepared") })).rejects.toThrow("CTR-033: artifact-validation currently requires same-repository publication");
  });
  it.each(["other", "owner/other", "other/repo"])("rejects unauthorized dynamic repository %s under the workflow-only policy", async repo => {
    message.repo = repo;
    writeOutput();
    fs.copyFileSync(outputPath, path.join(staging, "agent_output.json"));
    checkedInput();
    const coreApi = { setOutput: vi.fn(), exportVariable: vi.fn() };
    await expect(prepareArtifactValidation({ ...options, preparedRoot: path.join(root, "prepared"), core: coreApi })).rejects.toThrow("CTR-033: artifact repository is not authorized");
    expect(coreApi.setOutput).not.toHaveBeenCalled();
    expect(coreApi.exportVariable).not.toHaveBeenCalled();
  });
  it.each(["repo", "owner/repo"])("prepares the triggering PR against its available trusted head with canonical repository %s", async repo => {
    const baseline = git("rev-parse", "main");
    message.type = "push_to_pull_request_branch";
    message.repo = repo;
    message.pull_number = 1;
    config = { push_to_pull_request_branch: { ...config.create_pull_request } };
    fs.writeFileSync(configPath, JSON.stringify(config));
    writeOutput();
    fs.copyFileSync(outputPath, path.join(staging, "agent_output.json"));
    checkedInput();
    const context = { payload: { pull_request: { number: 1, head: { sha: baseline, repo: { full_name: "owner/repo" } } } } };
    const descriptor = await prepareArtifactValidation({ ...options, preparedRoot: path.join(root, "prepared"), context });
    expect(descriptor.entries[0].identity.repo).toBe("owner/repo");
    expect(descriptor.entries[0].base).toBe(baseline);
    expect(descriptor.entries[0].tree).toBe(git("rev-parse", "feature^{tree}"));
  });
  it.each([
    ["wildcard", "supports only the triggering PR target"],
    ["fixed number", "supports only the triggering PR target"],
    ["missing context", "requires trusted triggering PR head context"],
    ["different number", "PR target differs from trusted context"],
    ["missing head object", "trusted PR head is unavailable locally"],
    ["different baseline", "PR baseline differs from trusted head"],
    ["base override", "PR base override differs from trusted head"],
    ["fork head", "requires same-repository PR heads"],
  ])("denies unsupported protected PR context without exporting a contract (%s)", async (scenario, reason) => {
    const baseline = git("rev-parse", "main");
    message.type = "push_to_pull_request_branch";
    message.pull_number = 1;
    config = { push_to_pull_request_branch: { ...config.create_pull_request } };
    let context = { payload: { pull_request: { number: 1, head: { sha: baseline, repo: { full_name: "owner/repo" } } } } };
    if (scenario === "wildcard") config.push_to_pull_request_branch.target = "*";
    if (scenario === "fixed number") config.push_to_pull_request_branch.target = "1";
    if (scenario === "missing context") context = undefined;
    if (scenario === "different number") message.pull_request_number = 2;
    if (scenario === "missing head object") context.payload.pull_request.head.sha = "b".repeat(40);
    if (scenario === "different baseline") vi.stubEnv("GH_AW_PR_HEAD_BASE_SHA", git("rev-parse", "feature"));
    if (scenario === "base override") message.base = "feature";
    if (scenario === "fork head") context.payload.pull_request.head.repo.full_name = "other/repo";
    fs.writeFileSync(configPath, JSON.stringify(config));
    writeOutput();
    fs.copyFileSync(outputPath, path.join(staging, "agent_output.json"));
    checkedInput();
    const coreApi = { setOutput: vi.fn(), exportVariable: vi.fn() };
    await expect(prepareArtifactValidation({ ...options, preparedRoot: path.join(root, "prepared"), context, core: coreApi })).rejects.toMatchObject({ code: "CTR-033", message: expect.stringContaining(reason) });
    expect(coreApi.setOutput).not.toHaveBeenCalled();
    expect(coreApi.exportVariable).not.toHaveBeenCalled();
  });
  it("explicitly rejects unsupported first-class claim validation", async () => {
    message.claim_handle = "one";
    writeOutput();
    fs.copyFileSync(outputPath, path.join(staging, "agent_output.json"));
    checkedInput();
    await expect(prepareArtifactValidation({ ...options, preparedRoot: path.join(root, "prepared") })).rejects.toThrow("CTR-033: artifact-validation does not support claim-scoped publication");
  });
  it("rejects output file references that are not part of the sealed artifact contract", async () => {
    message.filePath = "publisher-only-data.txt";
    writeOutput();
    fs.copyFileSync(outputPath, path.join(staging, "agent_output.json"));
    checkedInput();
    await expect(prepareArtifactValidation({ ...options, preparedRoot: path.join(root, "prepared") })).rejects.toThrow("CTR-033: artifact-validation does not support unbound content file references");
  });
  it("reports malformed output without disclosing artifact contents", async () => {
    fs.writeFileSync(path.join(staging, "agent_output.json"), "DO_NOT_LOG_ARTIFACT_CONTENTS");
    checkedInput();
    try {
      await prepareArtifactValidation({ ...options, preparedRoot: path.join(root, "prepared") });
      expect.fail("malformed output must be rejected");
    } catch (error) {
      expect(error.message).toBe("CTR-033: agent output is not valid UTF-8 JSON");
      expect(error.message).not.toContain("DO_NOT_LOG_ARTIFACT_CONTENTS");
    }
  });
  it.each(["", "downloaded"])("validates and publishes the same immutable bytes (%s layout)", async nesting => {
    if (nesting) {
      fs.mkdirSync(path.join(staging, nesting));
      fs.renameSync(path.join(staging, "aw-owner-repo-feature.patch"), path.join(staging, nesting, "aw-owner-repo-feature.patch"));
    }
    const prepared = await seal();
    // Replacing the original staging file cannot alter the approved payload.
    fs.writeFileSync(path.join(staging, nesting, "aw-owner-repo-feature.patch"), "alternate payload");
    const paths = resolveTransportPaths(message, "owner/repo", { config: config.create_pull_request });
    expect(fs.readFileSync(paths.patchPath)).toEqual(patch);
    git("checkout", "feature");
    expect(() => assertApprovedTree({ cwd: checkout, baseRef: "main", repo: "owner/repo" })).not.toThrow();
    expect(prepared.entries[0].identity.repo).toBe("owner/repo");
  });
  it.each(["", "nested"])("combined alias regression denies approval before remote writes (%s)", async nesting => {
    git("checkout", "-b", "alternate", "main");
    fs.writeFileSync(path.join(checkout, "forbidden.txt"), "policy-violating change\n");
    git("add", "forbidden.txt");
    git("commit", "-m", "alternate policy-violating patch");
    const alternate = execFileSync("git", ["format-patch", "--stdout", "main..alternate"], { cwd: checkout });
    git("checkout", "main");
    fs.mkdirSync(path.join(staging, nesting), { recursive: true });
    if (nesting) fs.renameSync(path.join(staging, "aw-owner-repo-feature.patch"), path.join(staging, nesting, "aw-owner-repo-feature.patch"));
    fs.writeFileSync(path.join(staging, nesting, "aw-repo-feature.patch"), alternate);
    fs.copyFileSync(outputPath, path.join(staging, nesting, "agent_output.json"));
    checkedInput();
    expect(message.repo).toBe("repo");
    expect(alternate.equals(patch)).toBe(false);
    const githubClient = { rest: { git: { updateRef: vi.fn() }, pulls: { create: vi.fn() } } };
    const coreApi = { setOutput: vi.fn(), exportVariable: vi.fn() };
    await expect(
      prepareArtifactValidation({ ...options, preparedRoot: path.join(root, "prepared"), core: coreApi, github: githubClient })
        .then(() => githubClient.rest.git.updateRef())
        .then(() => githubClient.rest.pulls.create())
    ).rejects.toThrow("CTR-033: ambiguous");
    expect(githubClient.rest.git.updateRef).not.toHaveBeenCalled();
    expect(githubClient.rest.pulls.create).not.toHaveBeenCalled();
    expect(coreApi.setOutput).not.toHaveBeenCalled();
    expect(coreApi.exportVariable).not.toHaveBeenCalled();
    expect(fs.existsSync(approved)).toBe(false);
  });
  it("denies replacement of imported bytes between validation and finalization", async () => {
    const manifest = await prepare(options);
    const state = approved + ".prepared";
    const prepareDigest = hash(fs.readFileSync(path.join(state, "manifest.json")));
    const payload = path.join(state, manifest.entries[0].artifacts[0].path);
    fs.chmodSync(payload, 0o600);
    fs.writeFileSync(payload, "changed");
    expect(() => finalize({ ...options, prepareDigest })).toThrow("CTR-033: artifact changed during validation");
  });
  it("denies a validator rewriting its own approval metadata", async () => {
    await prepare(options);
    const statePath = path.join(approved + ".prepared", "manifest.json");
    const prepareDigest = hash(fs.readFileSync(statePath));
    const state = JSON.parse(fs.readFileSync(statePath, "utf8"));
    state.entries[0].tree = "f".repeat(40);
    fs.writeFileSync(statePath, JSON.stringify(state));
    expect(() => finalize({ ...options, prepareDigest })).toThrow("CTR-033: trusted preparation digest");
  });
  it("denies modified output between preparation and finalization", async () => {
    await prepare(options);
    const prepareDigest = hash(fs.readFileSync(path.join(approved + ".prepared", "manifest.json")));
    message.branch = "another";
    writeOutput();
    expect(() => finalize({ ...options, prepareDigest })).toThrow("CTR-033: output changed during validation");
  });
  it("denies replaced approved bytes before signed or direct publication", async () => {
    await seal();
    const paths = approvedTransport(message, "owner/repo", { config: config.create_pull_request });
    fs.chmodSync(paths.patchPath, 0o600);
    fs.writeFileSync(paths.patchPath, "replaced");
    const mutate = vi.fn();
    git("checkout", "feature");
    expect(() => {
      assertApprovedTree({ cwd: checkout, baseRef: "main", repo: "owner/repo" });
      mutate();
    }).toThrow("CTR-033: approved artifact replaced");
    expect(mutate).not.toHaveBeenCalled();
  });
  it.each(["GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT", "GITHUB_WORKFLOW_REF", "GITHUB_SHA"])("denies approval replay with changed %s", async key => {
    await seal();
    process.env[key] = "changed";
    expect(() => readApproval()).toThrow("CTR-033: approval run or policy mismatch");
  });
  it("binds approval to the explicitly selected trusted checkout policy revision", async () => {
    vi.stubEnv("GH_AW_ARTIFACT_POLICY_REF", git("rev-parse", "main"));
    await seal();
    expect(() => readApproval()).not.toThrow();
    vi.stubEnv("GH_AW_ARTIFACT_POLICY_REF", git("rev-parse", "feature"));
    expect(() => readApproval()).toThrow("CTR-033: approval run or policy mismatch");
  });
  it("matches credential-free preparation to publisher policy despite per-handler token rotation", async () => {
    config.push_to_pull_request_branch = { ...config.create_pull_request };
    fs.writeFileSync(configPath, JSON.stringify(config));
    vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify(config));
    const manifest = await seal();
    const preparedPolicy = fs.readFileSync(path.join(approved + ".prepared", "config.json"), "utf8");
    expect(preparedPolicy).not.toMatch(/github-token|head-github-token/);
    expect(JSON.stringify(manifest.entries[0].policy)).not.toMatch(/github-token|head-github-token/);
    vi.stubEnv("GH_AW_ARTIFACT_APPROVAL_ROOT", approved);
    const coreApi = { setOutput: vi.fn(), exportVariable: vi.fn() };
    expect(() => initializeArtifactPublication({ core: coreApi })).not.toThrow();
    for (const generation of ["first", "rotated"]) {
      const publisherConfig = Object.fromEntries(
        Object.entries(config).map(([type, policy]) => [
          type,
          {
            ...policy,
            "github-token": `synthetic-${generation}-${type}-token`,
            "head-github-token": `synthetic-${generation}-${type}-head-token`,
          },
        ])
      );
      vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify(publisherConfig));
      expect(readApproval().manifest.binding.policy).toBe(manifest.binding.policy);
      expect(() => approvedTransport(message, "owner/repo", { config: publisherConfig.create_pull_request })).not.toThrow();
    }
  });
  it.each([
    ["allowed_files", ["docs/**"]],
    ["base_branch", "another-base"],
    ["patch_format", "bundle"],
    ["signed_commits", false],
  ])("does not exclude protection field %s alongside publisher credentials", async (field, changed) => {
    await seal();
    const publisherPolicy = { ...config.create_pull_request, "github-token": "synthetic-main-token", "head-github-token": "synthetic-head-token" };
    vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify({ create_pull_request: publisherPolicy }));
    expect(() => readApproval()).not.toThrow();
    expect(() => approvedTransport(message, "owner/repo", { config: { ...publisherPolicy, [field]: changed } })).toThrow("CTR-033: approved handler policy mismatch");
    vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify({ create_pull_request: { ...publisherPolicy, [field]: changed } }));
    expect(() => readApproval()).toThrow("CTR-033: approval run or policy mismatch");
  });
  it("denies forged or tampered approval without trusted digest", async () => {
    await seal();
    fs.appendFileSync(process.env.GH_AW_ARTIFACT_APPROVAL_MANIFEST, " ");
    expect(() => approvedTransport(message, "owner/repo", { config: config.create_pull_request })).toThrow("CTR-033: approval digest mismatch");
  });
  it("initializes protected handler output before checkout using only the approved package and serialized policy", async () => {
    await seal();
    fs.rmSync(checkout, { recursive: true, force: true });
    vi.stubEnv("GH_AW_ARTIFACT_APPROVAL_ROOT", approved);
    vi.stubEnv("GH_AW_ARTIFACT_POLICY_FILE", "");
    vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify(config));
    vi.stubEnv("GH_AW_AGENT_OUTPUT", outputPath);
    vi.stubEnv("GH_AW_ARTIFACT_APPROVAL_MANIFEST", outputPath);
    const coreApi = { setOutput: vi.fn(), exportVariable: vi.fn() };
    const githubClient = { graphql: vi.fn(), rest: { git: { createRef: vi.fn() } } };
    const manifest = initializeArtifactPublication({ core: coreApi, github: githubClient, context: { repo: { owner: "owner", repo: "repo" }, payload: {} } });
    expect(manifest.entries[0].identity.repo).toBe("owner/repo");
    expect(coreApi.setOutput).toHaveBeenCalledExactlyOnceWith("GH_AW_AGENT_OUTPUT", path.join(approved, "agent_output.json"));
    expect(coreApi.exportVariable).toHaveBeenCalledWith("GH_AW_AGENT_OUTPUT", path.join(approved, "agent_output.json"));
    for (const name of ["GH_AW_ARTIFACT_MANIFEST", "GH_AW_RESOLVED_ARTIFACT_MANIFEST", "GH_AW_ARTIFACT_APPROVAL_MANIFEST"]) expect(coreApi.exportVariable).toHaveBeenCalledWith(name, path.join(approved, "manifest.json"));
    expect(githubClient.graphql).not.toHaveBeenCalled();
    expect(githubClient.rest.git.createRef).not.toHaveBeenCalled();
  });
  it.each(["manifest", "output", "artifact", "policy", "run", "missing digest", "transport", "path"])("denies invalid publication initialization without exposing handler paths (%s)", async tamper => {
    const manifest = await seal();
    vi.stubEnv("GH_AW_ARTIFACT_APPROVAL_ROOT", approved);
    const manifestPath = path.join(approved, "manifest.json");
    if (tamper === "manifest") fs.appendFileSync(manifestPath, " ");
    if (tamper === "output") fs.appendFileSync(path.join(approved, "agent_output.json"), " ");
    if (tamper === "artifact") {
      const artifactPath = path.join(approved, manifest.entries[0].artifacts[0].path);
      fs.chmodSync(artifactPath, 0o600);
      fs.appendFileSync(artifactPath, " ");
    }
    if (tamper === "policy") vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify({ create_pull_request: { allowed_files: ["**"] } }));
    if (tamper === "run") vi.stubEnv("GITHUB_RUN_ATTEMPT", "2");
    if (tamper === "missing digest") vi.stubEnv("GH_AW_ARTIFACT_APPROVAL_DIGEST", "");
    if (tamper === "transport" || tamper === "path") {
      if (tamper === "transport") manifest.entries[0].artifacts[0].transport = "unknown";
      else manifest.entries[0].artifacts[0].path = "../outside.patch";
      fs.writeFileSync(manifestPath, JSON.stringify(manifest));
      vi.stubEnv("GH_AW_ARTIFACT_APPROVAL_DIGEST", hash(fs.readFileSync(manifestPath)));
    }
    const coreApi = { setOutput: vi.fn(), exportVariable: vi.fn() };
    expect(() => initializeArtifactPublication({ core: coreApi, github: {}, context: {} })).toThrow("CTR-033:");
    expect(coreApi.setOutput).not.toHaveBeenCalled();
    expect(coreApi.exportVariable).not.toHaveBeenCalled();
  });
  it.each(["mode", "root"])("requires protected publication configuration (%s)", async missing => {
    await seal();
    vi.stubEnv("GH_AW_ARTIFACT_APPROVAL_ROOT", missing === "root" ? "" : approved);
    if (missing === "mode") vi.stubEnv("GH_AW_ARTIFACT_APPROVAL_REQUIRED", "false");
    const coreApi = { setOutput: vi.fn(), exportVariable: vi.fn() };
    expect(() => initializeArtifactPublication({ core: coreApi })).toThrow("CTR-033: protected publication requires an approved artifact root");
    expect(coreApi.setOutput).not.toHaveBeenCalled();
    expect(coreApi.exportVariable).not.toHaveBeenCalled();
  });
  it("denies missing approval before signed and unsigned push APIs", async () => {
    process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED = "true";
    const { pushSignedCommits } = require("./push_signed_commits.cjs");
    const githubClient = { graphql: vi.fn(), rest: { git: { createRef: vi.fn() } } };
    for (const signedCommits of [true, false])
      await expect(pushSignedCommits({ githubClient, owner: "owner", repo: "repo", branch: "feature", baseRef: "main", cwd: checkout, signedCommits, artifactApprovalRequired: true })).rejects.toThrow("CTR-033");
    expect(githubClient.graphql).not.toHaveBeenCalled();
    expect(githubClient.rest.git.createRef).not.toHaveBeenCalled();
  });
  it("denies changed resulting tree and changed base before any mutation", async () => {
    await seal();
    approvedTransport(message, "owner/repo", { config: config.create_pull_request });
    git("checkout", "feature");
    expect(() => assertApprovedTree({ cwd: checkout, baseRef: "feature", repo: "owner/repo" })).toThrow("CTR-033: publication base");
    fs.writeFileSync(path.join(checkout, "src", "allowed.txt"), "another\n");
    git("commit", "-am", "another change");
    expect(() => assertApprovedTree({ cwd: checkout, baseRef: "main", repo: "owner/repo" })).toThrow("CTR-033: publication base or resulting tree");
  });
  it("checks allowed-files against the actual applied patch tree", async () => {
    config.create_pull_request.allowed_files = ["docs/**"];
    fs.writeFileSync(configPath, JSON.stringify(config));
    await expect(prepare(options)).rejects.toThrow("CTR-033: resulting tree violates file policy");
  });
  it("checks bundle resulting tree even when its patch preview is benign", async () => {
    git("checkout", "feature");
    fs.writeFileSync(path.join(checkout, "other.txt"), "not allowed\n");
    git("add", "other.txt");
    git("commit", "-m", "bundle-only forbidden change");
    git("bundle", "create", path.join(staging, "aw-owner-repo-feature.bundle"), "main..feature");
    git("checkout", "main");
    config.create_pull_request.patch_format = "bundle";
    fs.writeFileSync(configPath, JSON.stringify(config));
    await expect(prepare(options)).rejects.toThrow("CTR-033: resulting tree violates file policy");
  });
  it("preserves legitimate bundle transport and resulting tree approval", async () => {
    git("bundle", "create", path.join(staging, "aw-owner-repo-feature.bundle"), "main..feature");
    config.create_pull_request.patch_format = "bundle";
    fs.writeFileSync(configPath, JSON.stringify(config));
    await seal();
    const paths = approvedTransport(message, "owner/repo", { config: config.create_pull_request });
    expect(paths.bundlePath).toBeTruthy();
    git("checkout", "feature");
    expect(() => assertApprovedTree({ cwd: checkout, baseRef: "main", repo: "owner/repo" })).not.toThrow();
  });
  it("preserves an approved empty-commit base tree", async () => {
    config.create_pull_request.allow_empty = true;
    fs.writeFileSync(configPath, JSON.stringify(config));
    fs.writeFileSync(path.join(staging, "aw-owner-repo-feature.patch"), "");
    await seal();
    approvedTransport(message, "owner/repo", { config: config.create_pull_request });
    git("commit", "--allow-empty", "-m", "empty");
    // The baseline remains the original parent, not the newly advanced main ref.
    expect(() => assertApprovedTree({ cwd: checkout, baseRef: "HEAD^", repo: "owner/repo" })).not.toThrow();
  });
  it("validates PR-branch publication against its trusted head rather than the configured base branch", async () => {
    git("checkout", "-b", "pr-head", "main");
    fs.writeFileSync(path.join(checkout, "src", "allowed.txt"), "head\n");
    git("commit", "-am", "PR head baseline");
    const baseline = git("rev-parse", "HEAD");
    git("checkout", "feature");
    git("reset", "--hard", "pr-head");
    fs.writeFileSync(path.join(checkout, "src", "allowed.txt"), "new\n");
    git("commit", "-am", "PR head update");
    patch = execFileSync("git", ["format-patch", "--stdout", "pr-head..feature"], { cwd: checkout });
    fs.writeFileSync(path.join(staging, "aw-owner-repo-feature.patch"), patch);
    git("checkout", "main");
    vi.stubEnv("GH_AW_PR_HEAD_BASE_SHA", baseline);
    message.type = "push_to_pull_request_branch";
    message.pull_number = 1;
    config = { push_to_pull_request_branch: { base_branch: "main", allowed_files: ["src/**"], patch_format: "am" } };
    fs.writeFileSync(configPath, JSON.stringify(config));
    writeOutput();
    const descriptor = await seal();
    expect(descriptor.entries[0].base).toBe(baseline);
    approvedTransport(message, "owner/repo", { config: config.push_to_pull_request_branch });
    git("checkout", "feature");
    expect(() => assertApprovedTree({ cwd: checkout, baseRef: baseline, repo: "owner/repo" })).not.toThrow();
  });
  it("keeps application credential-free and prevents repository hook execution", async () => {
    process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED = "true";
    const executor = { exec: vi.fn().mockResolvedValue(0), getExecOutput: vi.fn() };
    const original = executor.exec;
    protectArtifactGitExecution(executor);
    await executor.exec("git", ["am", "--3way", "approved.patch"], { cwd: checkout, env: { GITHUB_TOKEN: "secret", SSH_AUTH_SOCK: "socket", GIT_CONFIG_COUNT: "1", GIT_CONFIG_KEY_0: "http.extraheader", GIT_CONFIG_VALUE_0: "secret" } });
    const [, args, opts] = original.mock.calls[0];
    expect(args).toContain("credential.helper=");
    expect(args.some(arg => String(arg).includes("core.hooksPath=") && String(arg).endsWith("verified_artifact.cjs"))).toBe(true);
    expect(opts.env.GITHUB_TOKEN).toBeUndefined();
    expect(opts.env.SSH_AUTH_SOCK).toBeUndefined();
    expect(opts.env.GIT_CONFIG_COUNT).toBeUndefined();
    expect(opts.env.GIT_NO_LAZY_FETCH).toBe("1");
  });
  it.each([
    ["am", true],
    ["am", false],
    ["bundle", true],
    ["bundle", false],
  ])("preserves approved %s publication with signed-commits=%s", async (format, signedCommits) => {
    const remote = path.join(root, "remote.git");
    execFileSync("git", ["init", "--bare", remote], { stdio: "pipe" });
    git("remote", "add", "origin", remote);
    git("push", "origin", "main");
    config.create_pull_request.patch_format = format;
    config.create_pull_request.signed_commits = signedCommits;
    fs.writeFileSync(configPath, JSON.stringify(config));
    if (format === "bundle") git("bundle", "create", path.join(staging, "aw-owner-repo-feature.bundle"), "main..feature");
    const manifest = await seal();
    approvedTransport(message, "owner/repo", { config: config.create_pull_request });
    git("checkout", "feature");
    const head = git("rev-parse", "HEAD");
    const saved = { core: global.core, exec: global.exec };
    global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn() };
    const getExecOutput = async (command, args, opts = {}) => {
      const result = spawnSync(command, args, { cwd: opts.cwd || checkout, env: opts.env || process.env, encoding: "utf8" });
      if (result.status !== 0 && !opts.ignoreReturnCode) throw new Error(result.stderr || "git failed");
      return { exitCode: result.status, stdout: result.stdout || "", stderr: result.stderr || "" };
    };
    global.exec = { getExecOutput, exec: async (command, args, opts) => (await getExecOutput(command, args, opts)).exitCode };
    const githubClient = {
      rest: {
        git: {
          createRef: vi.fn(async ({ ref, sha }) => {
            execFileSync("git", [`--git-dir=${remote}`, "update-ref", ref, sha], { stdio: "pipe" });
          }),
        },
      },
      graphql: vi.fn(async (_query, { input }) => {
        expect(input.fileChanges.additions).toEqual([{ path: "src/allowed.txt", contents: Buffer.from("new\n").toString("base64") }]);
        expect(input.fileChanges.deletions).toEqual([]);
        execFileSync("git", [`--git-dir=${remote}`, "fetch", checkout, "feature"], { stdio: "pipe" });
        execFileSync("git", [`--git-dir=${remote}`, "update-ref", "refs/heads/feature", head], { stdio: "pipe" });
        return { createCommitOnBranch: { commit: { oid: head } } };
      }),
    };
    try {
      const { pushSignedCommits } = require("./push_signed_commits.cjs");
      const result = await pushSignedCommits({ githubClient, owner: "owner", repo: "repo", branch: "feature", baseRef: "main", cwd: checkout, signedCommits, validationConfig: config.create_pull_request, artifactApprovalRequired: true });
      expect(result).toBe(head);
      const tree = execFileSync("git", [`--git-dir=${remote}`, "rev-parse", "feature^{tree}"], { encoding: "utf8" }).trim();
      expect(tree).toBe(manifest.entries[0].tree);
      expect(githubClient.graphql).toHaveBeenCalledTimes(signedCommits ? 1 : 0);
    } finally {
      Object.assign(global, saved);
    }
  });
  it.each([
    ["create_pull_request", "approval"],
    ["create_pull_request", "base"],
    ["create_pull_request", "tree"],
    ["create_pull_request", "empty branch"],
    ["push_to_pull_request_branch", "approval"],
    ["push_to_pull_request_branch", "base"],
    ["push_to_pull_request_branch", "tree"],
    ["push_to_pull_request_branch", "review tree"],
  ])("%s never mutates fallback or CI surfaces after initialized approval fails (%s)", async (type, failure) => {
    const saved = { core: global.core, github: global.github, context: global.context, exec: global.exec, getOctokit: global.getOctokit };
    const originalCwd = process.cwd();
    const baseline = git("rev-parse", "main");
    message.type = type;
    message.pull_number = 1;
    const handlerConfig = { ...config.create_pull_request, signed_commits: false, check_branch_protection: false };
    if (failure === "empty branch") {
      handlerConfig.allow_empty = true;
      fs.writeFileSync(path.join(staging, "aw-owner-repo-feature.patch"), "");
    }
    config = { [type]: handlerConfig };
    fs.writeFileSync(configPath, JSON.stringify(config));
    writeOutput();
    vi.stubEnv("GH_AW_PR_HEAD_BASE_SHA", baseline);
    vi.stubEnv("GITHUB_REPOSITORY_ID", "123");
    vi.stubEnv("GITHUB_JOB", "safe_outputs");
    vi.stubEnv("GH_AW_JOB_CHECK_RUN_ID", "456");
    await seal();
    vi.stubEnv("GH_AW_ARTIFACT_APPROVAL_ROOT", approved);
    vi.stubEnv("GH_AW_CI_TRIGGER_TOKEN", "synthetic-ci-token");
    vi.stubEnv("GH_AW_PROMPTS_DIR", path.resolve(originalCwd, "..", "md"));
    const mutation = () => vi.fn().mockRejectedValue(new Error("unexpected remote mutation"));
    const mutations = {
      pulls: { create: mutation(), update: mutation(), createReview: mutation(), requestReviewers: mutation() },
      issues: { create: mutation(), update: mutation(), createComment: mutation(), updateComment: mutation(), addLabels: mutation(), addAssignees: mutation() },
      git: { createRef: mutation(), deleteRef: mutation(), updateRef: mutation(), createCommit: mutation(), createTree: mutation() },
    };
    const coreApi = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
      setOutput: vi.fn(),
      exportVariable: vi.fn(),
      startGroup: vi.fn(),
      endGroup: vi.fn(),
      summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue(undefined) },
    };
    global.core = coreApi;
    initializeArtifactPublication({ core: coreApi });
    git("checkout", "feature");
    if (failure === "tree" || failure === "empty branch" || failure === "review tree") {
      fs.writeFileSync(path.join(checkout, "src", "allowed.txt"), "changed after initialization\n");
      git("commit", "-am", "post-initialization tree change");
    }
    if (failure === "approval") fs.appendFileSync(path.join(approved, "manifest.json"), " ");
    if (failure === "review tree") vi.stubEnv("GH_AW_DETECTION_CONCLUSION", "warning");
    const remoteHead = failure === "base" ? git("rev-parse", "feature") : baseline;
    git("update-ref", "refs/remotes/origin/main", remoteHead);
    git("update-ref", "refs/remotes/origin/feature", remoteHead);
    const pull = {
      number: 1,
      title: "change",
      labels: [],
      head: { ref: "feature", sha: remoteHead, repo: { full_name: "owner/repo", owner: { login: "owner" }, name: "repo" } },
      base: { ref: "main", repo: { full_name: "owner/repo", owner: { login: "owner" }, name: "repo" } },
    };
    global.github = {
      rest: {
        pulls: { ...mutations.pulls, get: vi.fn().mockResolvedValue({ data: pull }), list: vi.fn().mockResolvedValue({ data: [] }) },
        issues: mutations.issues,
        git: { ...mutations.git, getRef: vi.fn().mockResolvedValue({ data: { object: { sha: remoteHead } } }) },
        repos: { get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }), getBranch: vi.fn().mockResolvedValue({ data: { protected: false, commit: { sha: remoteHead } } }) },
      },
      graphql: mutation(),
    };
    global.getOctokit = vi.fn().mockReturnValue(global.github);
    global.context = { repo: { owner: "owner", repo: "repo" }, eventName: "pull_request", payload: { pull_request: { ...pull, head: { ...pull.head, sha: baseline } } } };
    const execute = vi.fn().mockResolvedValue(0);
    global.exec = {
      exec: execute,
      getExecOutput: vi.fn(async (_command, args) => {
        let stdout = "";
        if (args.includes("rev-parse")) stdout = args.includes("--show-toplevel") ? checkout : remoteHead;
        if (args.includes("rev-list")) stdout = "1";
        if (args.includes("diff") && args.includes("--name-only")) stdout = args.includes("-z") ? "src/allowed.txt\0" : "src/allowed.txt\n";
        return { exitCode: 0, stdout, stderr: "" };
      }),
    };
    const extra = vi.spyOn(require("./extra_empty_commit.cjs"), "pushExtraEmptyCommit").mockResolvedValue({ success: true, skipped: false });
    const approvalApi = require("./verified_artifact.cjs");
    const boundary = vi.spyOn(approvalApi, "assertApprovedTree");
    delete require.cache[require.resolve(`./${type}.cjs`)];
    try {
      process.chdir(checkout);
      const handler = await require(`./${type}.cjs`).main(handlerConfig);
      if (failure === "approval") await expect(handler(message, {})).resolves.toMatchObject({ success: false, code: "CTR-033", error: expect.stringContaining("approval digest mismatch") });
      else await expect(handler(message, {})).rejects.toMatchObject({ code: "CTR-033", message: expect.stringContaining("publication base or resulting tree") });
      if ((type === "create_pull_request" && failure !== "approval") || failure === "review tree") expect(boundary).toHaveBeenCalled();
      for (const group of Object.values(mutations)) for (const api of Object.values(group)) expect(api).not.toHaveBeenCalled();
      expect(global.github.graphql).not.toHaveBeenCalled();
      expect(extra).not.toHaveBeenCalled();
      expect(execute.mock.calls.some(([, args]) => args.includes("push"))).toBe(false);
    } finally {
      process.chdir(originalCwd);
      extra.mockRestore();
      boundary.mockRestore();
      delete require.cache[require.resolve(`./${type}.cjs`)];
      Object.assign(global, saved);
    }
  });
  it.each([
    ["create_pull_request", null],
    ["push_to_pull_request_branch", null],
    ["create_pull_request", ""],
    ["push_to_pull_request_branch", ""],
    ["create_pull_request", "nested"],
    ["push_to_pull_request_branch", "nested"],
  ])("%s denies missing approval or conflicting aliases (%s) without remote mutations", async (type, nesting) => {
    const saved = { core: global.core, github: global.github, context: global.context, exec: global.exec };
    global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn(), setOutput: vi.fn() };
    global.github = { rest: { pulls: { create: vi.fn() }, git: { createRef: vi.fn(), deleteRef: vi.fn(), updateRef: vi.fn() }, issues: { create: vi.fn() } }, graphql: vi.fn() };
    global.context = { repo: { owner: "owner", repo: "repo" }, payload: {}, eventName: "workflow_dispatch" };
    const execSpy = vi.fn();
    global.exec = { exec: execSpy, getExecOutput: vi.fn() };
    process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED = "true";
    delete process.env.GH_AW_ARTIFACT_APPROVAL_MANIFEST;
    if (nesting !== null) {
      process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED = "";
      process.env.GH_AW_ARTIFACT_STAGING_ROOT = staging;
      git("checkout", "-b", "alternate", "main");
      fs.writeFileSync(path.join(checkout, "forbidden.txt"), "policy-violating change\n");
      git("add", "forbidden.txt");
      git("commit", "-m", "alternate policy-violating patch");
      const alternate = execFileSync("git", ["format-patch", "--stdout", "main..alternate"], { cwd: checkout });
      git("checkout", "main");
      fs.mkdirSync(path.join(staging, nesting), { recursive: true });
      fs.writeFileSync(path.join(staging, nesting, "aw-repo-feature.patch"), alternate);
    }
    try {
      const { main } = require(`./${type}.cjs`);
      const handler = await main({ base_branch: "main", allowed_files: ["src/**"] });
      const result = await handler({ ...message, type }, {});
      expect(result).toMatchObject({ success: false, code: "CTR-033" });
      expect(global.github.rest.pulls.create).not.toHaveBeenCalled();
      expect(global.github.rest.git.createRef).not.toHaveBeenCalled();
      expect(global.github.rest.git.deleteRef).not.toHaveBeenCalled();
      expect(global.github.rest.git.updateRef).not.toHaveBeenCalled();
      expect(global.github.rest.issues.create).not.toHaveBeenCalled();
      expect(global.github.graphql).not.toHaveBeenCalled();
      expect(execSpy).not.toHaveBeenCalled();
    } finally {
      Object.assign(global, saved);
    }
  });
});
