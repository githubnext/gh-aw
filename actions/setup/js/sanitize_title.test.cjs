// @ts-check
import { describe, it, expect } from "vitest";

const { sanitizeTitle, applyTitlePrefix } = require("./sanitize_title.cjs");

describe("sanitize_title", () => {
  describe("sanitizeTitle", () => {
    describe("basic sanitization", () => {
      it("should return empty string for null/undefined", () => {
        expect(sanitizeTitle(null)).toBe("");
        expect(sanitizeTitle(undefined)).toBe("");
        expect(sanitizeTitle("")).toBe("");
      });

      it("should trim whitespace", () => {
        expect(sanitizeTitle("  Test Title  ")).toBe("Test Title");
        expect(sanitizeTitle("\n\tTest Title\n\t")).toBe("Test Title");
      });

      it("should handle normal ASCII titles", () => {
        expect(sanitizeTitle("Bug Report")).toBe("Bug Report");
        expect(sanitizeTitle("Feature Request: Add new feature")).toBe("Feature Request: Add new feature");
      });
    });

    describe("Unicode security hardening", () => {
      it("should apply NFC normalization", () => {
        // Composed vs decomposed é (U+00E9 vs U+0065 U+0301)
        const composed = "café";
        const decomposed = "café"; // Using combining character
        expect(sanitizeTitle(composed)).toBe(sanitizeTitle(decomposed));
      });

      it("should strip zero-width characters", () => {
        expect(sanitizeTitle("Test\u200BTitle")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u200CTitle")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u200DTitle")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u2060Title")).toBe("TestTitle");
        expect(sanitizeTitle("Test\uFEFFTitle")).toBe("TestTitle");
      });

      it("should remove bidirectional override controls", () => {
        expect(sanitizeTitle("Test\u202ATitle")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u202BTitle")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u202CTitle")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u202DTitle")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u202ETitle")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u2066Title")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u2067Title")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u2068Title")).toBe("TestTitle");
        expect(sanitizeTitle("Test\u2069Title")).toBe("TestTitle");
      });

      it("should convert fullwidth ASCII to standard ASCII", () => {
        // Fullwidth brackets
        expect(sanitizeTitle("［Test］")).toBe("[Test]");
        // Fullwidth letters and numbers; NFKC normalization also converts fullwidth space U+3000 to regular space U+0020
        expect(sanitizeTitle("Ｔｅｓｔ　１２３")).toBe("Test 123");
        // Mix of fullwidth and normal
        expect(sanitizeTitle("Test［Ａｇｅｎｔ］")).toBe("Test[Agent]");
      });

      it("should handle complex Unicode attacks", () => {
        // Combining zero-width spaces with bidirectional overrides
        const malicious = "Test\u200B\u202ETi\u200Ctle\u202C\uFEFF";
        expect(sanitizeTitle(malicious)).toBe("TestTitle");
      });
    });

    describe("duplicate prefix removal", () => {
      it("should remove exact prefix match", () => {
        expect(sanitizeTitle("[Agent] Fix bug", "[Agent] ")).toBe("Fix bug");
        expect(sanitizeTitle("🤖 Fix bug", "🤖 ")).toBe("Fix bug");
        expect(sanitizeTitle("[WIP] Update docs", "[WIP] ")).toBe("Update docs");
      });

      it("should remove prefix with colon separator", () => {
        expect(sanitizeTitle("[Agent]: Fix bug", "[Agent] ")).toBe("Fix bug");
        expect(sanitizeTitle("Agent: Fix bug", "Agent ")).toBe("Fix bug");
      });

      it("should remove prefix with dash separator", () => {
        expect(sanitizeTitle("[Agent] - Fix bug", "[Agent] ")).toBe("Fix bug");
        expect(sanitizeTitle("Agent - Fix bug", "Agent ")).toBe("Fix bug");
      });

      it("should remove prefix with pipe separator", () => {
        expect(sanitizeTitle("[Agent] | Fix bug", "[Agent] ")).toBe("Fix bug");
        expect(sanitizeTitle("Agent | Fix bug", "Agent ")).toBe("Fix bug");
      });

      it("should not remove prefix from middle of title", () => {
        expect(sanitizeTitle("Fix [Agent] bug", "[Agent] ")).toBe("Fix [Agent] bug");
        expect(sanitizeTitle("Update Agent feature", "Agent ")).toBe("Update Agent feature");
      });

      it("should handle case-sensitive prefix matching", () => {
        expect(sanitizeTitle("[AGENT] Fix bug", "[Agent] ")).toBe("[AGENT] Fix bug");
        expect(sanitizeTitle("[agent] Fix bug", "[Agent] ")).toBe("[agent] Fix bug");
      });

      it("should handle empty or whitespace-only prefix", () => {
        expect(sanitizeTitle("Test Title", "")).toBe("Test Title");
        expect(sanitizeTitle("Test Title", "   ")).toBe("Test Title");
      });

      it("should prevent double prefix from agent confusion", () => {
        // Agent might generate "[Agent] [Agent] Title" if confused
        // sanitizeTitle removes all instances of the prefix, so both get removed
        expect(sanitizeTitle("[Agent] [Agent] Fix bug", "[Agent] ")).toBe("Fix bug");
      });

      it("should handle Unicode in prefixes", () => {
        // sanitizeTitle removes all instances of the prefix
        expect(sanitizeTitle("🤖 🤖 Fix bug", "🤖 ")).toBe("Fix bug");
        // After hardening, fullwidth brackets become regular brackets, then get removed as prefix
        expect(sanitizeTitle("［Agent］Fix bug", "[Agent] ")).toBe("Fix bug");
      });
    });

    describe("combined sanitization and prefix removal", () => {
      it("should apply Unicode hardening before prefix removal", () => {
        const title = "［Agent］\u200BFix\u202Ebug\u202C";
        // After hardening: "[Agent]Fixbug", then "[Agent]" prefix removed (no space), leaving "Fixbug"
        expect(sanitizeTitle(title, "[Agent] ")).toBe("Fixbug");
      });

      it("should handle malicious prefix injection attempts", () => {
        // Attacker tries to inject prefix with invisible characters
        // After hardening: "[Agent] [Agent] Fix bug", then both prefixes get removed
        const title = "[Agent]\u200B\u202A [Agent]\u202C Fix bug";
        expect(sanitizeTitle(title, "[Agent] ")).toBe("Fix bug");
      });

      it("should preserve legitimate content after sanitization", () => {
        const title = "[Agent] Fix bug #123: Update configuration";
        expect(sanitizeTitle(title, "[Agent] ")).toBe("Fix bug #123: Update configuration");
      });
    });
  });

  describe("applyTitlePrefix", () => {
    it("should add prefix to clean title", () => {
      expect(applyTitlePrefix("Fix bug", "[Agent] ")).toBe("[Agent] Fix bug");
      expect(applyTitlePrefix("Update docs", "🤖 ")).toBe("🤖 Update docs");
    });

    it("should not duplicate prefix if already present", () => {
      expect(applyTitlePrefix("[Agent] Fix bug", "[Agent] ")).toBe("[Agent] Fix bug");
      expect(applyTitlePrefix("🤖 Fix bug", "🤖 ")).toBe("🤖 Fix bug");
    });

    it("should handle empty prefix", () => {
      expect(applyTitlePrefix("Fix bug", "")).toBe("Fix bug");
      expect(applyTitlePrefix("Fix bug", "   ")).toBe("Fix bug");
    });

    it("should trim inputs", () => {
      // applyTitlePrefix should use titlePrefix as-is, but the title is trimmed
      // When prefix ends with ], space is added automatically
      expect(applyTitlePrefix("  Fix bug  ", "  [Agent]  ")).toBe("  [Agent]  Fix bug");
      // When prefix ends with ], space is added even if prefix has leading spaces
      expect(applyTitlePrefix("  Fix bug  ", "  [Agent]")).toBe("  [Agent] Fix bug");
    });

    it("should handle empty title", () => {
      expect(applyTitlePrefix("", "[Agent] ")).toBe("");
      expect(applyTitlePrefix("   ", "[Agent] ")).toBe("");
    });

    it("should add space after prefix ending with ]", () => {
      expect(applyTitlePrefix("Fix bug", "[Agent]")).toBe("[Agent] Fix bug");
      expect(applyTitlePrefix("Update docs", "[WIP]")).toBe("[WIP] Update docs");
      expect(applyTitlePrefix("Contribution Check", "[Contribution Check Report]")).toBe("[Contribution Check Report] Contribution Check");
    });

    it("should add space after prefix ending with -", () => {
      expect(applyTitlePrefix("Fix bug", "Agent-")).toBe("Agent- Fix bug");
      expect(applyTitlePrefix("Update docs", "WIP-")).toBe("WIP- Update docs");
    });

    it("should not add extra space if prefix already has trailing space", () => {
      expect(applyTitlePrefix("Fix bug", "[Agent] ")).toBe("[Agent] Fix bug");
      expect(applyTitlePrefix("Update docs", "Agent- ")).toBe("Agent- Update docs");
    });

    it("should not add space if prefix ends with other characters", () => {
      expect(applyTitlePrefix("Fix bug", "Agent:")).toBe("Agent:Fix bug");
      expect(applyTitlePrefix("Update docs", "🤖")).toBe("🤖Update docs");
    });
  });

  describe("integration scenarios", () => {
    it("should handle typical workflow: sanitize then apply prefix", () => {
      const rawTitle = "［Agent］\u200BFix\u202Ebug #123\u202C";
      const sanitized = sanitizeTitle(rawTitle, "[Agent] ");
      const final = applyTitlePrefix(sanitized, "[Agent] ");
      // After sanitization, prefix is removed, so we need to apply it again
      expect(final).toBe("[Agent] Fixbug #123");
    });

    it("should prevent agent-generated duplicate prefixes", () => {
      // Agent generates title with prefix already included
      const agentTitle = "[Agent] Update configuration";
      const sanitized = sanitizeTitle(agentTitle, "[Agent] ");
      const final = applyTitlePrefix(sanitized, "[Agent] ");
      // Prefix gets removed during sanitization, then re-applied
      expect(final).toBe("[Agent] Update configuration");
    });

    it("should handle fullwidth brackets in agent output", () => {
      // Agent uses fullwidth brackets (common in some locales)
      const agentTitle = "［Agent］Fix critical bug";
      const sanitized = sanitizeTitle(agentTitle, "[Agent] ");
      const final = applyTitlePrefix(sanitized, "[Agent] ");
      // Fullwidth brackets get converted to ASCII "[Agent]Fix", then "[Agent]" removed leaving "Fix critical bug"
      expect(final).toBe("[Agent] Fix critical bug");
    });
  });

  describe("content sanitization via sanitizeContent", () => {
    it("should escape @mentions in titles", () => {
      expect(sanitizeTitle("Fix bug for @username")).toBe("Fix bug for `@username`");
      expect(sanitizeTitle("@user please review")).toBe("`@user` please review");
    });

    it("should sanitize URLs with disallowed protocols", () => {
      expect(sanitizeTitle("Click javascript:alert(1)")).toBe("Click (redacted)");
      expect(sanitizeTitle("Visit data:text/html")).toBe("Visit (redacted)");
    });

    it("should preserve normal text content", () => {
      expect(sanitizeTitle("Fix bug #123: Update configuration")).toBe("Fix bug #123: Update configuration");
      expect(sanitizeTitle("Feature: Add new feature")).toBe("Feature: Add new feature");
    });

    it("should apply content sanitization pipeline", () => {
      // Unicode hardening (zero-width removed), @mention escaping, URL sanitization
      const title = "@user Test\u200Btitle with javascript:alert(1)";
      const sanitized = sanitizeTitle(title);
      expect(sanitized).toBe("`@user` Testtitle with (redacted)");
    });

    it("should decode &gt; in title to prevent literal &gt; appearing in issues", () => {
      // If an agent outputs a title with &gt; (e.g. because the prompt context contained it),
      // the sanitizer must decode it back to > so the issue title is not &gt; in markdown.
      expect(sanitizeTitle("value &gt; threshold")).toBe("value > threshold");
    });

    it("should decode double-encoded &amp;gt; in title to >", () => {
      expect(sanitizeTitle("value &amp;gt; threshold")).toBe("value > threshold");
    });

    it("should decode &lt; in title and neutralize any resulting HTML tags", () => {
      // &lt;tag&gt; → <tag> → convertXmlTags → (tag)
      expect(sanitizeTitle("&lt;script&gt; injection")).toBe("(script) injection");
    });

    it("should decode &amp; in title to &", () => {
      expect(sanitizeTitle("cats &amp; dogs")).toBe("cats & dogs");
    });

    it("should be idempotent - sanitizing a title with > twice gives same result", () => {
      const title = "Fix bug: value > 5";
      const once = sanitizeTitle(title);
      const twice = sanitizeTitle(once);
      expect(once).toBe("Fix bug: value > 5");
      expect(twice).toBe(once);
    });

    it("should be idempotent - sanitizing &gt; title twice should not produce &gt;", () => {
      // Simulates agent outputting &gt; in title because prompt context had HTML-encoded >
      const title = "Fix bug: value &gt; 5";
      const once = sanitizeTitle(title);
      const twice = sanitizeTitle(once);
      expect(once).not.toContain("&gt;");
      expect(once).toBe("Fix bug: value > 5");
      // Idempotency: a second pass on the decoded result should not re-introduce &gt;
      expect(twice).toBe(once);
    });
  });
});
