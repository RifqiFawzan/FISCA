// iPhone 11 check: WebKit (Safari engine), iPhone 11 device descriptor
// (414x896, touch, Safari UA). Asserts no horizontal overflow, no console
// errors, and every visible text input is >=16px (iOS auto-zooms below that).
import { webkit, devices } from "@playwright/test";
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

const browser = await webkit.launch();
const ctx = await browser.newContext({ ...devices["iPhone 11"] });
const failures = [];

async function check(path, { authed = true, name, waitFor } = {}) {
  const page = await ctx.newPage();
  const errors = [];
  page.on("console", (m) => { if (m.type() === "error") errors.push(m.text().slice(0, 160)); });
  page.on("pageerror", (e) => errors.push(e.message.slice(0, 160)));
  if (authed) await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
  await page.goto(`${WEB}${path}`, { waitUntil: "networkidle", timeout: 45000 }).catch(() => {});
  // Wait for the real (authed) UI, not the pre-hydration no-token branch.
  if (waitFor) await page.waitForSelector(waitFor, { timeout: 12000 }).catch(() => {});
  await page.waitForTimeout(600);

  const info = await page.evaluate(() => {
    const overflow = Math.max(0, document.documentElement.scrollWidth - window.innerWidth);
    const fonts = [...document.querySelectorAll("input, textarea")]
      .filter((el) => el.type !== "hidden" && el.type !== "color" && el.type !== "checkbox")
      .map((el) => ({
        label: el.getAttribute("aria-label") || el.id || el.placeholder || "?",
        size: parseFloat(getComputedStyle(el).fontSize),
      }));
    return { overflow, fonts, ua: navigator.userAgent.includes("iPhone") };
  });

  if (info.overflow > 1) failures.push(`${name} overflow ${info.overflow}px`);
  for (const f of info.fonts) if (f.size < 16) failures.push(`${name} input "${f.label}" font-size ${f.size} (<16 -> iOS zoom)`);
  for (const e of [...new Set(errors)]) failures.push(`${name} console: ${e}`);
  console.log(`${name}: overflow=${info.overflow} inputs=${JSON.stringify(info.fonts)} iphoneUA=${info.ua}`);
  await page.screenshot({ path: `tests/.artifacts/iphone-${name}.png`, fullPage: true });
  await page.close();
}

await check("/login", { authed: false, name: "login", waitFor: "#login-email" });
await check("/home", { name: "home", waitFor: "text=Ringkasan arus kas" });
await check("/assistant", { name: "assistant", waitFor: 'input[aria-label="Tulis pesan"]' });
await check("/history", { name: "history", waitFor: 'input[placeholder*="Cari"]' });

await browser.close();
console.log("FAILURES:", failures.length);
for (const f of failures) console.log("  X " + f);
const pass = failures.length === 0;
console.log(pass ? "IPHONE11 PASS" : "IPHONE11 FAIL");
process.exit(pass ? 0 : 1);
