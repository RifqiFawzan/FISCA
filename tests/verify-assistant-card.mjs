// Verifies the interactive financial_summary card: friendly reply with real
// numbers, trend array on the card, rendered trend bars, stacked category
// bar, and the real /reports CTA.
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

// API level
const res = await fetch(`${API}/assistant/messages`, {
  method: "POST",
  headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
  body: JSON.stringify({ text: "gimana analisis keuangan aku bulan ini?", source: "text" }),
});
const data = (await res.json()).data;
const apiCheck = {
  replyFriendly: /pemasukan.*Rp/i.test(data.reply),
  replySample: data.reply.slice(0, 160),
  cardType: data.card?.type,
  trendPoints: data.card?.trend?.length ?? 0,
  hasSummary: !!data.card?.summary,
  hasCategories: (data.card?.categories?.length ?? 0) > 0,
};
console.log("API:", JSON.stringify(apiCheck, null, 2));

// Frontend level (renders the same card in the assistant)
const browser = await chromium.launch({ channel: "chrome" });
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);
await page.goto(`${WEB}/assistant`, { waitUntil: "networkidle", timeout: 45000 });
// open the conversation we just created (sidebar first item)
await page.locator("aside ul button").first().click();
await page.waitForTimeout(1200);

const dom = await page.evaluate(() => {
  const cards = [...document.querySelectorAll("section")];
  const summaryCard = cards.find((s) => s.textContent.includes("Ringkasan"));
  if (!summaryCard) return { cardFound: false };
  const trend = summaryCard.querySelector('[role="img"][aria-label*="Grafik batang"]');
  const stackBar = [...summaryCard.querySelectorAll("div")].find(
    (d) => d.className.includes("flex h-2 w-full overflow-hidden"),
  );
  const cta = summaryCard.querySelector('a[href="/reports"]');
  const kpis = [...summaryCard.querySelectorAll("p.financial")].length;
  return {
    cardFound: true,
    trendBarGroups: trend ? trend.children.length : 0,
    trendLegend: !!summaryCard.textContent.includes("Tren masuk vs keluar"),
    stackSegments: stackBar ? stackBar.children.length : 0,
    ctaHref: cta ? cta.getAttribute("href") : null,
    ctaText: cta ? cta.textContent : null,
    kpiCount: kpis,
    monthLabels: trend ? [...trend.parentElement.querySelectorAll("div.mt-1\\.5 span")].map((s) => s.textContent) : [],
  };
});
console.log("DOM:", JSON.stringify(dom, null, 2));
await browser.close();

const pass =
  apiCheck.replyFriendly &&
  apiCheck.trendPoints > 0 &&
  apiCheck.hasSummary &&
  apiCheck.hasCategories &&
  dom.cardFound &&
  dom.trendBarGroups >= 1 &&
  dom.trendLegend &&
  dom.stackSegments >= 2 &&
  dom.ctaHref === "/reports" &&
  dom.kpiCount >= 3;
console.log(pass ? "ASSISTANT CARD UPGRADE PASS" : "ASSISTANT CARD UPGRADE FAIL");
process.exit(pass ? 0 : 1);
