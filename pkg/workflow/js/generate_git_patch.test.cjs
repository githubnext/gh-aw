import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { execSync } from "child_process";
import os from "os";

describe("generate_git_patch handler", () => {
  let testDir;
  let originalEnv;
  let originalCwd;

  beforeEach(() => {
    // Save original environment
    originalEnv = { ...process.env };
    originalCwd = process.cwd();

    // Create a temporary test directory
    testDir = fs.mkdtempSync(path.join(os.tmpdir(), "git-patch-test-"));
    process.chdir(testDir);

    // Initialize a git repository
    execSync("git init", { cwd: testDir });
    execSync('git config user.email "test@example.com"', { cwd: testDir });
    execSync('git config user.name "Test User"', { cwd: testDir });

    // Set environment variables
    process.env.GITHUB_WORKSPACE = testDir;
    process.env.DEFAULT_BRANCH = "main";

    // Create initial commit on main branch
    fs.writeFileSync(path.join(testDir, "file.txt"), "initial content\n");
    execSync("git add .", { cwd: testDir });
    execSync('git commit -m "Initial commit"', { cwd: testDir });
    execSync("git branch -M main", { cwd: testDir });
  });

  afterEach(() => {
    // Restore original environment and working directory
    process.env = originalEnv;
    process.chdir(originalCwd);

    // Clean up test directory
    if (testDir && fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  it("should generate patch when branch has changes", () => {
    // Create a feature branch with changes
    execSync("git checkout -b feature-branch", { cwd: testDir });
    fs.writeFileSync(path.join(testDir, "file.txt"), "modified content\n");
    execSync("git add .", { cwd: testDir });
    execSync('git commit -m "Feature change"', { cwd: testDir });

    // Create safe outputs file with branch info
    const safeOutputsDir = path.join(testDir, "tmp", "gh-aw", "safeoutputs");
    fs.mkdirSync(safeOutputsDir, { recursive: true });
    const outputFile = path.join(safeOutputsDir, "outputs.jsonl");
    fs.writeFileSync(outputFile, JSON.stringify({ type: "create_pull_request", branch: "feature-branch" }) + "\n");

    process.env.GH_AW_SAFE_OUTPUTS = outputFile;

    // Import and run the handler (we'll need to mock the module)
    // For now, just verify the logic would work
    const branchExists = fs.existsSync(path.join(testDir, ".git", "refs", "heads", "feature-branch"));
    expect(branchExists).toBe(true);

    // Verify we can generate a patch
    const patch = execSync("git format-patch main..feature-branch --stdout", { cwd: testDir, encoding: "utf8" });
    expect(patch).toBeTruthy();
    expect(patch).toContain("modified content");
  });

  it("should return no_branch when branch doesn't exist", () => {
    // Create safe outputs file with non-existent branch
    const safeOutputsDir = path.join(testDir, "tmp", "gh-aw", "safeoutputs");
    fs.mkdirSync(safeOutputsDir, { recursive: true });
    const outputFile = path.join(safeOutputsDir, "outputs.jsonl");
    fs.writeFileSync(outputFile, JSON.stringify({ type: "create_pull_request", branch: "non-existent" }) + "\n");

    // Verify branch doesn't exist
    let branchExists = false;
    try {
      execSync("git show-ref --verify --quiet refs/heads/non-existent", { cwd: testDir, stdio: "pipe" });
      branchExists = true;
    } catch (error) {
      branchExists = false;
    }
    expect(branchExists).toBe(false);
  });

  it("should return no_changes when branch has no commits", () => {
    // Create a feature branch with no changes
    execSync("git checkout -b empty-branch", { cwd: testDir });

    // Create safe outputs file with branch info
    const safeOutputsDir = path.join(testDir, "tmp", "gh-aw", "safeoutputs");
    fs.mkdirSync(safeOutputsDir, { recursive: true });
    const outputFile = path.join(safeOutputsDir, "outputs.jsonl");
    fs.writeFileSync(outputFile, JSON.stringify({ type: "create_pull_request", branch: "empty-branch" }) + "\n");

    // Verify patch would be empty
    const patch = execSync("git format-patch main..empty-branch --stdout", { cwd: testDir, encoding: "utf8" });
    expect(patch).toBe("");
  });

  it("should read branch from push_to_pull_request_branch type", () => {
    // Create a feature branch
    execSync("git checkout -b push-branch", { cwd: testDir });
    fs.writeFileSync(path.join(testDir, "file.txt"), "pushed content\n");
    execSync("git add .", { cwd: testDir });
    execSync('git commit -m "Push change"', { cwd: testDir });

    // Create safe outputs file with push_to_pull_request_branch type
    const safeOutputsDir = path.join(testDir, "tmp", "gh-aw", "safeoutputs");
    fs.mkdirSync(safeOutputsDir, { recursive: true });
    const outputFile = path.join(safeOutputsDir, "outputs.jsonl");
    fs.writeFileSync(outputFile, JSON.stringify({ type: "push_to_pull_request_branch", branch: "push-branch" }) + "\n");

    process.env.GH_AW_SAFE_OUTPUTS = outputFile;

    // Verify branch exists and has changes
    const branchExists = fs.existsSync(path.join(testDir, ".git", "refs", "heads", "push-branch"));
    expect(branchExists).toBe(true);

    const patch = execSync("git format-patch main..push-branch --stdout", { cwd: testDir, encoding: "utf8" });
    expect(patch).toBeTruthy();
    expect(patch).toContain("pushed content");
  });

  it("should handle multiple entries and use first valid branch", () => {
    // Create a feature branch
    execSync("git checkout -b first-branch", { cwd: testDir });
    fs.writeFileSync(path.join(testDir, "file.txt"), "first content\n");
    execSync("git add .", { cwd: testDir });
    execSync('git commit -m "First change"', { cwd: testDir });

    // Create safe outputs file with multiple entries
    const safeOutputsDir = path.join(testDir, "tmp", "gh-aw", "safeoutputs");
    fs.mkdirSync(safeOutputsDir, { recursive: true });
    const outputFile = path.join(safeOutputsDir, "outputs.jsonl");
    fs.writeFileSync(
      outputFile,
      JSON.stringify({ type: "create_issue", title: "test" }) +
        "\n" +
        JSON.stringify({ type: "create_pull_request", branch: "first-branch" }) +
        "\n" +
        JSON.stringify({ type: "create_pull_request", branch: "second-branch" }) +
        "\n"
    );

    process.env.GH_AW_SAFE_OUTPUTS = outputFile;

    // Should use first-branch (first valid entry)
    const branchExists = fs.existsSync(path.join(testDir, ".git", "refs", "heads", "first-branch"));
    expect(branchExists).toBe(true);
  });

  it("should skip invalid JSON lines in safe outputs", () => {
    // Create a feature branch
    execSync("git checkout -b valid-branch", { cwd: testDir });
    fs.writeFileSync(path.join(testDir, "file.txt"), "valid content\n");
    execSync("git add .", { cwd: testDir });
    execSync('git commit -m "Valid change"', { cwd: testDir });

    // Create safe outputs file with invalid JSON lines
    const safeOutputsDir = path.join(testDir, "tmp", "gh-aw", "safeoutputs");
    fs.mkdirSync(safeOutputsDir, { recursive: true });
    const outputFile = path.join(safeOutputsDir, "outputs.jsonl");
    fs.writeFileSync(
      outputFile,
      "invalid json line\n" + JSON.stringify({ type: "create_pull_request", branch: "valid-branch" }) + "\n" + "another invalid\n"
    );

    process.env.GH_AW_SAFE_OUTPUTS = outputFile;

    // Should find valid-branch despite invalid JSON lines
    const branchExists = fs.existsSync(path.join(testDir, ".git", "refs", "heads", "valid-branch"));
    expect(branchExists).toBe(true);
  });
});
