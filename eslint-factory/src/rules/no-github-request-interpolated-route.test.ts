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
        // 'client' is not in the allow-list and not bound to an Octokit source
        "client.request(`GET /repos/${owner}/${repo}`);",
        // Dynamic computed access on known client name is not flagged
        "github['request'](`GET /repos/${owner}/${repo}`);",
        // Node.js http/https request — must not be flagged (false-positive guard)
        "http.request(`http://example.com/${path}`, callback);",
        "https.request(`https://example.com/${path}`, options, callback);",
        // Client bound to an unrelated value — must not be flagged
        "const client = http.createServer(); client.request(`GET /repos/${owner}/${repo}`);",
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
        // `this.github` is not resolved — `this` is not an Identifier
        "this.github.request(`GET /repos/${owner}/${repo}`, { owner, repo });",
        // Variable indirection for the route argument is not resolved
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

  it("invalid: opaque whole-route helpers are flagged with tailored guidance", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [],
      invalid: [
        {
          code: "function addReaction(endpoint) { github.request(`POST ${endpoint}`, { content: '+1' }); }",
          errors: [
            {
              messageId: "opaqueWholeRoute",
              data: { kind: "template literal with interpolations", client: "github" },
            },
          ],
        },
        {
          code: 'function addComment(endpoint) { github.request("POST " + endpoint, { body }); }',
          errors: [
            {
              messageId: "opaqueWholeRoute",
              data: { kind: "string concatenation expression", client: "github" },
            },
          ],
        },
      ],
    });
  });

  it("valid: simple aliases bound to non-Octokit sources are not flagged", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [
        // Alias of an unknown source — must not be flagged
        "const gh = someUnknownClient; gh.request(`GET /repos/${owner}/${repo}`);",
        // Alias of http module — must not be flagged
        "const client = http; client.request(`GET /repos/${owner}/${repo}`);",
        // Unknown .getOctokit() owner must not be treated as toolkit Octokit source
        "const helperClient = myHelper.getOctokit(token); helperClient.request(`GET /repos/${owner}/${repo}`);",
      ],
      invalid: [],
    });
  });

  it("invalid: context.github.request() with interpolated route is flagged", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [],
      invalid: [
        {
          code: "context.github.request(`GET /repos/${owner}/${repo}`, { owner, repo });",
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "template literal with interpolations", client: "context.github" },
            },
          ],
        },
        {
          code: `context.github.request("GET /repos/" + owner + "/" + repo, { owner, repo });`,
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "string concatenation expression", client: "context.github" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: simple const alias of a known Octokit client is flagged", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [],
      invalid: [
        {
          code: "const gh = github; gh.request(`GET /repos/${owner}/${repo}`, { owner, repo });",
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "template literal with interpolations", client: "gh" },
            },
          ],
        },
        {
          code: `const myClient = octokit; myClient.request("GET /repos/" + owner + "/" + repo, { });`,
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "string concatenation expression", client: "myClient" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: const alias of getOctokit() result is flagged", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [],
      invalid: [
        {
          code: "const fallbackClient = getOctokit(token); fallbackClient.request(`POST /repos/${owner}/${repo}/issues/${number}/comments`, { body });",
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "template literal with interpolations", client: "fallbackClient" },
            },
          ],
        },
        {
          code: `const client = getOctokit(token); client.request("POST /repos/" + owner + "/" + repo, { });`,
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "string concatenation expression", client: "client" },
            },
          ],
        },
        {
          code: "const client = actions.getOctokit(token); client.request(`POST /repos/${owner}/${repo}/issues`, { });",
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "template literal with interpolations", client: "client" },
            },
          ],
        },
      ],
    });
  });

  it("valid: mutable aliases are out of scope and not flagged", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [
        // Mutable let bindings are not trusted aliases.
        "let gh = github; gh.request(`GET /repos/${owner}/${repo}`, { owner, repo });",
        // var aliases are always out of scope, even without reassignment.
        "var legacyClient = github; legacyClient.request(`GET /repos/${owner}/${repo}`);",
        // var alias can be reassigned, so it is intentionally out of scope.
        "var client = getOctokit(token); client = http; client.request(`GET /repos/${owner}/${repo}`);",
      ],
      invalid: [],
    });
  });

  it("invalid: const alias of context.github is flagged", () => {
    cjsRuleTester.run("no-github-request-interpolated-route", noGithubRequestInterpolatedRouteRule, {
      valid: [],
      invalid: [
        {
          code: "const gh = context.github; gh.request(`GET /repos/${owner}/${repo}`, { owner, repo });",
          errors: [
            {
              messageId: "interpolatedRoute",
              data: { kind: "template literal with interpolations", client: "gh" },
            },
          ],
        },
      ],
    });
  });
});
