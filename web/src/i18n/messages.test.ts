import { describe, it, expect } from "vitest";
import { messages } from "./messages";

describe("landing i18n keys", () => {
  it("provides the new menu and hero panel keys in both locales", () => {
    for (const locale of ["en", "vi"] as const) {
      expect(messages[locale]["landing.nav.menu"].length).toBeGreaterThan(0);
      expect(messages[locale]["landing.nav.close"].length).toBeGreaterThan(0);
      expect(messages[locale]["landing.hero.panelLabel"].length).toBeGreaterThan(0);
      expect(messages[locale]["landing.hero.callout"].length).toBeGreaterThan(0);
    }
  });
});
