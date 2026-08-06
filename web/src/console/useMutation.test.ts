import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { useMutation } from "./useMutation";

describe("useMutation", () => {
  it("runs the action and reports pending", async () => {
    const action = vi
      .fn<(...args: number[]) => Promise<number>>()
      .mockResolvedValue(42);
    const { result } = renderHook(() => useMutation(action));
    let resolved: number | undefined;
    act(() => {
      void result.current.run(1).then((v) => {
        resolved = v;
      });
    });
    expect(result.current.pending).toBe(true);
    await waitFor(() => expect(result.current.pending).toBe(false));
    expect(action).toHaveBeenCalledWith(1);
    expect(resolved).toBe(42);
  });

  it("captures the error instead of throwing", async () => {
    const action = vi
      .fn<() => Promise<number>>()
      .mockRejectedValue(new Error("nope"));
    const { result } = renderHook(() => useMutation(action));
    await act(async () => {
      await result.current.run();
    });
    expect(result.current.error).toBe("nope");
    expect(result.current.pending).toBe(false);
  });
});
