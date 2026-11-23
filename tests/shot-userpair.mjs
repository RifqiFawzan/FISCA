// Screenshots the first user/assistant pair (scrolls the scroller to top).
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
const page = await browser.newPage({ viewport: { width: 1600, height: 950 } });
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await page.goto(`${WEB}/assistant`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(900);
const convo = page.locator("aside ul button").first();
if (await convo.count()) await convo.click().catch(() => {});
await page.waitForSelector('[data-slot="bubble"]', { timeout: 15000 }).catch(() => {});
// Send a message: the anchor settles the person's turn near the top with the
// reply below, which is exactly the pair we want to photograph.
await page.getByLabel("Tulis pesan").fill("uji tampilan bubble asisten");
await page.getByRole("button", { name: "Kirim pesan" }).click();
await page.waitForTimeout(4000);
await page.screenshot({ path: "tests/.artifacts/assistant-ai-userpair.png" });
await browser.close();
console.log("done");
