import { describe, it, expect } from "vitest";
import { structureDescription } from "./structureDescription";

const labels = {
  what: "What happened",
  observations: "Observations",
  impact: "Impact",
  actions: "Actions taken",
};

const assets = [
  { id: "a1", tag: "P-302" },
  { id: "a2", tag: "P-304" },
];

describe("structureDescription", () => {
  it("turns the first sentence into a summary", () => {
    const result = structureDescription("P-302 pump vibrating loudly since 09:00.", assets, labels);
    expect(result.summary).toBe("P-302 pump vibrating loudly since 09:00");
    expect(result.details).toContain("What happened: P-302 pump vibrating loudly since 09:00.");
  });

  it("suggests CRITICAL priority from urgency keywords", () => {
    const result = structureDescription("Emergency: pump down, production halted.", assets, labels);
    expect(result.priority).toBe("CRITICAL");
  });

  it("suggests HIGH priority from vibration keywords", () => {
    const result = structureDescription("High vibration on the bearing, very loud.", assets, labels);
    expect(result.priority).toBe("HIGH");
  });

  it("returns an empty priority when nothing matches", () => {
    const result = structureDescription("Inspected the pump and found it fine.", assets, labels);
    expect(result.priority).toBe("");
  });

  it("matches the asset tag mentioned in the text", () => {
    const result = structureDescription("P-304 bearing temperature rising.", assets, labels);
    expect(result.assetId).toBe("a2");
  });

  it("groups observations, impact and actions into sections", () => {
    const result = structureDescription(
      "P-302 pump vibrating loudly. Bearing temperature reading 95 Celsius. Production line stopped for 2 hours. Called the electrician and switched to standby pump.",
      assets,
      labels,
    );
    expect(result.details).toContain("What happened: P-302 pump vibrating loudly.");
    expect(result.details).toContain("Observations: Bearing temperature reading 95 Celsius.");
    expect(result.details).toContain("Impact: Production line stopped for 2 hours.");
    expect(result.details).toContain("Actions taken: Called the electrician and switched to standby pump.");
  });

  it("truncates long summaries", () => {
    const long = `${"The bearing housing on the outboard side of the main circulation pump ".repeat(3)}.`;
    const result = structureDescription(long, assets, labels);
    expect(result.summary.length).toBeLessThanOrEqual(90);
    expect(result.summary.endsWith("…")).toBe(true);
  });
});
