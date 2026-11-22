// Reproduces analytics.spec's setup: fresh user +3 seeded expenses, then
// open /reports at desktop width and report exactly what the page shows.
import { chromium } from "@playwright/test";

const WEB = "http://localhost:3001";
const API = "http://localhost:8080/api/v1";
const PASSWORD = "kata-sandi-e2e-123";

async function api(token, method, path, body, idem) {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
      ...(idem ? { "Idempotency-Key": idem } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  if (res.status >= 400) throw new Error(`${method} ${path} -> ${res.status} ${text.slice(0, 200)}`);
  return JSON.parse(text);
}

const stamp = Date.now();
const reg = await api(null, "POST", "/auth/register", { email: `repro-${stamp}@fisca.test`, password: PASSWORD });
const token = reg.data.token;
const accountId = (await api(token, "GET", "/accounts")).data[0].id;
for (const [i, [desc, amount]] of [["E2E kecil", 12000], ["E2E terbesar", 987654], ["E2E sedang", 54000]].entries()) {
  await api(token, "POST", "/transactions", { type: "expense", amount, description: desc, accountId, source: "manual", occurredAt: new Date().toISOString() }, `repro-${stamp}-${i}`);
}

const browser = await chromium.launch({ channel: "chrome" });
const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
const consoleErrors = [];
page.on("console", (m) => { if (m.type() === "error") consoleErrors.push(m.text().slice(0, 200)); });
page.on("pageerror", (e) => consoleErrors.push("PAGEERROR: " + e.message.slice(0, 300)));
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await page.goto(`${WEB}/reports`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(1500);

const state = await page.evaluate(() => ({
  tokenPresent: !!localStorage.getItem("fisca.token"),
  h1: document.querySelector("h1")?.textContent ?? "",
  loginPromptVisible: document.body.innerText.includes("untuk melanjutkan"),
  largestCard: [...document.querySelectorAll("h2")].some((h) => h.textContent.includes("Transaksi terbesar")),
  unusualCard: [...document.querySelectorAll("h2")].some((h) => h.textContent.includes("Pengeluaran tidak biasa")),
  url: location.href,
  bodySnippet: document.body.innerText.slice(0, 300),
}));
console.log(JSON.stringify({ state, consoleErrors: consoleErrors.slice(0, 6) }, null, 2));
await browser.close();
