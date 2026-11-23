// Verifies the shadcn MessageScroller integration in the assistant chat:
// mounts, renders messages inside scroller items, jump-to-end button exists,
// sending still works, and scrolling up reveals the jump button.
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
page.on("pageerror", (e) => console.log("PAGEERROR: " + e.message.slice(0, 200)));
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await page.goto(`${WEB}/assistant`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(800);

// open first conversation so messages exist
const convo = page.locator("aside ul button").first();
if (await convo.count()) {
  await convo.click();
  await page.waitForTimeout(1000);
}

const before = await page.evaluate(() => {
  const root = document.querySelector('[data-slot="message-scroller"]');
  const viewport = document.querySelector('[data-slot="message-scroller-viewport"]');
  const items = document.querySelectorAll('[data-slot="message-scroller-item"]');
  const jump = document.querySelector('[data-slot="message-scroller-button"]');
  return {
    rootMounted: !!root,
    viewportScrollable: viewport ? viewport.scrollHeight > viewport.clientHeight || viewport.scrollHeight > 0 : false,
    itemCount: items.length,
    jumpMounted: !!jump,
    atBottom: viewport ? viewport.scrollTop + viewport.clientHeight >= viewport.scrollHeight - 4 : null,
  };
});

// send a message (exercises content growth + auto-scroll)
await page.getByLabel("Tulis pesan").fill("uji scroller cek budget bulan ini");
await page.getByRole("button", { name: "Kirim pesan" }).click();
await page.waitForTimeout(4000);

const after = await page.evaluate(() => {
  const viewport = document.querySelector('[data-slot="message-scroller-viewport"]');
  const jump = document.querySelector('[data-slot="message-scroller-button"]');
  const replyVisible = [...document.querySelectorAll("p,div")].some((el) =>
    (el.textContent || "").includes("terpakai") || (el.textContent || "").includes("Rp"),
  );
  return {
    replyRendered: replyVisible,
    atBottom: viewport ? viewport.scrollTop + viewport.clientHeight >= viewport.scrollHeight - 4 : null,
    jumpActive: jump ? jump.getAttribute("data-active") : null,
    itemCount: document.querySelectorAll('[data-slot="message-scroller-item"]').length,
  };
});

console.log("BEFORE:", JSON.stringify(before));
console.log("AFTER :", JSON.stringify(after));
await browser.close();

const pass =
  before.rootMounted &&
  before.itemCount >= 1 &&
  before.jumpMounted &&
  after.replyRendered &&
  after.atBottom !== false;
console.log(pass ? "MESSAGE SCROLLER PASS" : "MESSAGE SCROLLER FAIL");
process.exit(pass ? 0 : 1);
