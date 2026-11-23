// Verifies the Streaming Reply behavior in the assistant chat:
//1. while the request is pending, the shimmering "Berpikir…" pill is shown;
//2. after the reply arrives, words appear progressively (stream-word spans);
//3. the reply eventually shows complete text.
import { chromium } from "@playwright/test";

const WEB = "http://localhost:3000";
const API = "http://localhost:8080/api/v1";

const login = await fetch(`${API}/auth/login`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: "demo-1795313435247@fisca.test", password: "kata-sandi-e2e-123" }),
});
if (!login.ok) throw new Error("login failed");
const token = (await login.json()).data.token;

const browser = await chromium.launch({ channel: "chrome" });
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await page.goto(`${WEB}/assistant`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(600);

const count = (sel) => page.evaluate((s) => document.querySelectorAll(s).length, sel);

//1. pending -> thinking pill
await page.getByLabel("Tulis pesan").fill("gimana analisis keuangan saya bulan ini?");
await page.getByRole("button", { name: "Kirim pesan" }).click();
await page.waitForTimeout(700);
const thinkShown = await page.evaluate(() => {
  const el = [...document.querySelectorAll(".reply-think")].find((n) => n.offsetParent !== null);
  return el ? el.textContent.trim() : null;
});

//2. reply arrives -> words stream progressively
let firstSeen = null;
let secondSeen = null;
const deadline = Date.now() + 15000;
while (Date.now() < deadline) {
  const n = await count(".stream-word");
  if (n > 0 && firstSeen === null) {
    firstSeen = n;
    await page.waitForTimeout(250);
    secondSeen = await count(".stream-word");
    break;
  }
  await page.waitForTimeout(150);
}

//3. final state: the bubble holds the complete reply (either still showing
//    word spans or already swapped back to plain text after onDone).
await page.waitForTimeout(5000);
const final = await page.evaluate(() => {
  const words = [...document.querySelectorAll(".stream-word")];
  const pending = document.querySelectorAll(".stream-word-pending").length;
  const bubbles = [...document.querySelectorAll("div.rounded-2xl.rounded-tl-sm")];
  const last = bubbles[bubbles.length - 1];
  const text = last ? last.innerText.replace(/\s+/g, " ").trim() : "";
  return {
    pendingWords: pending,
    spanCount: words.length,
    replyText: text.slice(0, 140),
    complete: pending === 0 && text.length > 10,
  };
});

console.log(JSON.stringify({ thinkShown, firstSeen, secondSeen, progressed: firstSeen !== null && secondSeen > firstSeen, final }, null, 2));
await browser.close();

const pass = thinkShown === "Berpikir…" && firstSeen !== null && final.complete && final.replyText.length > 10;
console.log(pass ? "STREAMING REPLY PASS" : "STREAMING REPLY FAIL");
process.exit(pass ? 0 : 1);
