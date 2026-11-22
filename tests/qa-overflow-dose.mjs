// QA §5 + §14: mobile overflow check @360 and runtime antislop dose audit.
// Runs against the Docker stack (:3000, production build of the new UI).
import { chromium } from "@playwright/test";

const WEB = "http://localhost:3000";
const API = "http://localhost:8080/api/v1";
const PAGES = ["/home", "/history", "/reports", "/budget", "/savings", "/profile", "/assistant", "/login"];

const reg = await fetch(`${API}/auth/register`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: `qa-${Date.now()}@fisca.test`, password: "kata-sandi-e2e-123" }),
});
const token = (await reg.json()).data.token;

const browser = await chromium.launch({ channel: "chrome" });
const page = await browser.newPage({ viewport: { width: 360, height: 800 } });
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);

const overflow = [];
for (const path of PAGES) {
  await page.goto(`${WEB}${path}`, { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(400);
  const result = await page.evaluate(() => {
    const doc = document.documentElement;
    const offenders = [];
    for (const el of document.querySelectorAll("*")) {
      const r = el.getBoundingClientRect();
      if (r.right > window.innerWidth + 1 && r.width > 0) {
        const cs = getComputedStyle(el);
        // Decorative blur orbs live inside overflow-hidden cards; flag only
        // elements that can actually push the page width.
        if (cs.position !== "absolute" && cs.position !== "fixed") {
          offenders.push(el.tagName.toLowerCase() + "." + String(el.className).slice(0, 60));
        }
      }
    }
    return { scrollWidth: doc.scrollWidth, innerWidth: window.innerWidth, offenders: offenders.slice(0, 5) };
  });
  const ok = result.scrollWidth <= result.innerWidth + 1 && result.offenders.length === 0;
  overflow.push(`${ok ? "PASS" : "FAIL"} ${path} scrollWidth=${result.scrollWidth}/${result.innerWidth} offenders=${JSON.stringify(result.offenders)}`);
}
console.log("=== QA §5 mobile overflow @360 ===");
for (const line of overflow) console.log(line);

// QA §14 runtime dose audit (desktop viewport)
const page2 = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
await page2.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
const dose = {};
for (const path of ["/home", "/assistant", "/reports"]) {
  await page2.goto(`${WEB}${path}`, { waitUntil: "networkidle", timeout: 45000 });
  await page2.waitForTimeout(400);
  dose[path] = await page2.evaluate(() => {
    let blurred = 0;
    let glowing = 0;
    for (const el of document.querySelectorAll("*")) {
      const cs = getComputedStyle(el);
      if (cs.backdropFilter && cs.backdropFilter !== "none") blurred++;
      if (cs.boxShadow && cs.boxShadow.includes("78, 222, 163")) glowing++;
    }
    return { blurred, glowing };
  });
}
console.log("=== QA §14 runtime dose (blur backdropFilter count / glow shadow count) ===");
console.log(JSON.stringify(dose));

await browser.close();
