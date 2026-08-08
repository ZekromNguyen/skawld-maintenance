import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Tabs } from "./Tabs";

describe("Tabs", () => {
  it("renders tabs with correct roles and selected state", () => {
    render(
      <Tabs
        label="filter"
        tabs={[{ id: "a", label: "A" }, { id: "b", label: "B" }]}
        active="a"
        onChange={() => {}}
      />,
    );
    expect(screen.getByRole("tablist", { name: "filter" })).toBeTruthy();
    expect(screen.getByRole("tab", { name: "A" }).getAttribute("aria-selected")).toBe("true");
    expect(screen.getByRole("tab", { name: "B" }).getAttribute("aria-selected")).toBe("false");
  });

  it("moves focus with arrow keys", () => {
    const onChange = (() => {}) as (id: string) => void;
    render(
      <Tabs
        label="filter"
        tabs={[{ id: "a", label: "A" }, { id: "b", label: "B" }, { id: "c", label: "C" }]}
        active="a"
        onChange={onChange}
      />,
    );
    const first = screen.getByRole("tab", { name: "A" });
    first.focus();
    fireEvent.keyDown(first, { key: "ArrowRight" });
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "B" }));
    fireEvent.keyDown(screen.getByRole("tab", { name: "B" }), { key: "ArrowLeft" });
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "A" }));
  });

  it("calls onChange when a tab is activated by key", () => {
    const calls: string[] = [];
    render(
      <Tabs
        label="filter"
        tabs={[{ id: "a", label: "A" }, { id: "b", label: "B" }]}
        active="a"
        onChange={(id) => calls.push(id)}
      />,
    );
    const first = screen.getByRole("tab", { name: "A" });
    first.focus();
    fireEvent.keyDown(first, { key: "ArrowRight" });
    expect(calls).toEqual(["b"]);
  });
});

describe("Tabs keyboard edges", () => {
  it("wraps around at the ends and supports Home/End", () => {
    const onChange = (() => {}) as (id: string) => void;
    render(
      <Tabs
        label="edge"
        tabs={[{ id: "a", label: "A" }, { id: "b", label: "B" }, { id: "c", label: "C" }]}
        active="a"
        onChange={onChange}
      />,
    );
    const first = screen.getByRole("tab", { name: "A" });
    first.focus();
    fireEvent.keyDown(first, { key: "ArrowLeft" });
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "C" }));
    fireEvent.keyDown(screen.getByRole("tab", { name: "C" }), { key: "End" });
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "C" }));
    fireEvent.keyDown(screen.getByRole("tab", { name: "C" }), { key: "Home" });
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "A" }));
  });

  it("activates the current tab on Enter", () => {
    const calls: string[] = [];
    render(
      <Tabs
        label="enter"
        tabs={[{ id: "a", label: "A" }, { id: "b", label: "B" }]}
        active="a"
        onChange={(id) => calls.push(id)}
      />,
    );
    const first = screen.getByRole("tab", { name: "A" });
    first.focus();
    fireEvent.keyDown(first, { key: "Enter" });
    expect(calls).toEqual(["a"]);
  });
});
