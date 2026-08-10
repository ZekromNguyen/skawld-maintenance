import { test, expect, type Page } from "@playwright/test";

async function mockApi(page: Page, principal: Record<string, unknown>) {
  await page.route("**/api/v1/**", async (route) => {
    const path = new URL(route.request().url()).pathname.replace("/api/v1", "");
    if (path === "/me") {
      return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(principal) });
    }
    if (["/incidents", "/assets", "/executions", "/handovers"].includes(path)) {
      return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], next_cursor: null, has_more: false }) });
    }
    return route.fulfill({ status: 200, contentType: "application/json", body: "{}" });
  });
}

const READ_ONLY = {
  id: "p2",
  display_name: "Dev Manager",
  organization_id: "o1",
  site_ids: ["s1"],
  permissions: ["incident:read", "handover:accept"],
};

test("create actions are gated for principals without the permission", async ({ page }) => {
  await mockApi(page, READ_ONLY);
  await page.goto("/incidents");
  await expect(page.getByRole("button", { name: /create incident/i })).toHaveCount(0);
  await page.goto("/assets");
  await expect(page.getByRole("button", { name: /add asset/i })).toHaveCount(0);
});

test("sidebar hides Reports for a principal without report:write", async ({ page }) => {
  await mockApi(page, READ_ONLY);
  await page.goto("/");
  await expect(page.getByRole("link", { name: "Reports", exact: true })).toHaveCount(0);
});
