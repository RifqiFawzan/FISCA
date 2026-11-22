// Verifies the new area/line cash-flow chart on /reports with the demo
// account (real multi-day data): SVG present, income/expense stroke colors
// match design tokens, area fills and hover tooltips exist.
import { chromium } from "@playwright/test";

const WEB = "http://localhost:3001";
const API = "http://localhost:8080/api/v1";
const OUT = "C:/Users/zimig/AppData/Local/Temp/opencode/shots";

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
await page.goto(`${WEB}/reports`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(800);

const chart = await page.evaluate(() => {
  const svg = [...document.querySelectorAll("svg")].find((s) => s.querySelector("title")?.textContent?.includes("Arus kas"));
  if (!svg) return { present: false };
  const paths = [...svg.querySelectorAll("path")];
  const strokes = paths.map((p) => getComputedStyle(p).stroke).filter((s) => s && s !== "none");
  const fills = paths.filter((p) => (p.getAttribute("fill") || "").startsWith("url(")).length;
  const tooltips = [...svg.querySelectorAll("rect title")].map((t) => t.textContent);
  const gridTexts = [...svg.querySelectorAll("text")].map((t) => t.textContent);
  return {
    present: true,
    lineCount: strokes.length,
    strokes,
    areaFills: fills,
    tooltipCount: tooltips.length,
    sampleTooltip: tooltips[0] || null,
    yLabels: gridTexts,
    w: svg.clientWidth,
    h: svg.clientHeight,
  };
});

await page.screenshot({ path: `${OUT}/demo-reports-curve.png`, fullPage: true });
await browser.close();
console.log(JSON.stringify(chart, null, 2));

const pass =
  chart.present &&
  chart.lineCount >= 2 &&
  chart.strokes.some((s) => s.replace(/\s/g, "") === "rgb(52,211,153)") &&
  chart.strokes.some((s) => s.replace(/\s/g, "") === "rgb(251,113,133)") &&
  chart.areaFills >= 2 &&
  chart.tooltipCount >= 2;
console.log(pass ? "CURVE CHART PASS" : "CURVE CHART FAIL");
console.log("screenshot: " + OUT + "/demo-reports-curve.png");
process.exit(pass ? 0 : 1);
