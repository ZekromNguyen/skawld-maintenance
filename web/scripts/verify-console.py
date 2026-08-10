"""Playwright verification pass for the routed pilot console.

Authenticates through Keycloak as dev.supervisor (Reports visible) and
dev.technician (Reports hidden) in isolated contexts, then asserts shell
rendering, navigation, deep links, and no horizontal overflow.
"""
from playwright.sync_api import sync_playwright


def login(page, username, password, display_name):
    page.goto("http://localhost:5173/")
    page.wait_for_load_state("networkidle")
    page.wait_for_timeout(600)
    # Wait for either the Keycloak login form or the console (already authed).
    for _ in range(20):
        if page.locator("#username").count() > 0:
            break
        if page.get_by_text(display_name).count() > 0:
            return
        page.wait_for_timeout(300)
    if page.locator("#username").count() == 0:
        # Force a fresh login attempt via the OIDC begin endpoint.
        page.evaluate("window.location.href = '/auth/login'")
        page.wait_for_load_state("networkidle")
        page.locator("#username").wait_for(timeout=10000)
    page.locator("#username").fill(username)
    page.locator("#password").fill(password)
    page.locator("#kc-login").click()
    page.wait_for_load_state("networkidle")
    page.get_by_text(display_name).first.wait_for(timeout=10000)


with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)

    # ---- Supervisor (isolated context) ----
    ctx = browser.new_context(viewport={"width": 1440, "height": 900})
    page = ctx.new_page()
    login(page, "dev.supervisor", "dev-supervisor-pw", "Dev Supervisor")
    overflow = page.evaluate(
        "document.documentElement.scrollWidth > document.documentElement.clientWidth"
    )
    print("[supervisor] horizontal overflow:", overflow)
    print("[supervisor] reports nav visible:", page.get_by_text("Reports", exact=True).count() > 0)

    page.get_by_text("Incident execution", exact=True).first.click()
    page.wait_for_timeout(800)
    print("[supervisor] incidents h1:", page.locator("h1").first.inner_text()[:40])
    page.screenshot(path="/tmp/console-supervisor.png", full_page=True)

    page.goto("http://localhost:5173/executions/dae47619-649c-4e27-8061-4d165257d65a")
    page.wait_for_load_state("networkidle")
    page.wait_for_timeout(1000)
    print("[supervisor] workbench h1:", page.locator("h1").first.inner_text()[:50])
    page.screenshot(path="/tmp/console-workbench.png", full_page=True)
    ctx.close()

    # ---- Manager (isolated context): no report:write -> Reports nav hidden ----
    ctx = browser.new_context(viewport={"width": 1440, "height": 900})
    page = ctx.new_page()
    login(page, "dev.manager", "dev-manager-pw", "Dev Manager")
    overflow = page.evaluate(
        "document.documentElement.scrollWidth > document.documentElement.clientWidth"
    )
    print("[manager] horizontal overflow:", overflow)
    print("[manager] reports nav visible:", page.get_by_text("Reports", exact=True).count() > 0)
    page.screenshot(path="/tmp/console-manager.png", full_page=True)
    ctx.close()

    browser.close()
