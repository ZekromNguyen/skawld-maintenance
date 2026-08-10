import { test, expect, type Page } from "@playwright/test";

async function mockApi(page: Page, fixtures: Record<string, unknown>) {
  await page.route("**/api/v1/**", async (route) => {
    const path = new URL(route.request().url()).pathname.replace("/api/v1", "");
    if (path === "/me") {
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(fixtures.principal ?? { id: "p1", display_name: "Dev Supervisor", organization_id: "o1", site_ids: ["s1"], permissions: ["incident:read"] }),
      });
    }
    if (["/incidents", "/assets", "/executions", "/handovers"].includes(path)) {
      const key = path.slice(1) as "incidents" | "assets" | "executions" | "handovers";
      return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(fixtures[key] ?? { items: [], next_cursor: null, has_more: false }) });
    }
    return route.fulfill({ status: 200, contentType: "application/json", body: "{}" });
  });
}

const EMPTY = {
  incidents: { items: [], next_cursor: null, has_more: false },
  assets: { items: [], next_cursor: null, has_more: false },
  executions: { items: [], next_cursor: null, has_more: false },
  handovers: { items: [], next_cursor: null, has_more: false },
};

test("dashboard shows truthful empty queues with a way forward", async ({ page }) => {
  await mockApi(page, EMPTY);
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Operations overview/);
  // Annunciator cells read a truthful zero.
  const incidentsLink = page.getByRole("link", { name: /open incidents/i });
  await expect(incidentsLink).toContainText("0");
  // Recommended tab empty state with CTA.
  await expect(page.getByText("No open incidents right now.")).toBeVisible();
  await page.getByRole("tab", { name: /assigned to me/i }).click();
  await expect(page.getByText("No executions assigned to you right now.")).toBeVisible();
});

test("incident queue renders an empty state instead of a bare table", async ({ page }) => {
  await mockApi(page, EMPTY);
  await page.goto("/incidents");
  await expect(page.getByText("No incidents match your filters.")).toBeVisible();
});
