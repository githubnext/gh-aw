// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";

const req = createRequire(import.meta.url);

// Set up global.core mock before loading any module so shim.cjs does not
// overwrite it with its stderr-based fallback.
const mockCore = {
  info: vi.fn(),
  debug: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
};
global.core = mockCore;

// Load safe_output_handler_manager first so we can patch its exports through
// the shared CJS module cache before process_safe_outputs.cjs is loaded.
const handlerManagerModule = req("./safe_output_handler_manager.cjs");
const originalHandlerMain = handlerManagerModule.main;

// Load the module under test after the dependency is in the cache.
const { main } = req("./process_safe_outputs.cjs");

describe("process_safe_outputs.cjs", () => {
  /** @type {ReturnType<typeof vi.fn>} */
  let mockHandlerMain;

  beforeEach(() => {
    vi.clearAllMocks();
    mockHandlerMain = vi.fn().mockResolvedValue(undefined);
    handlerManagerModule.main = mockHandlerMain;
  });

  afterEach(() => {
    handlerManagerModule.main = originalHandlerMain;
    vi.restoreAllMocks();
  });

  it("calls the safe output handler manager", async () => {
    await main();
    expect(mockHandlerMain).toHaveBeenCalledOnce();
  });

  it("shares activation links across handlers and items, but isolates independent invocations", async () => {
    const createPRModule = req("./create_pull_request.cjs");
    const pushModule = req("./push_to_pull_request_branch.cjs");
    const accumulators = [];
    const mockFactory = async config => {
      accumulators.push(config._activationCommentLinks);
      return async message => {
        config._activationCommentLinks.push(message.link);
        return { success: true };
      };
    };
    vi.spyOn(createPRModule, "main").mockImplementation(mockFactory);
    vi.spyOn(pushModule, "main").mockImplementation(mockFactory);
    mockHandlerMain.mockImplementation(async options => {
      const handlers = await handlerManagerModule.loadHandlers({ create_pull_request: {}, push_to_pull_request_branch: {} }, undefined, [], options.activationCommentLinks);
      await handlers.get("create_pull_request")({ link: "PR 1" });
      await handlers.get("create_pull_request")({ link: "PR 2" });
      await handlers.get("push_to_pull_request_branch")({ link: "Commit 3" });
      expect(options.activationCommentLinks).toEqual(["PR 1", "PR 2", "Commit 3"]);
    });

    await main();
    await main();

    expect(accumulators).toHaveLength(4);
    expect(accumulators[0]).toBe(accumulators[1]);
    expect(accumulators[2]).toBe(accumulators[3]);
    expect(accumulators[0]).not.toBe(accumulators[2]);
  });

  it("propagates handler rejection", async () => {
    mockHandlerMain.mockRejectedValue(new Error("handler failed"));
    await expect(main()).rejects.toThrow("handler failed");
  });

  it("does not create stdout or stderr log files", async () => {
    await main();
    expect(fs.existsSync("/tmp/gh-aw/process-safe-outputs.stdout.log")).toBe(false);
    expect(fs.existsSync("/tmp/gh-aw/process-safe-outputs.stderr.log")).toBe(false);
  });
});
