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

describe("custom fields", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("forwards custom_values on createIncident", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "inc-1", custom_values: { "def-1": "PO-42" } }), { status: 201 }),
    );
    await api.createIncident({
      site_id: "s1", asset_id: "a1", summary: "Pump", priority: "HIGH",
      custom_values: { po_number: "PO-42" }
    });
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const body = JSON.parse(String(init.body));
    expect(body.custom_values).toEqual({ po_number: "PO-42" });
  });

  it("serializes custom field filters on incident list", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ items: [], next_cursor: null, has_more: false, custom_fields: [] }), { status: 200 }),
    );
    await api.incidents({ custom_fields: { po_number: "PO-42" } });
    const [url] = fetchMock.mock.calls[0] as [string];
    expect(String(url)).toContain("custom_field.po_number=PO-42");
  });

  it("lists field definitions and creates one", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ items: [{ id: "def-1", key: "po_number" }] }), { status: 200 }),
    );
    const listed = await api.listFieldDefinitions("incident");
    expect(listed.items).toHaveLength(1);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "def-2", key: "zone" }), { status: 201 }),
    );
    const created = await api.createFieldDefinition({
      entity_type: "incident", key: "zone", label: "Zone", field_type: "SELECT",
      config: { options: [{ label: "A", value: "a" }] }, sort_order: 1
    });
    expect(created.key).toBe("zone");
    const [, init] = fetchMock.mock.calls[1] as [string, RequestInit];
    expect(String(init.method)).toBe("POST");
    expect(String(init.body)).toContain('"field_type":"SELECT"');
  });

  it("patches and retires a field definition", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ id: "def-1", label: "PO" }), { status: 200 }));
    await api.updateFieldDefinition("def-1", {
      label: "PO", field_type: "TEXT", config: {}, sort_order: 1, expected_version: 2
    });
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(String(init.method)).toBe("PATCH");
    expect(String(init.body)).toContain('"expected_version":2');
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ id: "def-1", status: "RETIRED" }), { status: 200 }));
    const retired = await api.retireFieldDefinition("def-1");
    expect(retired.status).toBe("RETIRED");
  });
});
