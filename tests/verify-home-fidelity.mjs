// Verifies the home visual-fidelity deltas programmatically (immune to the
// flaky image-delivery path): header panel, tinted KPI tiles, tinted account
// tile, desktop table vs mobile list.
import { chromium } from "@playwright/test";

const WEB = "http://localhost:3001";
const API = "http://localhost:8080/api/v1";
const reg = await fetch(`${API}/auth/register`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: `verify-${Date.now()}@fisca.test`, password: "kata-sandi-e2e-123" }),
});
const token = (await reg.json()).data.token;
// one transaction so the table has a row
const accounts = await (await fetch(`${API}/accounts`, { headers: { Authorization: `Bearer ${token}` } })).json();
await fetch(`${API}/transactions`, {
  method: "POST",
  headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json", "Idempotency-Key": `v-${Date.now()}` },
  body: JSON.stringify({ type: "expense", amount: 55000, description: "Kopi verifikasi", accountId: accounts.data[0].id, source: "voice", occurredAt: new Date().toISOString() }),
});

const browser = await chromium.launch({ channel: "chrome" });
const results = [];

// Desktop
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await page.goto(`${WEB}/home`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(600);
const d = await page.evaluate(() => {
  const out = {};
  const header = document.querySelector("section");
  out.headerPanelBlur = header ? getComputedStyle(header).backdropFilter : "none";
  const tiles = [...document.querySelectorAll("span")].filter((s) => s.className.includes("size-10") && s.className.includes("place-items-center"));
  out.kpiTileBg = tiles.slice(0, 4).map((t) => getComputedStyle(t).backgroundColor);
  const acctTile = [...document.querySelectorAll("li span")].find((s) => String(s.className).includes("size-9"));
  out.acctTileColor = acctTile ? getComputedStyle(acctTile).color : null;
  const table = [...document.querySelectorAll("table")].find((t) => t.textContent.includes("Transaksi"));
  out.tableVisible = table ? table.offsetParent !== null : false;
  out.tableHeaders = table ? [...table.querySelectorAll("th")].map((th) => th.textContent.trim()) : [];
  out.tableRows = table ? table.querySelectorAll("tbody tr").length : 0;
  const lists = [...document.querySelectorAll("ul")].filter((u) => u.className.includes("divide-y"));
  out.listHiddenAtXl = lists.length > 0 ? lists.every((u) => getComputedStyle(u).display === "none" || u.closest("[class*='xl:hidden']")) : null;
  return out;
});
results.push({ viewport: 1600, ...d });
await page.close();

// Mobile
const m = await browser.newPage({ viewport: { width: 390, height: 844 } });
await m.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await m.goto(`${WEB}/home`, { waitUntil: "networkidle", timeout: 45000 });
await m.waitForTimeout(600);
const mob = await m.evaluate(() => {
  const out = {};
  const header = document.querySelector("section");
  out.headerPanelBlur = header ? getComputedStyle(header).backdropFilter : "none";
  const table = [...document.querySelectorAll("table")].find((t) => t.textContent.includes("Transaksi"));
  out.tableVisible = table ? table.offsetParent !== null : false;
  const list = [...document.querySelectorAll("ul")].find((u) => u.className.includes("divide-y"));
  out.listVisible = list ? getComputedStyle(list).display !== "none" : null;
  out.scrollWidth = document.documentElement.scrollWidth;
  return out;
});
results.push({ viewport: 390, ...mob });
await browser.close();
console.log(JSON.stringify(results, null, 2));

// assertions
const [desk, phone] = results;
const pass =
  desk.headerPanelBlur.includes("blur(16px)") &&
  desk.kpiTileBg.length === 4 &&
  desk.kpiTileBg.every((c) => c.includes("/ 0.1)")) && // tinted tiles (Tailwind v4 computes in oklab)
  desk.acctTileColor === "rgb(52, 211, 153)" && // cash account tile = income tint
  desk.tableVisible && desk.tableHeaders.length === 6 && desk.tableRows >= 1 &&
  phone.headerPanelBlur === "none" && phone.tableVisible === false && phone.listVisible === true &&
  phone.scrollWidth <= 390;
console.log(pass ? "ALL HOME FIDELITY CHECKS PASS" : "HOME FIDELITY CHECK FAILED");
process.exit(pass ? 0 : 1);
