// Verifies the history fidelity deltas: desktop KPI strip + grouped
// selectable table + grouped mobile list, with exactly one variant mounted.
import { chromium } from "@playwright/test";

const WEB = "http://localhost:3001";
const API = "http://localhost:8080/api/v1";

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
  if (res.status >= 400) throw new Error(`${path} -> ${res.status} ${text.slice(0, 150)}`);
  return JSON.parse(text);
}

const reg = await api(null, "POST", "/auth/register", {
  email: `verify-hist-${Date.now()}@fisca.test`,
  password: "kata-sandi-e2e-123",
});
const token = reg.data.token;
const accountId = (await api(token, "GET", "/accounts")).data[0].id;
const day = (n) => {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d.toISOString();
};
await api(token, "POST", "/transactions", { type: "expense", amount: 42000, description: "Kopi hari ini", accountId, source: "voice", occurredAt: day(0) }, `h1-${Date.now()}`);
await api(token, "POST", "/transactions", { type: "expense", amount: 90000, description: "Bensin kemarin", accountId, source: "manual", occurredAt: day(1) }, `h2-${Date.now()}`);

const browser = await chromium.launch({ channel: "chrome" });
const results = [];

const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await page.goto(`${WEB}/history`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(600);
const desk = await page.evaluate(() => {
  const kpi = [...document.querySelectorAll("p")].some((p) => p.textContent === "Selisih arus kas");
  const table = [...document.querySelectorAll("table")].find((t) => t.textContent.includes("Transaksi"));
  const groupHeaders = table
    ? [...table.querySelectorAll("td[colspan]")].map((td) => td.textContent.trim())
    : [];
  const listMounted = [...document.querySelectorAll("ul")].some((u) => u.className.includes("divide-y"));
  return { kpiStrip: kpi, tableMounted: !!table, groupHeaders, listMounted, rows: table ? table.querySelectorAll("tbody tr").length : 0 };
});
results.push({ viewport: 1600, ...desk });
await page.close();

const m = await browser.newPage({ viewport: { width: 390, height: 844 } });
await m.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await m.goto(`${WEB}/history`, { waitUntil: "networkidle", timeout: 45000 });
await m.waitForTimeout(600);
const mob = await m.evaluate(() => {
  const kpi = [...document.querySelectorAll("p")].some((p) => p.textContent === "Selisih arus kas");
  const table = !!document.querySelector("table");
  const groupHeaders = [...document.querySelectorAll("ul li")]
    .filter((li) => li.className.includes("uppercase") && li.className.includes("tracking-wider"))
    .map((li) => li.textContent.trim());
  return { kpiStripMounted: kpi, tableMounted: table, groupHeaders, scrollWidth: document.documentElement.scrollWidth };
});
results.push({ viewport: 390, ...mob });
await browser.close();
console.log(JSON.stringify(results, null, 2));

const pass =
  desk.kpiStrip && desk.tableMounted && desk.groupHeaders.length >= 2 && desk.rows >= 2 && !desk.listMounted &&
  !mob.kpiStripMounted && !mob.tableMounted && mob.groupHeaders.length >= 2 && mob.scrollWidth <= 390;
console.log(pass ? "ALL HISTORY FIDELITY CHECKS PASS" : "HISTORY FIDELITY CHECK FAILED");
process.exit(pass ? 0 : 1);
