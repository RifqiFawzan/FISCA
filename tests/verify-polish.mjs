// Verifies the final polish pass: savings goal (icon tile + percent),
// profile (section icon tiles, initials tiles,2-col category grid),
// assistant (AI message bubble).
import { chromium } from "@playwright/test";

const WEB = "http://localhost:3001";
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
const out = {};

// Savings
await page.goto(`${WEB}/savings`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(600);
out.savings = await page.evaluate(() => {
  const headers = [...document.querySelectorAll("h2")];
  const goal = headers.find((h) => h.textContent.includes("Dana Darurat"));
  const header = goal?.closest("div");
  const iconTile = header?.parentElement?.querySelector("span.size-10");
  const percent = [...document.querySelectorAll("span.financial")].find((s) => /\d+\.\d%$/.test(s.textContent.trim()));
  return {
    goalHeaderIconTile: !!iconTile,
    percentText: percent ? percent.textContent : null,
    percentColor: percent ? getComputedStyle(percent).color : null,
  };
});

// Profile
await page.goto(`${WEB}/profile`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(600);
out.profile = await page.evaluate(() => {
  const tiles = [...document.querySelectorAll("span.size-10.bg-primary\\/10")];
  const catList = document.querySelector("ul.grid");
  const catCols = catList ? getComputedStyle(catList).gridTemplateColumns.split(" ").length : 0;
  const acctTile = [...document.querySelectorAll("li span.size-9")][0];
  return {
    sectionIconTiles: tiles.length,
    categoryGridCols: catCols,
    accountTileText: acctTile ? acctTile.textContent : null,
    accountTileColor: acctTile ? getComputedStyle(acctTile).color : null,
  };
});

// Assistant: open first conversation, then AI bubbles should exist
await page.goto(`${WEB}/assistant`, { waitUntil: "networkidle", timeout: 45000 });
await page.waitForTimeout(600);
const convoBtn = page.locator("aside button").filter({ hasText: /Percakapan|./ }).first();
try {
  await page.locator("aside ul button").first().click({ timeout: 5000 });
  await page.waitForTimeout(800);
} catch {
  /* conversation list may be empty */
}
out.assistant = await page.evaluate(() => {
  const bubbles = [...document.querySelectorAll("div")].filter(
    (d) => String(d.className).includes("rounded-tl-sm") && String(d.className).includes("bg-surface-low"),
  );
  return { aiBubbles: bubbles.length, sample: bubbles[0]?.textContent?.slice(0, 50) ?? null };
});

await browser.close();
console.log(JSON.stringify(out, null, 2));

const pass =
  out.savings.goalHeaderIconTile &&
  out.savings.percentText &&
  out.savings.percentColor === "rgb(78, 222, 163)" && // on_track <80% → primary
  out.profile.sectionIconTiles >= 3 &&
  out.profile.categoryGridCols === 2 &&
  out.profile.accountTileText &&
  out.assistant.aiBubbles >= 1;
console.log(pass ? "POLISH CHECKS PASS" : "POLISH CHECK FAILED");
process.exit(pass ? 0 : 1);
