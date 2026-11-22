// Menangkap screenshot semua halaman FISCA (login + terautentikasi) untuk
// dibandingkan dengan design-refs/ (screenshot Stitch).
// Dipakai: node scripts/screenshot-web.cjs <email> <password>
// Output: %TEMP%/opencode/web-shots/*.png (di luar repo agar tidak menumpuk).
const { chromium } = require("@playwright/test");
const path = require("node:path");
const os = require("node:os");
const fs = require("node:fs");

const BASE = "http://localhost:3000";
const OUT = path.join(os.tmpdir(), "opencode", "web-shots");

const email = process.argv[2];
const password = process.argv[3];
if (!email || !password) {
  console.error("pakai: node scripts/screenshot-web.cjs <email> <password>");
  process.exit(1);
}

// Label -> path halaman. Sign-up ditangkap sebagai mode di halaman login.
const PAGES = [
  ["home", "/home"],
  ["history", "/history"],
  ["reports", "/reports"],
  ["budget", "/budget"],
  ["savings", "/savings"],
  ["profile", "/profile"],
  ["assistant", "/assistant"],
];

async function settle(page) {
  await page.waitForLoadState("networkidle", { timeout: 15000 }).catch(() => {});
}

(async () => {
  fs.mkdirSync(OUT, { recursive: true });
  // Chrome sistem — sama seperti playwright.config.ts (tanpa unduh browser).
  const browser = await chromium.launch({ channel: "chrome" });
  const ctx = await browser.newContext({
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 2,
  });
  const page = await ctx.newPage();

  // 1. Halaman login (tampilan awal, belum diisi).
  await page.goto(`${BASE}/login`, { waitUntil: "domcontentloaded" });
  await settle(page);
  await page.screenshot({ path: path.join(OUT, "00-login.png"), fullPage: true });

  // 2. Mode daftar (klik tautan "Belum punya akun? Daftar").
  const daftar = page.getByRole("button", { name: /Belum punya akun/i });
  if (await daftar.count()) {
    await daftar.first().click();
    await settle(page);
    await page.screenshot({ path: path.join(OUT, "01-signup.png"), fullPage: true });
    // kembali ke mode masuk
    const masuk = page.getByRole("button", { name: /Sudah punya akun|Masuk/i }).first();
    if (await masuk.count()) await masuk.click().catch(() => {});
    await settle(page);
  }

  // 3. Isi form masuk.
  await page.locator('input[type="email"]').first().fill(email);
  await page.locator("#login-password").fill(password);
  await page.locator('button[type="submit"]').first().click();
  await page.waitForFunction(() => !location.pathname.startsWith("/login"), null, {
    timeout: 15000,
  });

  // 4. Semua halaman terautentikasi.
  for (const [label, route] of PAGES) {
    await page.goto(`${BASE}${route}`, { waitUntil: "domcontentloaded" });
    await settle(page);
    await page.screenshot({ path: path.join(OUT, `${label}.png`), fullPage: true });
    console.log(`ok: ${label} (${route})`);
  }

  await browser.close();
  console.log(`\noutput: ${OUT}`);
})().catch((err) => {
  console.error("GAGAL:", err.message);
  process.exit(1);
});
