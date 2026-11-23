// Verifies the profile account rows reflow: single info block, balance on
// line one, meta and actions aligned on line two (no orphaned wrap).
import { chromium } from "@playwright/test";

const WEB = process.env.WEB_URL || "http://localhost:3001";
const API = "http://localhost:8080/api/v1";
const login = await fetch(`${API}/auth/login`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: "demo-1795313435247@fisca.test", password: "kata-sandi-e2e-123" }),
});
const token = (await login.json()).data.token;

const browser = await chromium.launch({ channel: "chrome" });
const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await page.goto(`${WEB}/profile`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(900);

const rows = await page.evaluate(() => {
  const out = [];
  for (const li of document.querySelectorAll("li")) {
    if (!/Dibuat/i.test(li.textContent || "")) continue;
    const balance = li.querySelector(".financial");
    const meta = [...li.querySelectorAll("p")].find((p) => /Dibuat/i.test(p.textContent));
    const actions = [...li.querySelectorAll("button")];
    const r = (el) => (el ? el.getBoundingClientRect() : null);
    // Structure proof: every action lives inside the same line-two container
    // as the meta text (no orphaned wrap), and the block count is stable.
    const lineTwo = meta ? meta.parentElement : null;
    out.push({
      childBlocks: li.children.length,
      height: Math.round(li.getBoundingClientRect().height),
      sameLine: !!(lineTwo && actions.length && actions.every((b) => lineTwo.contains(b))),
      balanceOnLine1: balance && lineTwo ? r(balance).top < r(lineTwo).top : null,
      actionSizes: actions.map((b) => Math.round(r(b).width) + "x" + Math.round(r(b).height)),
    });
  }
  return out;
});
console.log(JSON.stringify(rows, null, 1));
await browser.close();
const pass =
  rows.length >= 1 &&
  rows.every(
    (r) =>
      r.childBlocks === 2 &&
      r.sameLine === true &&
      r.balanceOnLine1 === true &&
      r.height < 120 &&
      r.actionSizes.every((s) => {
        const [w, h] = s.split("x").map(Number);
        return w >= 44 && h >= 44;
      }),
  );
console.log(pass ? "PROFILE ROWS PASS" : "PROFILE ROWS FAIL");
process.exit(pass ? 0 : 1);
