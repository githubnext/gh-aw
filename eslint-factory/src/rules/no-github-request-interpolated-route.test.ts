import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noGithubRequestInterpolatedRouteRule } from "./no-github-request-interpolated-route";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-github-request-interpolated-route", () => {
  it("uses the correct docs URL", () => {
    expect(noGithubRequestInterpolatedRouteRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-github-request-interpolated-route");
  });

  it("valid: plain string literal route is accepted", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [
        `github.request("POST /repos/{owner}/{repo}/issues/{issue_number}/reactions", { owner, repo, issue_number, content: "+1" });`,
        `octokit.request("GET /repos/{owner}/{repo}", { owner, repo });`,
        `githubClient.request("DELETE /reactions/{reaction_id}", { reaction_id });`,
        `octokitClient.request("PATCH /issues/{issue_number}", { issue_number });`,
      ],
      invalid: [],
    });
  });

  it("valid: plain template literal without interpolations is accepted", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [
        // Template literal with no expressions — equivalent to a plain string
        "github.request(`GET /repos/{owner}/{repo}`, { owner, repo });",
      ],
      invalid: [],
    });
  });

  it("valid: unrelated method calls on known client names are not flagged", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: ["github.rest.issues.create({ owner, repo, title });", "octokit.paginate(octokit.rest.pulls.list, { owner, repo });", `github.graphql(\`{ viewer { login } }\`);`],
      invalid: [],
    });
  });

  it("valid: request() on an unknown client name is not flagged", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [
        // 'client' is not in the allow-list
        "client.request(`GET /repos/${owner}/${repo}`);",
        // Dynamic computed access on known client name is not flagged
        "github['request'](`GET /repos/${owner}/${repo}`);",
      ],
      invalid: [],
    });
  });

  it("valid: request() called without arguments is not flagged", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: ["github.request();"],
      invalid: [],
    });
  });

  it("valid: intentionally out-of-scope route forms are accepted", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [
        "this.github.request(`GET /repos/${owner}/${repo}`, { owner, repo });",
        "context.github.request(`GET /repos/${owner}/${repo}`, { owner, repo });",
        "const route = `GET /repos/${owner}/${repo}`; github.request(route, { owner, repo });",
        `github.request("GET /repos/".concat(owner, "/", repo), { owner, repo });`,
        `github.request("GET /repos" + "/{owner}/{repo}", { owner, repo });`,
      ],
      invalid: [],
    });
  });

  it("invalid: template literal with interpolations is flagged for all known client names", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [],
      invalid: [
        {
          code: "github.request(`POST /repos/${owner}/${repo}/issues/${issue_number}/reactions`, { content: '+1' });",
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "template literal with interpolations", client: "github" },
            },
          ],
        },
        {
          code: "octokit.request(`GET /repos/${owner}/${repo}`, { });",
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "template literal with interpolations", client: "octokit" },
            },
          ],
        },
        {
          code: "githubClient.request(`DELETE /reactions/${reaction_id}`, { });",
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "template literal with interpolations", client: "githubClient" },
            },
          ],
        },
        {
          code: "octokitClient.request(`PATCH /issues/${issue_number}`, { });",
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "template literal with interpolations", client: "octokitClient" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: string concatenation route is flagged", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [],
      invalid: [
        {
          code: `github.request("GET /repos/" + owner + "/" + repo, { });`,
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "string concatenation expression", client: "github" },
            },
          ],
        },
        {
          code: `octokit.request("POST /orgs/" + org + "/teams", { name });`,
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "string concatenation expression", client: "octokit" },
            },
          ],
        },
        {
          code: `githubClient.request("GET /repos/" + owner + "/" + repo + "/actions/runs/" + runId, { });`,
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "string concatenation expression", client: "githubClient" },
            },
          ],
        },
        {
          code: `octokitClient.request("GET /repos/" + owner + "/" + repo, { });`,
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "string concatenation expression", client: "octokitClient" },
            },
          ],
        },
      ],
    });
  });
});
