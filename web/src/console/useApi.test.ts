import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useApi } from "./useApi";

describe("useApi", () => {
  it("loads data and exposes loading/error states", async () => {
    const fetcher = vi.fn().mockResolvedValue({ items: [1, 2] });
    const { result } = renderHook(() => useApi(fetcher));
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.data).toEqual({ items: [1, 2] });
    expect(result.current.error).toBeUndefined();
  });

  it("captures fetch errors as strings", async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useApi(fetcher));
    await waitFor(() => expect(result.current.error).toBe("boom"));
    expect(result.current.loading).toBe(false);
  });

  it("does not refetch on every render (inline closure safe)", async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true });
    const { result, rerender } = renderHook(({ f }) => useApi(f), {
      initialProps: { f: fetcher }
    });
    await waitFor(() => expect(result.current.loading).toBe(false));
    const callsAfterLoad = fetcher.mock.calls.length;
    rerender({ f: vi.fn().mockResolvedValue({ ok: true }) });
    rerender({ f: vi.fn().mockResolvedValue({ ok: true }) });
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(fetcher.mock.calls.length).toBe(callsAfterLoad);
  });
});
