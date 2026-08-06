import { test, expect } from "@playwright/test";

test("favicon and OpenAPI spec resolve without 404", async ({ request }) => {
  const favicon = await request.get("/favicon.svg");
  expect(favicon.status()).toBe(200);
  const openapi = await request.get("/openapi.yaml");
  expect(openapi.status()).toBe(200);
});

test("landing page loads without console errors at 390px", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(msg.text());
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/landing");
  await page.waitForLoadState("networkidle");
  expect(errors).toEqual([]);
});
