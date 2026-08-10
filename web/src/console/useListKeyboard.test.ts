import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useListKeyboard } from "./useListKeyboard";

const rows = ["a", "b", "c"];

function fire(handler: (event: React.KeyboardEvent) => void, key: string) {
  const event = { key, preventDefault: vi.fn() } as unknown as React.KeyboardEvent;
  act(() => handler(event));
  return event;
}

describe("useListKeyboard", () => {
  it("moves selection down with j and clamps at the last row", () => {
    const onOpen = vi.fn();
    const { result } = renderHook(() => useListKeyboard({ rows, onOpen }));
    fire(result.current.onKeyDown, "j");
    expect(result.current.selectedIndex).toBe(0);
    fire(result.current.onKeyDown, "j");
    fire(result.current.onKeyDown, "j");
    expect(result.current.selectedIndex).toBe(2);
    fire(result.current.onKeyDown, "j");
    expect(result.current.selectedIndex).toBe(0); // wraps
  });

  it("moves selection up with k", () => {
    const onOpen = vi.fn();
    const { result } = renderHook(() => useListKeyboard({ rows, onOpen }));
    act(() => result.current.setSelectedIndex(2));
    fire(result.current.onKeyDown, "k");
    expect(result.current.selectedIndex).toBe(1);
  });

  it("opens the selected row on Enter", () => {
    const onOpen = vi.fn();
    const { result } = renderHook(() => useListKeyboard({ rows, onOpen }));
    act(() => result.current.setSelectedIndex(1));
    const event = fire(result.current.onKeyDown, "Enter");
    expect(event.preventDefault).toHaveBeenCalled();
    expect(onOpen).toHaveBeenCalledWith("b", 1);
  });

  it("does not open when nothing is selected", () => {
    const onOpen = vi.fn();
    const { result } = renderHook(() => useListKeyboard({ rows, onOpen }));
    fire(result.current.onKeyDown, "Enter");
    expect(onOpen).not.toHaveBeenCalled();
  });

  it("clears selection on Escape", () => {
    const onOpen = vi.fn();
    const { result } = renderHook(() => useListKeyboard({ rows, onOpen }));
    act(() => result.current.setSelectedIndex(2));
    fire(result.current.onKeyDown, "Escape");
    expect(result.current.selectedIndex).toBe(-1);
  });

  it("does nothing when the list is empty", () => {
    const onOpen = vi.fn();
    const { result } = renderHook(() => useListKeyboard({ rows: [], onOpen }));
    const event = fire(result.current.onKeyDown, "j");
    expect(event.preventDefault).not.toHaveBeenCalled();
  });

  it("resets selection when rows change", () => {
    const onOpen = vi.fn();
    const { result, rerender } = renderHook(
      ({ list, open }: { list: string[]; open: typeof onOpen }) =>
        useListKeyboard({ rows: list, onOpen: open }),
      { initialProps: { list: rows, open: onOpen } },
    );
    act(() => result.current.setSelectedIndex(1));
    act(() => rerender({ list: ["x"], open: onOpen }));
    expect(result.current.selectedIndex).toBe(-1);
  });
});
