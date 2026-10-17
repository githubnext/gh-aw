import { describe, expect, it, vi } from "vitest";
import { registerTests } from "./work_queue_replay_checks.cjs";
import { parseTransactionRecords } from "./work_queue_replay.cjs";
registerTests({ describe, it });

describe("shared JSONL work-queue framing", () => {
  it("preserves strict decoding across LF and CRLF records", () => {
    expect(parseTransactionRecords('{"id":1}\r\n{"id":2}\n')).toEqual([{ id: 1 }, { id: 2 }]);
  });

  it("enforces byte limits before shared JSON decoding, including multibyte records", () => {
    for (const text of ["x".repeat(8 * 1024 * 1024), "é".repeat(4 * 1024 * 1024)]) {
      const parse = vi.spyOn(JSON, "parse");
      try {
        expect(() => parseTransactionRecords(`"${text}"\n`)).toThrow(expect.objectContaining({ code: "resource_limit" }));
        expect(parse).not.toHaveBeenCalled();
      } finally {
        parse.mockRestore();
      }
    }
  });

  it.each([
    ['{"id":1,"id":2}\n', "duplicate_key"],
    ['{"id":1,"\\u0069d":2}\n', "duplicate_key"],
    ['{"id":1.0}\n', "noncanonical_number"],
    ['{"id":1e0}\n', "noncanonical_number"],
    ['{"id":9007199254740992}\n', "noncanonical_number"],
    ['{"id":"\\ud800"}\n', "invalid_unicode"],
    ['\ufeff{"id":1}\n', "codec_invalid"],
    ['\u00a0{"id":1}\n', "codec_invalid"],
    ['{"id":}\n', "codec_invalid"],
    ["{}\n\n", "ledger_invalid"],
    ["{}\n \r\n", "ledger_invalid"],
    ["{}", "ledger_invalid"],
  ])("retains strict rejection of %j", (input, code) => {
    expect(() => parseTransactionRecords(input)).toThrow(expect.objectContaining({ code }));
  });
});
