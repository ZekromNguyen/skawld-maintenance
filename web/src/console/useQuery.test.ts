import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { useQuery } from "./useQuery";

describe("useQuery", () => {
  it("loads data and exposes it", async () => {
    const fetcher = vi.fn().mockResolvedValue("ok");
    const { result } = renderHook(() => useQuery(fetcher));
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.data).toBe("ok"));
    expect(result.current.loading).toBe(false);
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it("refetches when a dependency changes", async () => {
    const fetcher = vi.fn().mockResolvedValue("a");
    const { result, rerender } = renderHook(({ dep }) => useQuery(fetcher, [dep]), {
      initialProps: { dep: "x" },
    });
    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(1));
    fetcher.mockResolvedValue("b");
    rerender({ dep: "y" });
    await waitFor(() => expect(result.current.data).toBe("b"));
    expect(fetcher).toHaveBeenCalledTimes(2);
  });

  it("drops stale responses (last request wins)", async () => {
    let resolveFirst: (v: string) => void = () => {};
    const fetcher = vi.fn().mockImplementationOnce(
      () => new Promise<string>((resolve) => { resolveFirst = resolve; }),
    );
    const { result } = renderHook(() => useQuery(fetcher));
    const first = result.current.refetch();
    fetcher.mockResolvedValueOnce("second");
    await result.current.refetch();
    await waitFor(() => expect(result.current.data).toBe("second"));
    act(() => resolveFirst("stale"));
    await first;
    expect(result.current.data).toBe("second");
  });

  it("surfaces errors and recovers on refetch", async () => {
    const fetcher = vi.fn().mockRejectedValueOnce(new Error("boom")).mockResolvedValueOnce("ok");
    const { result } = renderHook(() => useQuery(fetcher));
    await waitFor(() => expect(result.current.error).toBe("boom"));
    await act(async () => { await result.current.refetch(); });
    expect(result.current.data).toBe("ok");
    expect(result.current.error).toBeUndefined();
  });
});
