// @ts-check

const fs = require("fs");
const os = require("os");
const path = require("path");
const { mergeAliases, mergeModelAliasesInConfig } = require("./merge_model_aliases.cjs");

describe("merge_model_aliases", () => {
  test("mergeAliases overlays delta aliases on top of base aliases", () => {
    const merged = mergeAliases(
      {
        sonnet: ["copilot/*sonnet*"],
        auto: ["sonnet"],
      },
      {
        sonnet: ["custom/sonnet"],
        "custom-alias": ["custom/model"],
      }
    );

    expect(merged).toEqual({
      sonnet: ["custom/sonnet"],
      auto: ["sonnet"],
      "custom-alias": ["custom/model"],
    });
  });

  test("mergeModelAliasesInConfig merges delta into config apiProxy.models", () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "merge-model-aliases-"));
    const configPath = path.join(tmpDir, "awf-config.json");
    const aliasesPath = path.join(tmpDir, "model_aliases.json");

    fs.writeFileSync(
      aliasesPath,
      JSON.stringify({
        aliases: {
          sonnet: ["copilot/*sonnet*"],
          auto: ["sonnet"],
        },
      }),
      "utf8"
    );

    fs.writeFileSync(
      configPath,
      JSON.stringify({
        apiProxy: {
          enabled: true,
          models: {
            sonnet: ["custom/sonnet"],
            "custom-alias": ["custom/model"],
          },
        },
      }),
      "utf8"
    );

    const changed = mergeModelAliasesInConfig(configPath, aliasesPath);
    expect(changed).toBe(true);

    const parsed = JSON.parse(fs.readFileSync(configPath, "utf8"));
    expect(parsed.apiProxy.models).toEqual({
      sonnet: ["custom/sonnet"],
      auto: ["sonnet"],
      "custom-alias": ["custom/model"],
    });
  });

  test("mergeModelAliasesInConfig is a no-op when models delta is absent", () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "merge-model-aliases-"));
    const configPath = path.join(tmpDir, "awf-config.json");
    const aliasesPath = path.join(tmpDir, "model_aliases.json");

    fs.writeFileSync(aliasesPath, JSON.stringify({ aliases: { sonnet: ["copilot/*sonnet*"] } }), "utf8");
    fs.writeFileSync(configPath, JSON.stringify({ apiProxy: { enabled: true } }), "utf8");

    const changed = mergeModelAliasesInConfig(configPath, aliasesPath);
    expect(changed).toBe(false);
  });
});
