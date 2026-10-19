import { createHash, timingSafeEqual } from "node:crypto";

export class GatewayError extends Error {
  constructor(code, status = 400) {
    super(code);
    this.code = code;
    this.status = status;
  }
}

export function requireThat(condition, code, status = 400) {
  if (!condition) throw new GatewayError(code, status);
}

export const object = value => value !== null && typeof value === "object" && !Array.isArray(value);
export const nonempty = value => typeof value === "string" && value.trim().length > 0;
export const hash = value => createHash("sha256").update(value).digest("hex");
export const equalSecret = (left, right) => timingSafeEqual(createHash("sha256").update(left).digest(), createHash("sha256").update(right).digest());
export const scalar = (value, type) => (type === "integer" ? Number.isSafeInteger(value) : ["string", "boolean"].includes(type) && typeof value === type);

export function shape(value, required, optional = []) {
  requireThat(object(value) && required.every(key => Object.hasOwn(value, key)) && Object.keys(value).every(key => [...required, ...optional].includes(key)), "invalid-closed-shape");
}

export function canonical(value) {
  if (Array.isArray(value)) return `[${value.map(canonical).join(",")}]`;
  if (object(value))
    return `{${Object.keys(value)
      .sort()
      .map(key => `${JSON.stringify(key)}:${canonical(value[key])}`)
      .join(",")}}`;
  return JSON.stringify(value);
}

export function decimal(value) {
  requireThat(typeof value === "string" && /^[1-9][0-9]*$/.test(value), "invalid-decimal-id");
  return value;
}

export function numericId(value) {
  requireThat(Number.isSafeInteger(value) && value > 0, "unsafe-numeric-id");
  return String(value);
}

export function httpsURL(value) {
  requireThat(nonempty(value), "invalid-resource-url");
  let url;
  try {
    url = new URL(value);
  } catch {
    throw new GatewayError("invalid-resource-url");
  }
  requireThat(url.protocol === "https:" && !url.username && !url.password && !value.includes("?") && !value.endsWith("#") && url.href === value, "invalid-resource-url");
  return url;
}

export function valuesAt(value, path) {
  if (!path.length) return value === undefined || value === null ? [] : [value];
  const [key, ...tail] = path;
  if (key === "*") return Array.isArray(value) ? value.flatMap(item => valuesAt(item, tail)) : [];
  return object(value) || Array.isArray(value) ? (Object.hasOwn(value, key) ? valuesAt(value[key], tail) : []) : [];
}

// Reject duplicate keys before any authorization decision, including nested keys.
// JSON.parse alone silently keeps the last duplicate and rounds oversized IDs.
export function parseJSON(bytes) {
  let text;
  try {
    text = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    throw new GatewayError("invalid-utf8");
  }
  let offset = 0;
  const whitespace = () => {
    while (/[ \t\r\n]/.test(text[offset] ?? "\0")) offset++;
  };
  function string() {
    const start = offset++;
    while (offset < text.length) {
      if (text[offset] === "\\") {
        offset += 2;
        continue;
      }
      if (text[offset++] === '"') {
        try {
          return JSON.parse(text.slice(start, offset));
        } catch {
          break;
        }
      }
    }
    throw new GatewayError("invalid-json");
  }
  function read(depth) {
    requireThat(depth <= 64, "json-too-deep");
    whitespace();
    if (text[offset] === '"') return string();
    if (text[offset] === "{" || text[offset] === "[") {
      const array = text[offset++] === "[";
      const result = array ? [] : Object.create(null);
      const end = array ? "]" : "}";
      whitespace();
      if (text[offset] === end) {
        offset++;
        return result;
      }
      while (offset < text.length) {
        whitespace();
        if (array) result.push(read(depth + 1));
        else {
          requireThat(text[offset] === '"', "invalid-json");
          const key = string();
          requireThat(!Object.hasOwn(result, key) && !["__proto__", "constructor", "prototype"].includes(key), "duplicate-or-unsafe-json-key");
          whitespace();
          requireThat(text[offset++] === ":", "invalid-json");
          result[key] = read(depth + 1);
        }
        whitespace();
        if (text[offset] === end) {
          offset++;
          return result;
        }
        requireThat(text[offset++] === ",", "invalid-json");
      }
      throw new GatewayError("invalid-json");
    }
    const match = text.slice(offset).match(/^(?:true|false|null|-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?)/);
    requireThat(match, "invalid-json");
    offset += match[0].length;
    const value = JSON.parse(match[0]);
    requireThat(typeof value !== "number" || (Number.isFinite(value) && Math.abs(value) <= Number.MAX_SAFE_INTEGER && (!Number.isInteger(value) || Number.isSafeInteger(value))), "unsafe-json-number");
    if (typeof value === "number" && Number.isInteger(value)) {
      const parts = match[0].match(/^(-?)([0-9]+)(?:\.([0-9]+))?(?:[eE]([+-]?[0-9]+))?$/);
      const fraction = parts[3] ?? "";
      const power = Number(parts[4] ?? 0) - fraction.length;
      requireThat(Math.abs(power) <= 100, "unsafe-json-number");
      const digits = BigInt(`${parts[1]}${parts[2]}${fraction}`);
      const factor = 10n ** BigInt(Math.abs(power));
      requireThat(power >= 0 ? digits * factor === BigInt(value) : digits === BigInt(value) * factor, "rounded-json-integer");
    }
    return value;
  }
  const result = read(0);
  whitespace();
  requireThat(offset === text.length, "invalid-json");
  return result;
}

export function envelope(subscription, deliveryId, payload) {
  const client_payload = { schema_version: 2, aw_context: { webhook: { event: subscription.event, deliveries: [{ delivery_id: deliveryId, payload }] } } };
  const request = { event_type: subscription.dispatch.event_type, client_payload };
  requireThat(Buffer.byteLength(JSON.stringify(client_payload), "utf8") <= 65535 && Buffer.byteLength(JSON.stringify(request), "utf8") <= 65535, "dispatch-too-large", 413);
  return request;
}
