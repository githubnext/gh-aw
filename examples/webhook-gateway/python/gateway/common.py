"""Closed errors, strict JSON, and shared scalar/URL rules."""

from __future__ import annotations

import json
import re
from urllib.parse import urlsplit, urlunsplit

MAX_SAFE_INTEGER = 9007199254740991
MAX_BYTES = 65535


class Rejected(Exception):
    """Only constant, credential-free messages may cross a trust boundary."""

    def __init__(self, code, status=400):
        super().__init__(code)
        self.code = code
        self.status = status


def require(condition, code="invalid_input", status=400):
    if not condition:
        raise Rejected(code, status)


def integer(value, positive=False):
    return (
        type(value) is int
        and abs(value) <= MAX_SAFE_INTEGER
        and (not positive or value > 0)
    )


def text(value):
    return type(value) is str and bool(value.strip())


def scalar(value, kind):
    return (
        integer(value) if kind == "integer"
        else type(value) is bool if kind == "boolean"
        else type(value) is str if kind == "string"
        else False
    )


def encode(value):
    try:
        return json.dumps(
            value, ensure_ascii=False, separators=(",", ":"), allow_nan=False
        ).encode("utf-8")
    except (UnicodeError, ValueError, TypeError, RecursionError):
        raise Rejected("invalid_json") from None


def decode(raw):
    def pairs(entries):
        result = {}
        for key, value in entries:
            require(key not in result, "duplicate_json_key")
            require(key not in {"__proto__", "prototype", "constructor"}, "unsafe_json_key")
            result[key] = value
        return result

    def constant(_):
        raise Rejected("nonfinite_json_number")

    try:
        value = json.loads(raw.decode("utf-8"), object_pairs_hook=pairs, parse_constant=constant)
        encode(value)
        return value
    except (UnicodeError, ValueError, TypeError, RecursionError):
        raise Rejected("invalid_json") from None


def values_at(value, path):
    if not path:
        return [value] if value is not None else []
    key, rest = path[0], path[1:]
    if key == "*":
        return [v for item in value for v in values_at(item, rest)] if type(value) is list else []
    if type(value) is list and key.isdigit():
        return values_at(value[int(key)], rest) if int(key) < len(value) else []
    return values_at(value[key], rest) if type(value) is dict and key in value else []


def at(value, path):
    result = values_at(value, path)
    return result[0] if result else None


def object_at(value, path):
    result = at(value, path)
    return result if type(result) is dict else {}


def canonical_url(value, fragment=False):
    require(text(value), "invalid_resource_url")
    try:
        url = urlsplit(value)
        authority = (url.hostname or "") + (":" + str(url.port) if url.port is not None else "")
        require(
            url.scheme == "https" and url.hostname and not url.username and not url.password
            and not url.query and (fragment or not url.fragment)
            and not re.search(r"[\x00-\x20\x7f\\]", value)
            and re.fullmatch(r"[a-z0-9.-]+", url.hostname)
            and url.netloc == authority and url.port != 443
            and urlunsplit(url) == value
            and all(segment not in {".", ".."} for segment in url.path.split("/"))
            and not re.search(r"%(?:2e|2f|5c)", url.path, re.I),
            "invalid_resource_url",
        )
        url.netloc.encode("ascii")
        url.path.encode("ascii")
        return url
    except (ValueError, UnicodeError):
        raise Rejected("invalid_resource_url") from None


def base_url(value):
    url = canonical_url(value)
    require(not url.fragment and not value.endswith("/"), "invalid_base_url")
    return value


def decimal(value):
    require(type(value) is str and re.fullmatch(r"[1-9][0-9]*", value), "invalid_resource_id")
    return value


def positive(value):
    require(integer(value, positive=True), "unsafe_integer_id")
    return str(value)
