const fs = require("fs"),
  os = require("os"),
  path = require("path"),
  substitutePlaceholders = require("./substitute_placeholders.cjs");
describe("substitutePlaceholders", () => {
  let tempDir, testFile;
  (beforeEach(() => {
    ((tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "substitute-test-"))), (testFile = path.join(tempDir, "test.txt")));
  }),
    afterEach(() => {
      (fs.existsSync(testFile) && fs.unlinkSync(testFile), fs.existsSync(tempDir) && fs.rmdirSync(tempDir));
    }),
    it("should substitute a single placeholder", async () => {
      (fs.writeFileSync(testFile, "Hello __NAME__!", "utf8"), await substitutePlaceholders({ file: testFile, substitutions: { NAME: "World" } }));
      const content = fs.readFileSync(testFile, "utf8");
      expect(content).toBe("Hello World!");
    }),
    it("should substitute multiple placeholders", async () => {
      (fs.writeFileSync(testFile, "Repository: __REPO__\nActor: __ACTOR__\nBranch: __BRANCH__", "utf8"), await substitutePlaceholders({ file: testFile, substitutions: { REPO: "test/repo", ACTOR: "testuser", BRANCH: "main" } }));
      const content = fs.readFileSync(testFile, "utf8");
      expect(content).toBe("Repository: test/repo\nActor: testuser\nBranch: main");
    }),
    it("should handle special characters safely", async () => {
      (fs.writeFileSync(testFile, "Command: __CMD__", "utf8"), await substitutePlaceholders({ file: testFile, substitutions: { CMD: "$(malicious) `backdoor` ${VAR} | pipe" } }));
      const content = fs.readFileSync(testFile, "utf8");
      expect(content).toBe("Command: $(malicious) `backdoor` ${VAR} | pipe");
    }),
    it("should handle placeholders appearing multiple times", async () => {
      (fs.writeFileSync(testFile, "__NAME__ is great. I love __NAME__!", "utf8"), await substitutePlaceholders({ file: testFile, substitutions: { NAME: "Testing" } }));
      const content = fs.readFileSync(testFile, "utf8");
      expect(content).toBe("Testing is great. I love Testing!");
    }),
    it("should leave unmatched placeholders unchanged", async () => {
      (fs.writeFileSync(testFile, "__FOO__ and __BAR__", "utf8"), await substitutePlaceholders({ file: testFile, substitutions: { FOO: "foo" } }));
      const content = fs.readFileSync(testFile, "utf8");
      expect(content).toBe("foo and __BAR__");
    }),
    it("should handle empty values", async () => {
      (fs.writeFileSync(testFile, "Value: __VAL__", "utf8"), await substitutePlaceholders({ file: testFile, substitutions: { VAL: "" } }));
      const content = fs.readFileSync(testFile, "utf8");
      expect(content).toBe("Value: ");
    }),
    it("should throw error if file parameter is missing", async () => {
      await expect(substitutePlaceholders({ substitutions: { NAME: "test" } })).rejects.toThrow("file parameter is required");
    }),
    it("should throw error if substitutions parameter is missing", async () => {
      await expect(substitutePlaceholders({ file: testFile })).rejects.toThrow("substitutions parameter must be an object");
    }),
    it("should throw error if file does not exist", async () => {
      await expect(substitutePlaceholders({ file: "/nonexistent/file.txt", substitutions: { NAME: "test" } })).rejects.toThrow("Failed to read file");
    }),
    it("should handle undefined values as empty strings", async () => {
      (fs.writeFileSync(testFile, "Value: __VAL__", "utf8"), await substitutePlaceholders({ file: testFile, substitutions: { VAL: undefined } }));
      const content = fs.readFileSync(testFile, "utf8");
      expect(content).toBe("Value: ");
    }),
    it("should handle null values as empty strings", async () => {
      (fs.writeFileSync(testFile, "Value: __VAL__", "utf8"), await substitutePlaceholders({ file: testFile, substitutions: { VAL: null } }));
      const content = fs.readFileSync(testFile, "utf8");
      expect(content).toBe("Value: ");
    }),
    it("should handle mixed undefined and defined values", async () => {
      (fs.writeFileSync(testFile, "Repo: __REPO__\nComment: __COMMENT__\nIssue: __ISSUE__", "utf8"), await substitutePlaceholders({ file: testFile, substitutions: { REPO: "test/repo", COMMENT: undefined, ISSUE: null } }));
      const content = fs.readFileSync(testFile, "utf8");
      expect(content).toBe("Repo: test/repo\nComment: \nIssue: ");
    }));
});
