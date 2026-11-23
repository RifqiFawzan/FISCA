// Diagnoses the "page grows downward" bug: measures document vs scroller heights.
import { chromium } from "@playwright/test";

const WEB = process.env.WEB_URL || "http://localhost:3000";
const API = "http://localhost:8080/api/v1";
const login = await fetch(`${API}/auth/login`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: "demo-1795313435247@fisca.test", password: "kata-sandi-e2e-123" }),
});
const token = (await login.json()).data.token;

const browser = await chromium.launch({ channel: "chrome" });
for (const width of [1600, 900]) {
  const page = await browser.newPage({ viewport: { width, height: 900 } });
  await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
  await page.goto(`${WEB}/assistant`, { waitUntil: "networkidle", timeout: 45000 });
  const convo = page.locator("aside ul button").first();
  if (await convo.count()) await convo.click();
  await page.waitForTimeout(800);
  const m = await page.evaluate(() => {
    const root = document.querySelector('[data-slot="message-scroller"]');
    const viewport = document.querySelector('[data-slot="message-scroller-viewport"]');
    const doc = document.scrollingElement;
    return {
      viewportH: window.innerHeight,
      docScrollH: doc ? doc.scrollHeight : 0,
      pageOverflows: doc ? doc.scrollHeight > window.innerHeight + 2 : null,
      rootHeight: root ? getComputedStyle(root).height : null,
      rootDisplay: root ? getComputedStyle(root).display : null,
      rootMinH: root ? getComputedStyle(root).minHeight : null,
      viewportOverflowY: viewport ? getComputedStyle(viewport).overflowY : null,
      viewportScrollable: viewport ? viewport.scrollHeight > viewport.clientHeight + 2 : false,
      mainHeight: document.querySelector("main") ? getComputedStyle(document.querySelector("main")).height : null,
    };
  });
  console.log(width + "px:", JSON.stringify(m, null, 1));
  await page.close();
}
await browser.close();
