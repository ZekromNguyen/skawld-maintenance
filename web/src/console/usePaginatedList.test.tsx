import { describe, it, expect } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { usePaginatedList } from "./usePaginatedList";

describe("usePaginatedList", () => {
  it("walks pages via next_cursor until has_more is false", async () => {
    const pages = [
      { items: [{ id: "1" }], next_cursor: "c1", has_more: true },
      { items: [{ id: "2" }], next_cursor: null, has_more: false },
    ];
    let calls = 0;
    const fetcher = async ({ cursor }: { page_size: number; cursor?: string }) => {
      const page = pages[calls++];
      if (cursor !== (calls === 2 ? "c1" : undefined)) throw new Error("bad cursor");
      return page;
    };
    const { result } = renderHook(() => usePaginatedList(fetcher, []));
    await waitFor(() => expect(result.current.items).toEqual([{ id: "1" }]));
    expect(result.current.hasMore).toBe(true);
    await act(async () => {
      await result.current.loadMore();
    });
    expect(result.current.items).toEqual([{ id: "1" }, { id: "2" }]);
    expect(result.current.hasMore).toBe(false);
  });
});
