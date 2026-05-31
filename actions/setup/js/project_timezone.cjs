// @ts-check

const fs = require("fs");
const path = require("path");

const REPO_CONFIG_PATH = [".github", "workflows", "aw.json"];

function isValidTimeZone(timeZone) {
  if (!timeZone) {
    return false;
  }

  try {
    new Intl.DateTimeFormat("en-US", { timeZone }).format(new Date());
    return true;
  } catch {
    return false;
  }
}

function getRepoConfigPath() {
  return path.join(process.env.GITHUB_WORKSPACE || process.cwd(), ...REPO_CONFIG_PATH);
}

function warn(message) {
  global.core?.warning?.(message);
}

function readRepoConfigTimeZone() {
  const repoConfigPath = getRepoConfigPath();

  try {
    const raw = fs.readFileSync(repoConfigPath, "utf8");
    const parsed = JSON.parse(raw);
    const timeZone = typeof parsed?.utc === "string" ? parsed.utc.trim() : "";
    if (!timeZone) {
      return "";
    }
    if (!isValidTimeZone(timeZone)) {
      warn(`Ignoring invalid repo timezone in ${repoConfigPath}: ${timeZone}`);
      return "";
    }
    return timeZone;
  } catch (error) {
    if (error && typeof error === "object" && "code" in error && (error.code === "ENOENT" || error.code === "ENOTDIR")) {
      return "";
    }
    if (error instanceof SyntaxError) {
      warn(`Ignoring invalid JSON in ${repoConfigPath}: ${error.message}`);
      return "";
    }
    warn(`Failed to read ${repoConfigPath}: ${error instanceof Error ? error.message : String(error)}`);
    return "";
  }
}

function readDefaultTimeZone() {
  const timeZone = (process.env.GH_AW_DEFAULT_UTC || "").trim();
  if (!timeZone) {
    return "";
  }
  if (!isValidTimeZone(timeZone)) {
    warn(`Ignoring invalid ${"GH_AW_DEFAULT_UTC"} timezone: ${timeZone}`);
    return "";
  }
  return timeZone;
}

function resolveProjectTimeZone() {
  return readRepoConfigTimeZone() || readDefaultTimeZone();
}

function normalizeUTCOffset(timeZoneName) {
  if (!timeZoneName || timeZoneName === "GMT" || timeZoneName === "UTC") {
    return "+00:00";
  }

  const match = timeZoneName.match(/^(?:GMT|UTC)([+-])(\d{1,2})(?::?(\d{2}))?$/);
  if (!match) {
    return "+00:00";
  }

  const [, sign, hours, minutes = "00"] = match;
  return `${sign}${hours.padStart(2, "0")}:${minutes.padStart(2, "0")}`;
}

function formatProjectTimeZoneOffset(date, timeZone) {
  const timeZonePart = new Intl.DateTimeFormat("en-US", {
    timeZone,
    timeZoneName: "shortOffset",
    hour: "numeric",
  })
    .formatToParts(date)
    .find(part => part.type === "timeZoneName");

  return normalizeUTCOffset(timeZonePart?.value || "UTC");
}

function formatDateInProjectTimeZone(date) {
  const timeZone = resolveProjectTimeZone();
  if (!timeZone) {
    return "";
  }

  const formatted = new Intl.DateTimeFormat("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZone,
    timeZoneName: "short",
  }).format(date);
  const offset = formatProjectTimeZoneOffset(date, timeZone);
  return `${formatted} (UTC${offset})`;
}

module.exports = {
  formatDateInProjectTimeZone,
  resolveProjectTimeZone,
};
