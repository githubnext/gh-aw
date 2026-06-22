// @ts-check
const { renderLockdownTokenErrorMessage, renderPublicStrictModeErrorMessage, renderPullRequestTargetErrorMessage } = require("./validate_lockdown_requirements_templates.cjs");

// Content-contract tests below assert specific phrasing and key terms from each template.
// They document intentional wording and will fail on purposeful rewording — that is the intent.
describe("validate_lockdown_requirements_templates", () => {
  describe("renderLockdownTokenErrorMessage", () => {
    it("returns a non-empty string", () => {
      const message = renderLockdownTokenErrorMessage();
      expect(message).toBeTypeOf("string");
      expect(message.length).toBeGreaterThan(0);
    });

    it("includes the auth documentation URL", () => {
      const message = renderLockdownTokenErrorMessage();
      expect(message).toContain("https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/auth.mdx");
    });

    it("includes GH_AW_GITHUB_TOKEN recommendation", () => {
      const message = renderLockdownTokenErrorMessage();
      expect(message).toContain("GH_AW_GITHUB_TOKEN (recommended)");
    });

    it("includes GH_AW_GITHUB_MCP_SERVER_TOKEN as alternative", () => {
      const message = renderLockdownTokenErrorMessage();
      expect(message).toContain("GH_AW_GITHUB_MCP_SERVER_TOKEN (alternative)");
    });

    it("includes the gh aw secrets set command", () => {
      const message = renderLockdownTokenErrorMessage();
      expect(message).toContain("gh aw secrets set GH_AW_GITHUB_TOKEN");
    });

    it("mentions lockdown mode is enabled", () => {
      const message = renderLockdownTokenErrorMessage();
      expect(message).toContain("Lockdown mode is enabled");
    });

    it("does not contain unreplaced {auth_docs_url} placeholder", () => {
      const message = renderLockdownTokenErrorMessage();
      expect(message).not.toContain("{auth_docs_url}");
    });
  });

  describe("renderPublicStrictModeErrorMessage", () => {
    it("returns a non-empty string", () => {
      const message = renderPublicStrictModeErrorMessage();
      expect(message).toBeTypeOf("string");
      expect(message.length).toBeGreaterThan(0);
    });

    it("includes the security documentation URL", () => {
      const message = renderPublicStrictModeErrorMessage();
      expect(message).toContain("https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/security.mdx");
    });

    it("includes the strict compile command", () => {
      const message = renderPublicStrictModeErrorMessage();
      expect(message).toContain("gh aw compile --strict");
    });

    it("mentions public repository context", () => {
      const message = renderPublicStrictModeErrorMessage();
      expect(message).toContain("public repository");
    });

    it("mentions strict mode", () => {
      const message = renderPublicStrictModeErrorMessage();
      expect(message).toContain("strict mode");
    });

    it("does not contain unreplaced {strict_compile_command} placeholder", () => {
      const message = renderPublicStrictModeErrorMessage();
      expect(message).not.toContain("{strict_compile_command}");
    });

    it("does not contain unreplaced {security_docs_url} placeholder", () => {
      const message = renderPublicStrictModeErrorMessage();
      expect(message).not.toContain("{security_docs_url}");
    });
  });

  describe("renderPullRequestTargetErrorMessage", () => {
    it("returns a non-empty string", () => {
      const message = renderPullRequestTargetErrorMessage();
      expect(message).toBeTypeOf("string");
      expect(message.length).toBeGreaterThan(0);
    });

    it("includes the security documentation URL", () => {
      const message = renderPullRequestTargetErrorMessage();
      expect(message).toContain("https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/security.mdx");
    });

    it("mentions the pull_request_target event", () => {
      const message = renderPullRequestTargetErrorMessage();
      expect(message).toContain("pull_request_target");
    });

    it("mentions pwn request security risk", () => {
      const message = renderPullRequestTargetErrorMessage();
      expect(message).toContain("pwn request");
    });

    it("mentions public repositories", () => {
      const message = renderPullRequestTargetErrorMessage();
      expect(message).toContain("public repositories");
    });

    it("suggests using the pull_request event instead", () => {
      const message = renderPullRequestTargetErrorMessage();
      expect(message).toContain("pull_request event instead");
    });

    it("does not contain unreplaced {security_docs_url} placeholder", () => {
      const message = renderPullRequestTargetErrorMessage();
      expect(message).not.toContain("{security_docs_url}");
    });
  });

  describe("cross-function checks", () => {
    it("each render function returns a distinct message", () => {
      const lockdown = renderLockdownTokenErrorMessage();
      const strictMode = renderPublicStrictModeErrorMessage();
      const prTarget = renderPullRequestTargetErrorMessage();

      expect(lockdown).not.toBe(strictMode);
      expect(lockdown).not.toBe(prTarget);
      expect(strictMode).not.toBe(prTarget);
    });

    it("lockdown message does not contain strict mode content", () => {
      const message = renderLockdownTokenErrorMessage();
      expect(message).not.toContain("gh aw compile --strict");
    });

    it("strict mode message does not contain lockdown token content", () => {
      const message = renderPublicStrictModeErrorMessage();
      expect(message).not.toContain("GH_AW_GITHUB_TOKEN");
    });

    it("module exposes exactly the expected exports", () => {
      const mod = require("./validate_lockdown_requirements_templates.cjs");
      const keys = Object.keys(mod);
      expect(keys).toHaveLength(3);
      expect(keys).toEqual(expect.arrayContaining(["renderLockdownTokenErrorMessage", "renderPublicStrictModeErrorMessage", "renderPullRequestTargetErrorMessage"]));
    });

    it("no message contains unreplaced placeholders", () => {
      const messages = [renderLockdownTokenErrorMessage(), renderPublicStrictModeErrorMessage(), renderPullRequestTargetErrorMessage()];
      for (const message of messages) {
        expect(message).not.toMatch(/\{[A-Za-z][A-Za-z0-9_]*\}/);
      }
    });
  });
});
