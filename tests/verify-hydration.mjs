// Asserts there is no hydration mismatch (pageerror) on the main app pages.
// Prints PASS/FAIL per page; on failure prints the full error message.
import { chromium } from "@playwright/test";

const WEB = process.env.WEB_URL || "http://localhost:3001";
const API = "http://localhost:8080/api/v1";
const PAGES = ["/", "/assistant", "/reports", "/history", "/budget", "/profile", "/savings"];

const login = await fetch(`${API}/auth/login`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: "demo-1795313435247@fisca.test", password: "kata-sandi-e2e-123" }),
});
if (!login.ok) throw new Error("login failed");
const token = (await login.json()).data.token;

const browser = await chromium.launch({ channel: "chrome" });
let failures = 0;
for (const path of PAGES) {
  const page = await browser.newPage({ viewport: { width: 1600, height: 900 } });
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
  await page.goto(`${WEB}${path}`, { waitUntil: "networkidle", timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(1200);
  if (errors.length === 0) {
    console.log(`${path} -> OK`);
  } else {
    failures += 1;
    console.log(`${path} -> ${errors.length} pageerror(s):\n${errors[0]}`);
  }
  await page.close();
}
await browser.close();
console.log(failures === 0 ? "HYDRATION CLEAN" : `HYDRATION FAIL (${failures} page(s))`);
process.exit(failures === 0 ? 0 : 1);
