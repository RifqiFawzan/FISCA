// Verifies structured advice output: the reply renders as a real list with
// bold (no raw markdown asterisks) and the summary metrics card with its
// chart appears below the answer.
import { chromium } from "@playwright/test";
import { mkdirSync } from "node:fs";

const WEB = process.env.WEB_URL || "http://localhost:3000";
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
const page = await browser.newPage({ viewport: { width: 1600, height: 950 } });
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await page.goto(`${WEB}/assistant`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(900);

await page.getByLabel("Tulis pesan").fill("Berikan saran untuk keuangan saya bulan ini");
await page.getByRole("button", { name: "Kirim pesan" }).click();

// Wait until the reply is a finished static render: a real list exists and
// no streaming word spans remain (bold appears only in the static mode).
let found = false;
const deadline = Date.now() + 25000;
while (Date.now() < deadline && !found) {
  found = await page.evaluate(() =>
    document.querySelectorAll('[data-slot="bubble-content"] li').length > 0 &&
    document.querySelectorAll(".stream-word, .stream-word-pending").length === 0,
  );
  if (!found) await page.waitForTimeout(300);
}

const out = await page.evaluate(() => {
  const bubbles = [...document.querySelectorAll('[data-slot="bubble-content"]')];
  const withList = bubbles.filter((b) => b.querySelector("li"));
  const reply = withList.length
    ? withList[withList.length - 1]
    : [...bubbles].reverse().find((b) => b.textContent.includes("saran") && !b.closest('[data-align="end"]')) || null;
  const card = [...document.querySelectorAll("section")].find((s) =>
    (s.textContent || "").includes("Ringkasan") &&
    (s.textContent || "").includes("Tren masuk vs keluar"),
  );
  const text = reply ? reply.innerText : "";
  return {
    bubblesWithList: withList.length,
    listItems: reply ? reply.querySelectorAll("li").length : 0,
    ordered: reply ? reply.querySelectorAll("ol li").length : 0,
    bold: reply ? reply.querySelectorAll("strong").length : 0,
    rawMarkdown: text.includes("**"),
    cardFound: !!card,
    cardChart: !!card && card.querySelectorAll('[role="img"]').length > 0,
    replyHead: text.slice(0, 90).replace(/\n/g, " | "),
  };
});
console.log(JSON.stringify(out, null, 1));
await page
  .screenshot({ path: "tests/.artifacts/advice-structured.png", fullPage: false, timeout: 15000 })
  .catch((e) => console.log("screenshot skipped: " + String(e.message).slice(0, 80)));
await browser.close();

const pass =
  out.bubblesWithList >= 1 &&
  out.listItems >= 3 &&
  out.bold >= 2 &&
  out.rawMarkdown === false &&
  out.cardFound === true &&
  out.cardChart === true;
console.log(pass ? "ADVICE STRUCTURE PASS" : "ADVICE STRUCTURE FAIL");
process.exit(pass ? 0 : 1);
