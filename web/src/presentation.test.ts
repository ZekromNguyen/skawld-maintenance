import { describe, expect, it } from "vitest";
import { relativeTime, severityTone } from "./presentation";

describe("maintenance presentation", () => {
  it("keeps critical incidents visually distinct", () => {
    expect(severityTone("CRITICAL")).toBe("critical");
    expect(severityTone("LOW")).toBe("low");
  });

  it("shows deterministic elapsed time", () => {
    const t = Date.parse("2026-07-26T02:00:00Z");
    expect(relativeTime("2026-07-26T00:00:00Z", "en", t)).toBe("2h ago");
    expect(relativeTime("2026-07-26T00:00:00Z", "vi", t)).toBe("2 giờ trước");
  });
});
