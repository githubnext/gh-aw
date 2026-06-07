// @ts-check

function normalizeUTCOffset(utcOffset) {
  const trimmed = typeof utcOffset === "string" ? utcOffset.trim() : "";
  const match = trimmed.match(/^([+-])(\d{2}):(\d{2})$/);
  if (!match) {
    return "";
  }

  const [, sign, hours, minutes] = match;
  const hourValue = Number.parseInt(hours, 10);
  const minuteValue = Number.parseInt(minutes, 10);
  if (hourValue > 14 || minuteValue > 59 || (hourValue === 14 && minuteValue !== 0)) {
    return "";
  }

  return `${sign}${hours}:${minutes}`;
}

function parseUTCOffsetMinutes(utcOffset) {
  const normalized = normalizeUTCOffset(utcOffset);
  if (!normalized) {
    return Number.NaN;
  }

  const sign = normalized.startsWith("-") ? -1 : 1;
  const hours = Number.parseInt(normalized.slice(1, 3), 10);
  const minutes = Number.parseInt(normalized.slice(4, 6), 10);
  return sign * (hours * 60 + minutes);
}

function warn(message) {
  global.core?.warning?.(message);
}

function readCompiledProjectTimeZone() {
  const raw = process.env.GH_AW_PROJECT_UTC || "";
  if (!raw.trim()) {
    return "";
  }
  const utcOffset = normalizeUTCOffset(raw);
  if (!utcOffset) {
    warn(`Ignoring invalid GH_AW_PROJECT_UTC offset: ${raw.trim()}`);
    return "";
  }
  return utcOffset;
}

function readDefaultTimeZone() {
  const raw = process.env.GH_AW_DEFAULT_UTC || "";
  if (!raw.trim()) {
    return "";
  }
  const utcOffset = normalizeUTCOffset(raw);
  if (!utcOffset) {
    warn(`Ignoring invalid GH_AW_DEFAULT_UTC offset: ${raw.trim()}`);
    return "";
  }
  return utcOffset;
}

function resolveProjectTimeZone() {
  return readCompiledProjectTimeZone() || readDefaultTimeZone();
}

function formatDateInProjectTimeZone(date) {
  const utcOffset = resolveProjectTimeZone();
  if (!utcOffset) {
    return "";
  }

  const offsetMinutes = parseUTCOffsetMinutes(utcOffset);
  if (Number.isNaN(offsetMinutes)) {
    return "";
  }

  const shiftedDate = new Date(date.getTime() + offsetMinutes * 60 * 1000);
  const formatted = new Intl.DateTimeFormat("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZone: "UTC",
  }).format(shiftedDate);
  return `${formatted} UTC${utcOffset}`;
}

module.exports = {
  formatDateInProjectTimeZone,
  resolveProjectTimeZone,
};
