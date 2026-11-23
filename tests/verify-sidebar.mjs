// Sidebar verification with a real pointer (Playwright):
// rail mode (900px): tooltip opens on hover with the right label, keyboard
// focus opens it too, active marker present; labeled mode (1600px): group
// labels visible, no tooltip on hover. Screenshots saved to tests/.artifacts.
import { chromium } from "@playwright/test";
import { mkdirSync } from "node:fs";

const WEB = process.env.WEB_URL || "http://localhost:3001";
const API = "http://localhost:8080/api/v1";
mkdirSync("tests/.artifacts", { recursive: true });

const login = await fetch(`${API}/auth/login`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: "demo-1795313435247@fisca.test", password: "kata-sandi-e2e-123" }),
});
if (!login.ok) throw new Error("login failed");
const token = (await login.json()).data.token;

const browser = await chromium.launch({ channel: "chrome" });
const out = {};

// --- rail mode:900px (md..lg) ---
{
  const page = await browser.newPage({ viewport: { width: 900, height: 800 } });
  await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
  await page.goto(`${WEB}/home`, { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(600);

  const budget = page.locator('aside a[href="/budget"]');
  await budget.hover();
  await page.waitForTimeout(600); // delay 200 + margin
  out.railHoverTooltip = await page.evaluate(() => {
    const t = document.querySelector('[data-slot="tooltip-content"]');
    return t ? { text: t.textContent, visible: !!t.offsetParent } : null;
  });
  await page.screenshot({ path: "tests/.artifacts/sidebar-rail-tooltip.png" });

  // keyboard focus opens it as well
  await page.mouse.move(450, 400); // leave the link
  await page.waitForTimeout(300);
  await budget.focus();
  await page.waitForTimeout(600);
  out.railFocusTooltip = await page.evaluate(() => {
    const t = document.querySelector('[data-slot="tooltip-content"]');
    return t ? { text: t.textContent, visible: !!t.offsetParent } : null;
  });

  out.railActiveMarker = await page.evaluate(() => {
    const a = document.querySelector('a[aria-current="page"]');
    if (!a) return null;
    const marker = a.querySelector("span.bg-primary");
    return { label: a.getAttribute("aria-label"), hasMarker: !!marker };
  });
  await page.mouse.move(450, 400);
  await page.waitForTimeout(300);
  await page.screenshot({ path: "tests/.artifacts/sidebar-rail.png" });
  await page.close();
}

// --- labeled mode:1600px ---
{
  const page = await browser.newPage({ viewport: { width: 1600, height: 900 } });
  await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
  await page.goto(`${WEB}/home`, { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(600);

  out.labeled = await page.evaluate(() => {
    const labels = [...document.querySelectorAll('nav[aria-label="Navigasi samping"] p')]
      .map((p) => ({ text: p.textContent, visible: !!p.offsetParent }));
    const budget = document.querySelector('a[href="/budget"]');
    return {
      groupLabels: labels,
      budgetVisibleText: budget ? budget.textContent.trim() : null,
      sidebarWidth: document.querySelector("aside")?.getBoundingClientRect().width ?? null,
    };
  });

  // hover should NOT open a tooltip when labels are visible
  await page.locator('aside a[href="/budget"]').hover();
  await page.waitForTimeout(600);
  out.labeledHoverTooltip = await page.evaluate(() => {
    const t = document.querySelector('[data-slot="tooltip-content"]');
    return t ? { text: t.textContent } : null;
  });
  await page.screenshot({ path: "tests/.artifacts/sidebar-labeled.png" });
  await page.close();
}

await browser.close();
console.log(JSON.stringify(out, null, 2));
const pass =
  out.railHoverTooltip?.text === "Budget" &&
  out.railHoverTooltip?.visible === true &&
  out.railFocusTooltip?.text === "Budget" &&
  out.railActiveMarker?.label === "Beranda" &&
  out.railActiveMarker?.hasMarker === true &&
  out.labeled?.groupLabels?.length === 3 &&
  out.labeled?.groupLabels?.every((l) => l.visible) &&
  out.labeledHoverTooltip === null;
console.log(pass ? "SIDEBAR PASS" : "SIDEBAR FAIL");
process.exit(pass ? 0 : 1);
