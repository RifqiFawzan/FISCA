// Sends a message, then measures the rendered AI Elements bubble styles
// (user = emerald right, assistant = surface-low left) and screenshots.
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
await page.getByLabel("Tulis pesan").fill("uji bubble pair");
await page.getByRole("button", { name: "Kirim pesan" }).click();
await page.waitForTimeout(8000);

const styles = await page.evaluate(() => {
  const pick = (el) => {
    if (!el) return null;
    const cs = getComputedStyle(el);
    return { bg: cs.backgroundColor, color: cs.color, radius: cs.borderRadius, border: cs.borderTopColor };
  };
  const userBubble = document.querySelector('[data-slot="bubble"][data-align="end"] [data-slot="bubble-content"]');
  const assistantBubble = document.querySelector('[data-slot="bubble"][data-align="start"] [data-slot="bubble-content"]');
  const avatars = [...document.querySelectorAll('[data-slot="message-avatar"]')].map(pick);
  return {
    userBubble: pick(userBubble),
    assistantBubble: pick(assistantBubble),
    userAvatar: avatars[0] ?? null,
    bubbleCount: document.querySelectorAll('[data-slot="bubble"]').length,
    messageCount: document.querySelectorAll('[data-slot="message"]').length,
  };
});
console.log(JSON.stringify(styles, null, 1));
await page.screenshot({ path: "tests/.artifacts/assistant-ai-userpair.png" });
await browser.close();
const pass =
  styles.bubbleCount >= 2 &&
  styles.userBubble?.bg === "rgb(78, 222, 163)" &&
  styles.assistantBubble?.bg === "rgb(24, 28, 36)";
console.log(pass ? "AI ELEMENTS BUBBLE PASS" : "AI ELEMENTS BUBBLE FAIL");
process.exit(pass ? 0 : 1);
