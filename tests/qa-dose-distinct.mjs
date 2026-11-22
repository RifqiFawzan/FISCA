// QA §14 refinement: count DISTINCT backdrop-filter values and glow shadow
// values (surface types, per DESIGN.md §4.5/§4.6), not instances.
import { chromium } from "@playwright/test";

const WEB = "http://localhost:3000";
const API = "http://localhost:8080/api/v1";
const reg = await fetch(`${API}/auth/register`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: `qa2-${Date.now()}@fisca.test`, password: "kata-sandi-e2e-123" }),
});
const token = (await reg.json()).data.token;

const browser = await chromium.launch({ channel: "chrome" });
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);

const allBlur = new Set();
const allGlow = new Set();
for (const path of ["/home", "/history", "/reports", "/budget", "/savings", "/profile", "/assistant"]) {
  await page.goto(`${WEB}${path}`, { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(300);
  const found = await page.evaluate(() => {
    const blur = new Set();
    const glow = new Set();
    for (const el of document.querySelectorAll("*")) {
      const cs = getComputedStyle(el);
      if (cs.backdropFilter && cs.backdropFilter !== "none") blur.add(cs.backdropFilter);
      if (cs.boxShadow && cs.boxShadow.includes("78, 222, 163")) glow.add(cs.boxShadow);
    }
    return { blur: [...blur], glow: [...glow] };
  });
  found.blur.forEach((v) => allBlur.add(v));
  found.glow.forEach((v) => allGlow.add(v));
}
console.log("distinct backdrop-filters: " + allBlur.size);
for (const v of allBlur) console.log("  " + v);
console.log("distinct emerald glow shadows: " + allGlow.size);
for (const v of allGlow) console.log("  " + v.slice(0, 100));
await browser.close();
