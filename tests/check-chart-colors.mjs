// Chart/color verification against an account WITH real transaction data
// (the demo account). Extracts computed colors from the charts on /reports
// and /home and compares them with the DESIGN.md §4.1 semantic tokens.
import { chromium } from "@playwright/test";

const WEB = "http://localhost:3001";
const API = "http://localhost:8080/api/v1";
const OUT = "C:/Users/zimig/AppData/Local/Temp/opencode/shots";

const login = await fetch(`${API}/auth/login`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email: "demo-1795313435247@fisca.test", password: "kata-sandi-e2e-123" }),
});
if (!login.ok) throw new Error("login failed: " + login.status);
const token = (await login.json()).data.token;

// Expected tokens (DESIGN.md §4.1, values adopted from Color_Design.md where noted)
const EXPECTED = {
  income: "rgb(52, 211, 153)",      // #34D399
  expense: "rgb(251, 113, 133)",    // #FB7185
  primary: "rgb(78, 222, 163)",     // #4EDEA3
  ai: "rgb(76, 215, 246)",          // #4CD7F6
};

const browser = await chromium.launch({ channel: "chrome" });
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
await page.addInitScript((t) => localStorage.setItem("fisca.token", t), token);

async function inspect(path, shotName) {
  await page.goto(`${WEB}${path}`, { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(800);
  const found = await page.evaluate(() => {
    const pick = (sel) =>
      [...document.querySelectorAll(sel)].map((el) => getComputedStyle(el).backgroundColor).filter((c) => c && c !== "rgba(0, 0, 0, 0)");
    return {
      incomeBars: pick('[class*="bg-income"]'),
      expenseBars: pick('[class*="bg-expense"]'),
      primaryBits: pick('[class*="bg-primary/"], [class*="bg-primary "]').slice(0, 6),
      inlineSegments: [...document.querySelectorAll("[style*='background-color']")]
        .map((el) => el.style.backgroundColor)
        .slice(0, 8),
      h1: document.querySelector("h1")?.textContent,
    };
  });
  await page.screenshot({ path: `${OUT}/${shotName}.png`, fullPage: true });
  return found;
}

const uniq = (arr) => [...new Set(arr)];
const reports = await inspect("/reports", "demo-reports");
const home = await inspect("/home", "demo-home");
await browser.close();

const norm = (c) => c.replace(/\s+/g, "");
// Tailwind v4 computes color-mix()/alpha values in oklab; convert back to sRGB
// before comparing (alpha is ignored: it does not shift hue).
const oklabToRgb = (str) => {
  const m = /oklab\(([\d.]+)\s+(-?[\d.]+)\s+(-?[\d.]+)/.exec(str);
  if (!m) return null;
  const L = parseFloat(m[1]), a = parseFloat(m[2]), b = parseFloat(m[3]);
  const l_ = L + 0.3963377774 * a + 0.2158037573 * b;
  const m_ = L - 0.1055613458 * a - 0.0638541728 * b;
  const s_ = L - 0.0894841775 * a - 1.291485548 * b;
  const l = l_ ** 3, mm = m_ ** 3, s = s_ ** 3;
  const lin = [
    +4.0767416621 * l - 3.3077115913 * mm + 0.2309699292 * s,
    -1.2684380046 * l + 2.6097574011 * mm - 0.3413193965 * s,
    -0.0041960863 * l - 0.7034186147 * mm + 1.707614701 * s,
  ];
  const gam = (c) => (c <= 0.0031308 ? 12.92 * c : 1.055 * Math.pow(c, 1 / 2.4) - 0.055);
  return "rgb(" + lin.map((c) => Math.round(Math.min(1, Math.max(0, gam(c))) * 255)).join(",") + ")";
};
const check = (values, expectedKey) => {
  const want = norm(EXPECTED[expectedKey]);
  const found = uniq(values.map((v) => oklabToRgb(v) ?? norm(v)));
  return { expected: EXPECTED[expectedKey], found, match: found.some((v) => v === want) };
};

const report = {
  "/reports": {
    incomeBars: check(reports.incomeBars, "income"),
    expenseBars: check(reports.expenseBars, "expense"),
    h1: reports.h1,
  },
  "/home": {
    income: check(home.incomeBars, "income"),
    expense: check(home.expenseBars, "expense"),
    primary: check(home.primaryBits, "primary"),
    categorySegmentsFromApi: home.inlineSegments,
    h1: home.h1,
  },
};
console.log(JSON.stringify(report, null, 2));

const pass =
  report["/reports"].incomeBars.match &&
  report["/reports"].expenseBars.match &&
  report["/home"].income.match &&
  report["/home"].expense.match;
console.log(pass ? "CHART COLORS MATCH DESIGN TOKENS" : "CHART COLOR MISMATCH");
console.log("screenshots: " + OUT + "/demo-reports.png, demo-home.png");
process.exit(pass ? 0 : 1);
