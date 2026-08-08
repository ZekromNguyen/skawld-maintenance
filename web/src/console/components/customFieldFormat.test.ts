import { describe, it, expect } from "vitest";
import { formatCustomValue } from "./customFieldFormat";

const selectField = {
  field_type: "SELECT" as const,
  config: { options: [{ label: "Zone A", value: "a" }, { label: "Zone B", value: "b" }] },
};

const multiField = {
  field_type: "MULTI_SELECT" as const,
  config: { options: [{ label: "Red", value: "r" }, { label: "Blue", value: "b" }] },
};

describe("formatCustomValue", () => {
  it("renders empty for nullish values", () => {
    expect(formatCustomValue(selectField, undefined)).toBe("");
    expect(formatCustomValue(selectField, null)).toBe("");
  });

  it("maps select values to labels", () => {
    expect(formatCustomValue(selectField, "a")).toBe("Zone A");
  });

  it("falls back to the raw value for unknown select options", () => {
    expect(formatCustomValue(selectField, "zzz")).toBe("zzz");
  });

  it("maps multi-select values to labels joined by commas", () => {
    expect(formatCustomValue(multiField, ["r", "b"])).toBe("Red, Blue");
  });

  it("falls back to raw values for unknown multi-select options", () => {
    expect(formatCustomValue(multiField, ["r", "x"])).toBe("Red, x");
  });

  it("stringifies plain values", () => {
    expect(formatCustomValue({ field_type: "TEXT", config: {} }, "PO-42")).toBe("PO-42");
    expect(formatCustomValue({ field_type: "NUMBER", config: {} }, 42)).toBe("42");
    expect(formatCustomValue({ field_type: "DATE", config: {} }, "2026-08-09")).toBe("2026-08-09");
  });

  it("joins non-multi arrays with commas", () => {
    expect(formatCustomValue({ field_type: "TEXT", config: {} }, ["a", "b"])).toBe("a, b");
  });
});
