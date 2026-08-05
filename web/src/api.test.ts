import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { api } from "./api";

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
