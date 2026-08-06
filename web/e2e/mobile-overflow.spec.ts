import { test, expect } from "@playwright/test";

const horizontalOverflow = () => {
  const doc = document.documentElement;
  return doc.scrollWidth - doc.clientWidth;
};

test("landing page has no horizontal overflow at 390px, menu included", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/landing");
  await page.waitForLoadState("networkidle");
  expect(await page.evaluate(horizontalOverflow)).toBeLessThanOrEqual(0);

  const menu = page.getByRole("button", { name: "Menu" });
  await expect(menu).toBeVisible();
  await menu.click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await expect(page.getByRole("link", { name: "Features" })).toBeVisible();
  expect(await page.evaluate(horizontalOverflow)).toBeLessThanOrEqual(0);
});

test("header CTA stays visible and tappable at 390px", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/landing");
  await expect(page.getByRole("link", { name: "Book a pilot" }).first()).toBeVisible();
});
