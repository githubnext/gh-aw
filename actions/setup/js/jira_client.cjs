// @ts-check

const { sanitizeContent } = require("./sanitize_content.cjs");

const JIRA_API_PATH = "/rest/api/3";

function normalizeJiraBaseUrl(value) {
  const raw = typeof value === "string" ? value.trim() : "";
  if (!raw) {
    throw new Error("Jira configuration is missing JIRA_BASE_URL");
  }

  let url;
  try {
    url = new URL(raw);
  } catch {
    throw new Error("JIRA_BASE_URL must be a valid URL");
  }

  const isLocal = url.hostname === "localhost" || url.hostname === "127.0.0.1" || url.hostname === "::1";
  if (url.protocol !== "https:" && !(isLocal && url.protocol === "http:")) {
    throw new Error("JIRA_BASE_URL must use HTTPS");
  }
  if (url.search || url.hash) {
    throw new Error("JIRA_BASE_URL must not include a query string or fragment");
  }

  url.pathname = url.pathname.replace(/\/+$/, "").replace(/\/rest\/api\/3$/i, "");
  return url.toString().replace(/\/+$/, "");
}

function textToADF(value) {
  const text = String(value).replace(/\r\n?/g, "\n");
  return {
    type: "doc",
    version: 1,
    content: text.split("\n").map(line => ({
      type: "paragraph",
      content: line ? [{ type: "text", text: line }] : [],
    })),
  };
}

function redactJiraSecrets(value, secrets) {
  let result = String(value);
  for (const secret of secrets) {
    if (secret) {
      result = result.split(secret).join("***");
    }
  }
  return result;
}

function formatJiraError(status, statusText, responseBody, secrets) {
  const details = [];
  if (responseBody && typeof responseBody === "object") {
    if (Array.isArray(responseBody.errorMessages)) {
      details.push(...responseBody.errorMessages.filter(message => typeof message === "string"));
    }
    if (responseBody.errors && typeof responseBody.errors === "object" && !Array.isArray(responseBody.errors)) {
      for (const [field, message] of Object.entries(responseBody.errors)) {
        if (typeof message === "string") {
          details.push(`${field}: ${message}`);
        }
      }
    }
  }

  const detail = details.length > 0 ? `: ${details.join("; ")}` : "";
  const safe = sanitizeContent(redactJiraSecrets(`Jira API request failed (${status} ${statusText || "Error"})${detail}`, secrets), 2000);
  return safe || `Jira API request failed (${status})`;
}

function createJiraClient(env = process.env, fetchImpl = global.fetch) {
  const baseUrl = normalizeJiraBaseUrl(env.JIRA_BASE_URL);
  const email = typeof env.JIRA_USER_EMAIL === "string" ? env.JIRA_USER_EMAIL.trim() : "";
  const token = typeof env.JIRA_API_TOKEN === "string" ? env.JIRA_API_TOKEN : "";
  if (!email || !token) {
    throw new Error("Jira configuration requires JIRA_USER_EMAIL and JIRA_API_TOKEN");
  }
  if (typeof fetchImpl !== "function") {
    throw new Error("Jira requests require the fetch API");
  }

  const authorization = `Basic ${Buffer.from(`${email}:${token}`, "utf8").toString("base64")}`;
  const secrets = [token, email, authorization];

  return {
    async request(path, options = {}) {
      const normalizedPath = path.startsWith("/") ? path : `/${path}`;
      const url = `${baseUrl}${JIRA_API_PATH}${normalizedPath}`;
      let response;
      try {
        response = await fetchImpl(url, {
          method: options.method || "GET",
          headers: {
            Accept: "application/json",
            Authorization: authorization,
            "Content-Type": "application/json",
          },
          ...(options.body === undefined ? {} : { body: JSON.stringify(options.body) }),
        });
      } catch {
        throw new Error("Jira API request failed due to a network error");
      }

      const responseText = await response.text();
      let responseBody = null;
      if (responseText) {
        try {
          responseBody = JSON.parse(responseText);
        } catch {
          if (response.ok) {
            throw new Error(`Jira API returned an invalid JSON response (${response.status})`);
          }
        }
      }

      if (!response.ok) {
        throw new Error(formatJiraError(response.status, response.statusText, responseBody, secrets));
      }
      return responseBody;
    },
  };
}

module.exports = {
  JIRA_API_PATH,
  createJiraClient,
  formatJiraError,
  normalizeJiraBaseUrl,
  textToADF,
};
