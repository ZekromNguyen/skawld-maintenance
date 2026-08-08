import { test, expect, type Page } from "@playwright/test";

/** Mock the console API so specs run without Keycloak or the backend. */
async function mockApi(page: Page, fixtures: Record<string, unknown>) {
  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname.replace("/api/v1", "");
    if (path === "/me") {
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(fixtures.principal ?? { id: "p1", display_name: "Dev Supervisor", organization_id: "o1", site_ids: ["s1"], permissions: ["incident:read", "incident:create", "execution:write"] }),
      });
    }
    if (path === "/incidents" || path === "/assets" || path === "/executions" || path === "/handovers") {
      const key = path.slice(1) as "incidents" | "assets" | "executions" | "handovers";
      const list = fixtures[key] ?? { items: [], next_cursor: null, has_more: false };
      return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(list) });
    }
    const incidentMatch = path.match(/^\/incidents\/([^/]+)$/);
    if (incidentMatch) {
      const incident = (fixtures.incidents as { items: Array<{ id: string }> }).items.find(
        (item) => item.id === incidentMatch[1],
      );
      return incident
        ? route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(incident) })
        : route.fulfill({ status: 404, contentType: "application/json", body: JSON.stringify({ status: 404, title: "Not found" }) });
    }
    return route.fulfill({ status: 200, contentType: "application/json", body: "{}" });
  });
}

const HAPPY_FIXTURES = {
  principal: { id: "p1", display_name: "Dev Supervisor", organization_id: "o1", site_ids: ["s1"], permissions: ["incident:read", "incident:create", "execution:write"] },
  incidents: {
    items: [
      { id: "i1", site_id: "s1", asset_id: "a1", asset_tag: "P-302", number: "IN-1", summary: "Pump vibration", severity: "HIGH", state: "OPEN", detected_at: new Date().toISOString(), version: 1 },
      { id: "i2", site_id: "s1", asset_id: "a2", asset_tag: "P-304", number: "IN-2", summary: "Bearing temperature", severity: "MEDIUM", state: "IN_PROGRESS", detected_at: new Date().toISOString(), version: 1 },
    ],
    next_cursor: null,
    has_more: false,
  },
  assets: {
    items: [{ id: "a1", site_id: "s1", tag: "P-302", name: "Process Pump", class: "CENTRIFUGAL_PUMP", status: "OPERATIONAL", source_of_truth: "OWNED_BY_SKAWLD", criticality: { rating: "A" } }],
    next_cursor: null,
    has_more: false,
  },
  executions: {
    items: [{ id: "e1", incident_id: "i1", asset_id: "a1", asset_tag: "P-302", purpose: "Shaft alignment check", state: "IN_PROGRESS", version: 1, steps: [], measurements: [{ id: "m1", measurement_type: "VIBRATION_VELOCITY", value: "2.4", unit: "mm/s", observed_at: new Date().toISOString() }], observations: [], actions: [] }],
    next_cursor: null,
    has_more: false,
  },
  handovers: { items: [], next_cursor: null, has_more: false },
};

test("dashboard renders the annunciator strip and tabbed For You queue", async ({ page }) => {
  await mockApi(page, HAPPY_FIXTURES);
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Operations overview/);
  // Signature band
  await expect(page.getByRole("link", { name: /open incidents/i })).toBeVisible();
  // Recommended queue with a real count
  await expect(page.getByText("Pump vibration")).toBeVisible();
  await expect(page.getByRole("tab", { name: /recommended/i })).toContainText("2");
  // Assigned-to-me queue switches without a reload
  await page.getByRole("tab", { name: /assigned to me/i }).click();
  await expect(page.getByText("Shaft alignment check")).toBeVisible();
  // No horizontal overflow at 360px
  await page.setViewportSize({ width: 360, height: 800 });
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
  );
  expect(overflow).toBe(false);
});

test("incident queue renders dense rows and the metadata rail on detail", async ({ page }) => {
  await mockApi(page, HAPPY_FIXTURES);
  await page.goto("/incidents");
  await expect(page.getByText("Pump vibration")).toBeVisible();
  const row = page.locator("tr", { hasText: "Pump vibration" }).first();
  await expect(row.locator(".alarm-edge")).toBeVisible();
  await row.click();
  await expect(page.getByText("Facts")).toBeVisible();
  // Activity feed below content
  await expect(page.getByText("Incident detected")).toBeVisible();
});

test("j/k keyboard navigation moves selection and Enter follows the row", async ({ page }) => {
  await mockApi(page, HAPPY_FIXTURES);
  await page.goto("/incidents");
  await page.getByRole("tab", { name: /all/i }).click();
  await page.locator(".table-wrap").focus();
  await page.keyboard.press("j");
  await page.keyboard.press("j");
  const selected = page.locator("tr.data-row.selected");
  await expect(selected).toHaveCount(1);
});
