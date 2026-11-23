// Captures "before" screenshots of the assistant screen for visual audit.
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

async function shot(width, height, name, { openFirst = true } = {}) {
  const page = await browser.newPage({ viewport: { width, height } });
  await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
  await page.goto(`${WEB}/assistant`, { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(900);
  if (openFirst) {
    const diag = await page.evaluate(() => {
      const aside = document.querySelector("aside");
      const btn = document.querySelector("aside ul button");
      const r = btn?.getBoundingClientRect();
      return {
        asideDisplay: aside ? getComputedStyle(aside).display : null,
        asideCls: aside ? aside.className : null,
        btnRect: r ? [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)] : null,
        btnVisible: btn ? !!(btn.offsetParent) : null,
        ulCount: document.querySelectorAll("aside ul").length,
      };
    });
    console.log(name, "diag:", JSON.stringify(diag));
    const convo = page.locator("aside ul button").first();
    if (await convo.count()) {
      await convo.click({ timeout: 5000 }).catch((e) => console.log(name, "click failed:", e.message.slice(0, 80)));
      await page.waitForTimeout(900);
    }
  }
  await page.screenshot({ path: `tests/.artifacts/${name}.png` });
  await page.close();
}

await shot(1600, 950, "assistant-ai-desktop");
await shot(1600, 950, "assistant-ai-welcome", { openFirst: false });
await shot(390, 844, "assistant-ai-mobile");

// focus state + tap-target measurement (DESIGN.md Â§11: important targets 44px)
{
  const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
  await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
  await page.goto(`${WEB}/assistant`, { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(900);
  const input = page.getByLabel("Tulis pesan");
  await input.click();
  await page.waitForTimeout(300);
  await page.screenshot({ path: "tests/.artifacts/assistant-ai-focus.png" });
  const measures = await page.evaluate(() => {
    const mic = document.querySelector('button[aria-label="Rekam suara"], button[aria-label="Hentikan rekaman"]');
    const send = document.querySelector('button[aria-label="Kirim pesan"]');
    const chips = [...document.querySelectorAll("button")].filter((b) => b.className.includes("rounded-full") && b.textContent.length > 8);
    const rect = (el) => (el ? [Math.round(el.getBoundingClientRect().width), Math.round(el.getBoundingClientRect().height)] : null);
    const form = document.querySelector("form");
    return {
      mic: rect(mic),
      send: rect(send),
      chip0: rect(chips[0]),
      inputFocusBorder: form ? getComputedStyle(form).borderTopColor : null,
    };
  });
  console.log("measures:", JSON.stringify(measures));
  await page.close();
}
await browser.close();
console.log("done");
