import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { api, fetchAll } from "./api";

describe("api request hardening", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("turns a non-JSON error body into a readable message", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response("<html>gateway error</html>", { status: 502 }),
    );
    await expect(api.incidents()).rejects.toThrow(/502|response|html/i);
  });

  it("exposes the HTTP status on the error", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ detail: "nope" }), { status: 403 }),
    );
    try {
      await api.incidents();
      expect.unreachable();
    } catch (err) {
      expect((err as { status?: number }).status).toBe(403);
    }
  });

  it("wraps network failures as ApiError", async () => {
    vi.mocked(fetch).mockRejectedValueOnce(new TypeError("Failed to fetch"));
    await expect(api.incidents()).rejects.toMatchObject({ status: 0 });
  });
});

describe("list query building", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("serializes site, repeated states, cursor, and page size", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ items: [], next_cursor: null, has_more: false }), { status: 200 }),
    );
    await api.reports({ site_id: "s1", state: ["DRAFT", "SUBMITTED"], cursor: "abc", page_size: 50 });
    const [url] = fetchMock.mock.calls[0];
    expect(String(url)).toContain("/api/v1/reports?");
    expect(String(url)).toContain("site_id=s1");
    expect(String(url)).toContain("state=DRAFT");
    expect(String(url)).toContain("state=SUBMITTED");
    expect(String(url)).toContain("cursor=abc");
    expect(String(url)).toContain("page_size=50");
  });
});

describe("fetchAll", () => {
  it("walks next_cursor pages until exhausted", async () => {
    const first = { items: [{ id: "a" }], next_cursor: "c1", has_more: true };
    const second = { items: [{ id: "b" }, { id: "c" }], next_cursor: null, has_more: false };
    const fetchPage = vi.fn().mockResolvedValue(second);
    const all = await fetchAll(first, fetchPage);
    expect(all.map((item) => item.id)).toEqual(["a", "b", "c"]);
    expect(fetchPage).toHaveBeenCalledWith("c1");
  });

  it("stops when the first page has no cursor", async () => {
    const first = { items: [{ id: "a" }], next_cursor: null, has_more: false };
    const fetchPage = vi.fn();
    const all = await fetchAll(first, fetchPage);
    expect(all).toHaveLength(1);
    expect(fetchPage).not.toHaveBeenCalled();
  });
});
