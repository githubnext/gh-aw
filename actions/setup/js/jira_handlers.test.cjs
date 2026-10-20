// @ts-check
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { main: createIssueMain } = require("./jira_create_issue.cjs");
const { main: updateIssueMain } = require("./jira_update_issue.cjs");
const { main: addCommentMain } = require("./jira_add_comment.cjs");
const { main: addLabelMain } = require("./jira_add_label.cjs");

describe("Jira safe-output handlers", () => {
  let requests;

  beforeEach(() => {
    requests = [];
    global.core = {
      info: vi.fn(),
      error: vi.fn(),
      warning: vi.fn(),
      debug: vi.fn(),
    };
    process.env.JIRA_BASE_URL = "https://example.atlassian.net";
    process.env.JIRA_USER_EMAIL = "jira@example.com";
    process.env.JIRA_API_TOKEN = "secret-token";
    global.fetch = vi.fn(async (url, options) => {
      requests.push({ url, options, body: options.body ? JSON.parse(options.body) : undefined });
      if (url.endsWith("/comment")) {
        return response(201, { id: "20001", self: `${url}/20001` });
      }
      if (options.method === "POST") {
        return response(201, { id: "10042", key: "ENG-123", self: `${url}/10042` });
      }
      return response(204);
    });
  });

  afterEach(() => {
    delete process.env.JIRA_BASE_URL;
    delete process.env.JIRA_USER_EMAIL;
    delete process.env.JIRA_API_TOKEN;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
  });

  it("creates Jira issues with issue type name and ADF description", async () => {
    const handler = await createIssueMain({ max: 1, body_footer: "Configured footer" });
    const result = await handler({
      temporary_id: "#aw_jira123",
      project_key: "ENG",
      issue_type: "Task",
      summary: "Investigate parser",
      description: "First paragraph\nSecond paragraph",
    });

    expect(result).toMatchObject({
      success: true,
      issue_key: "ENG-123",
      issue_id: "10042",
      temporaryId: "#aw_jira123",
      temporaryIdEntry: { provider: "jira", resourceType: "issue", issueKey: "ENG-123" },
    });
    expect(requests[0]).toMatchObject({
      url: "https://example.atlassian.net/rest/api/3/issue",
      body: {
        fields: {
          project: { key: "ENG" },
          issuetype: { name: "Task" },
          summary: "Investigate parser",
          description: {
            type: "doc",
            version: 1,
          },
        },
      },
    });
    expect(requests[0].body.fields.description.content.at(-1).content[0].text).toBe("Configured footer");
    expect(global.core.debug).toHaveBeenCalledWith("jira_create_issue: processing request");
    expect(global.core.debug).toHaveBeenCalledWith("jira_create_issue: request completed successfully");
  });

  it.each([
    [{ summary: "Updated" }, { summary: "Updated" }],
    [{ description: "Updated description" }, { description: { type: "doc", version: 1 } }],
    [
      { summary: "Updated", description: "Both" },
      { summary: "Updated", description: { type: "doc", version: 1 } },
    ],
  ])("updates only requested Jira fields", async (updates, expectedFields) => {
    const handler = await updateIssueMain({ max: 1 });
    const result = await handler({ issue_key: "ENG-123", ...updates });

    expect(result).toMatchObject({ success: true, issue_key: "ENG-123" });
    expect(requests[0].url).toBe("https://example.atlassian.net/rest/api/3/issue/ENG-123");
    expect(requests[0].body.fields).toMatchObject(expectedFields);
    expect(Object.keys(requests[0].body.fields)).toEqual(Object.keys(updates));
  });

  it.each([
    [updateIssueMain, { summary: "Updated" }, "/rest/api/3/issue/ENG-123"],
    [addCommentMain, { body: "Investigation complete." }, "/rest/api/3/issue/ENG-123/comment"],
    [addLabelMain, { label: "smoke-tested" }, "/rest/api/3/issue/ENG-123"],
  ])("resolves temporary Jira issue IDs", async (factory, fields, path) => {
    const handler = await factory({ max: 1 });
    const result = await handler(
      { issue_key: "#aw_jira123", ...fields },
      {
        aw_jira123: { provider: "jira", resourceType: "issue", issueKey: "ENG-123" },
      }
    );

    expect(result).toMatchObject({ success: true, issue_key: "ENG-123" });
    expect(requests[0].url).toBe(`https://example.atlassian.net${path}`);
  });

  it("appends a configured body footer to Jira issue updates", async () => {
    const handler = await updateIssueMain({ max: 1, body_footer: "Configured footer" });
    await handler({ issue_key: "ENG-123", description: "Updated description" });

    expect(requests[0].body.fields.description.content.at(-1).content[0].text).toBe("Configured footer");
  });

  it("rejects a Jira update with no changed fields", async () => {
    const handler = await updateIssueMain({ max: 1 });
    await expect(handler({ issue_key: "ENG-123" })).resolves.toMatchObject({
      success: false,
      error: "jira_update_issue requires summary or description",
    });
    expect(requests).toHaveLength(0);
  });

  it("adds a Jira comment as ADF", async () => {
    const handler = await addCommentMain({ max: 1 });
    const result = await handler({ issue_key: "ENG-123", body: "Investigation complete." });

    expect(result).toMatchObject({ success: true, issue_key: "ENG-123", comment_id: "20001" });
    expect(requests[0].body).toEqual({
      body: {
        type: "doc",
        version: 1,
        content: [{ type: "paragraph", content: [{ type: "text", text: "Investigation complete." }] }],
      },
    });
  });

  it("preserves valid Jira text without GitHub-specific sanitization", async () => {
    const handler = await createIssueMain({ max: 1 });
    await handler({
      temporary_id: "#aw_jira123",
      project_key: "ENG",
      issue_type: "Task",
      summary: "Fix @deprecated flag",
      description: "Use {{version}}.",
    });

    expect(requests[0].body.fields.summary).toBe("Fix @deprecated flag");
    expect(requests[0].body.fields.description.content[0].content[0].text).toBe("Use {{version}}.");
  });

  it.each([false, true])("normalizes controls and secrets while preserving literal ADF (staged=%s)", async staged => {
    vi.stubEnv("GH_AW_SECRET_NAMES", "JIRA_REPORT");
    vi.stubEnv("SECRET_JIRA_REPORT", "jira-report-fixture");
    const handler = await createIssueMain({ max: 1, staged, body_footer: "Footer jira-report-fixture" });
    const result = await handler({
      temporary_id: "#aw_jira123",
      project_key: "ENG",
      issue_type: "Task",
      summary: "e\u0301vidence\u0000\r\nreport jira-report-fixture",
      description: "Literal **text** <tag> @user\r\n\r\n  e\u0301vidence\u0000\t{{version}}\r\njira-report-fixture",
    });
    expect(result.success).toBe(true);
    if (staged) {
      expect(result.summary).toBe("\u00e9vidence report ***");
      expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining("Literal **text** <tag> @user\n\n  \u00e9vidence\t{{version}}\n***\n\nFooter ***"));
      expect(JSON.stringify(global.core.info.mock.calls)).not.toContain("jira-report-fixture");
      expect(requests).toHaveLength(0);
    } else {
      expect(requests[0].body.fields.summary).toBe("\u00e9vidence report ***");
      expect(requests[0].body.fields.description.content).toEqual([
        { type: "paragraph", content: [{ type: "text", text: "Literal **text** <tag> @user" }] },
        { type: "paragraph", content: [] },
        { type: "paragraph", content: [{ type: "text", text: "  \u00e9vidence\t{{version}}" }] },
        { type: "paragraph", content: [{ type: "text", text: "***" }] },
        { type: "paragraph", content: [] },
        { type: "paragraph", content: [{ type: "text", text: "Footer ***" }] },
      ]);
      expect(JSON.stringify(requests[0].body)).not.toContain("jira-report-fixture");
    }
  });

  it.each([false, true])("preserves multilingual spelling and literal entities (staged=%s)", async staged => {
    const summary = "Ошибка сборки / Σφάλμα μεταγλώττισης &amp; &#x41;";
    const description = "Сборка не удалась.\nΗ μεταγλώττιση απέτυχε.\nUse &lt;tag&gt;, &amp;amp;, &#64;user, ﬁ and Ａ.";
    const handler = await createIssueMain({ max: 1, staged, body_footer: "Отчет / Αναφορά" });
    const result = await handler({
      temporary_id: "#aw_jira123",
      project_key: "ENG",
      issue_type: "Task",
      summary,
      description,
    });

    expect(result.success).toBe(true);
    if (staged) {
      expect(result.summary).toBe(summary);
      expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining(`Summary: ${summary}; Description: ${description}\n\nОтчет / Αναφορά`));
      expect(requests).toHaveLength(0);
    } else {
      expect(requests[0].body.fields.summary).toBe(summary);
      expect(requests[0].body.fields.description.content.map(paragraph => paragraph.content.map(node => node.text).join("")).join("\n")).toBe(`${description}\n\nОтчет / Αναφορά`);
    }
  });

  it.each([false, true])("redacts secrets exposed by literal control cleanup (staged=%s)", async staged => {
    vi.stubEnv("GH_AW_SECRET_NAMES", "JIRA_REPORT");
    vi.stubEnv("SECRET_JIRA_REPORT", "jira-report-fixture");
    const splitSecret = "jira-\u0000re\u200bport-\u202efixture";
    const handler = await createIssueMain({ max: 1, staged, body_footer: `Отчет ${splitSecret}` });
    const result = await handler({
      temporary_id: "#aw_jira123",
      project_key: "ENG",
      issue_type: "Task",
      summary: `Ошибка\rсборки ${splitSecret}`,
      description: `Σφάλμα\u0000\r\n\t${splitSecret}\r\u001b[31me\u0000\u0301vidence\u001b[0m\u{e0041}`,
    });

    expect(result.success).toBe(true);
    if (staged) {
      expect(result.summary).toBe("Ошибка сборки ***");
      expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining("Σφάλμα\n\t***\névidence\n\nОтчет ***"));
      expect(requests).toHaveLength(0);
    } else {
      expect(requests[0].body.fields.summary).toBe("Ошибка сборки ***");
      expect(requests[0].body.fields.description.content).toEqual([
        { type: "paragraph", content: [{ type: "text", text: "Σφάλμα" }] },
        { type: "paragraph", content: [{ type: "text", text: "\t***" }] },
        { type: "paragraph", content: [{ type: "text", text: "évidence" }] },
        { type: "paragraph", content: [] },
        { type: "paragraph", content: [{ type: "text", text: "Отчет ***" }] },
      ]);
    }
    expect(JSON.stringify([requests, global.core.info.mock.calls])).not.toContain("jira-report-fixture");
  });

  it("rejects descriptions emptied by literal normalization before making an API write", async () => {
    const handler = await createIssueMain({ max: 1 });
    const result = await handler({ temporary_id: "#aw_jira123", project_key: "ENG", issue_type: "Task", summary: "Report", description: "\u0000" });
    expect(result).toMatchObject({ success: false, error: "description must be non-empty after normalization" });
    expect(requests).toHaveLength(0);
  });

  it.each([false, true])("validates normalized text and Jira limits (staged=%s)", async staged => {
    const cases = [
      [{ summary: "\u0000" }, "summary must be a non-empty string"],
      [{ summary: "x".repeat(256) }, "summary must be at most 255 characters"],
      [{ description: "" }, "description must be a non-empty string"],
      [{ description: "\u0000" }, "description must be non-empty after normalization"],
      [{ description: "x".repeat(32768) }, "description must be at most 32767 characters"],
    ];
    for (const [fields, error] of cases) {
      const handler = await createIssueMain({ max: 1, staged });
      const result = await handler({ temporary_id: "#aw_jira123", project_key: "ENG", issue_type: "Task", summary: "Report", ...fields });
      expect(result).toMatchObject({ success: false, error });
    }
    expect(requests).toHaveLength(0);
  });

  it("keeps descriptions optional even with a configured footer", async () => {
    const handler = await createIssueMain({ max: 1, body_footer: "Configured footer" });
    const result = await handler({ temporary_id: "#aw_jira123", project_key: "ENG", issue_type: "Task", summary: "Report" });
    expect(result.success).toBe(true);
    expect(requests[0].body.fields).not.toHaveProperty("description");
  });

  it("fits Jira descriptions without omitting the configured footer", async () => {
    const handler = await createIssueMain({ max: 1, body_footer: "Configured footer" });
    const result = await handler({ temporary_id: "#aw_jira123", project_key: "ENG", issue_type: "Task", summary: "x".repeat(255), description: "x".repeat(32767) });
    expect(result.success).toBe(true);
    expect(requests[0].body.fields.summary).toHaveLength(255);
    expect(requests[0].body.fields.description.content[0].content[0].text).toBe("Content redacted to fit size limit.");
    const text = requests[0].body.fields.description.content
      .flatMap(block => block.content || [])
      .map(inline => inline.text || "")
      .join("\n");
    expect(text.length).toBeLessThanOrEqual(32767);
    expect(text).toContain("Configured footer");
    expect(global.core.warning).not.toHaveBeenCalledWith(expect.stringContaining("Configured body footer omitted"));
  });

  it("adds one Jira label with additive update semantics", async () => {
    const handler = await addLabelMain({ max: 1 });
    const result = await handler({ issue_key: "ENG-123", label: "needs-investigation" });

    expect(result).toMatchObject({ success: true, issue_key: "ENG-123", label: "needs-investigation" });
    expect(requests[0].body).toEqual({ update: { labels: [{ add: "needs-investigation" }] } });
    expect(requests[0].body.fields).toBeUndefined();
  });

  it("rejects labels that Jira cannot accept", async () => {
    const handler = await addLabelMain({ max: 1 });
    await expect(handler({ issue_key: "ENG-123", label: "needs investigation" })).resolves.toMatchObject({
      success: false,
      error: "label must contain only letters, numbers, periods, hyphens, and underscores",
    });
    expect(requests).toHaveLength(0);
  });

  it.each([
    [createIssueMain, { temporary_id: "#aw_jira123", project_key: "ENG", issue_type: "Task", summary: "Preview" }, "Jira create issue"],
    [updateIssueMain, { issue_key: "ENG-123", summary: "Preview" }, "Jira update issue"],
    [addCommentMain, { issue_key: "ENG-123", body: "Preview" }, "Jira add comment"],
    [addLabelMain, { issue_key: "ENG-123", label: "preview" }, "Jira add label"],
  ])("stages every Jira operation without credentials or HTTP requests", async (factory, message, previewText) => {
    delete process.env.JIRA_BASE_URL;
    delete process.env.JIRA_USER_EMAIL;
    delete process.env.JIRA_API_TOKEN;
    const handler = await factory({ max: 1, staged: true });
    const result = await handler(message);

    expect(result).toMatchObject({ success: true, staged: true });
    expect(global.fetch).not.toHaveBeenCalled();
    expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining(previewText));
  });

  it("returns a safe Jira API error", async () => {
    global.fetch = vi.fn(async () =>
      response(400, {
        errorMessages: ["Invalid project"],
        errors: { summary: "Invalid summary" },
      })
    );
    const handler = await createIssueMain({ max: 1 });
    const result = await handler({ temporary_id: "#aw_jira123", project_key: "BAD", issue_type: "Task", summary: "Bad issue" });

    expect(result).toMatchObject({ success: false });
    expect(result.error).toContain("Invalid project");
    expect(result.error).not.toContain("secret-token");
    expect(global.core.debug).toHaveBeenCalledWith("jira_create_issue: request failed");
  });
});

function response(status, body) {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 204 ? "No Content" : status === 201 ? "Created" : "Bad Request",
    text: async () => (body === undefined ? "" : JSON.stringify(body)),
  };
}
