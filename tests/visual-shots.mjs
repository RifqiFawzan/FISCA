// Visual verification for the desktop adaptation pass (antislop R-35).
// Registers a fresh user, seeds real data through the API, then screenshots
// every adapted page at desktop width (plus Home on mobile as a regression
// glance). Console errors and page errors are printed as a report.
import { chromium } from "@playwright/test";

const WEB = process.env.WEB_URL ?? "http://localhost:3001";
const API = process.env.API_URL ?? "http://localhost:8080/api/v1";
const OUT = process.env.OUT_DIR ?? "C:/Users/zimig/AppData/Local/Temp/opencode/shots";
const PASSWORD = "kata-sandi-e2e-123";

const pages = [
  ["home", "/home"],
  ["history", "/history"],
  ["reports", "/reports"],
  ["savings", "/savings"],
  ["profile", "/profile"],
  ["assistant", "/assistant"],
];

async function api(token, method, path, body, idempotencyKey) {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(body ? { "Content-Type": "application/json" } : {}),
      ...(idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  const json = text ? JSON.parse(text) : null;
  if (res.status >= 400) {
    throw new Error(`${method} ${path} -> ${res.status} ${text.slice(0, 300)}`);
  }
  return json;
}

async function seed() {
  const email = `visual-${Date.now()}@fisca.test`;
  const reg = await api(null, "POST", "/auth/register", { email, password: PASSWORD });
  const token = reg.data.token;

  const accounts = await api(token, "GET", "/accounts");
  const accountId = accounts.data.find((a) => a.isDefault)?.id ?? accounts.data[0].id;
  const categories = await api(token, "GET", "/categories");
  const expenseCat = categories.data.find((c) => c.kind === "expense");
  const incomeCat = categories.data.find((c) => c.kind === "income");

  const today = new Date();
  const day = (n) => {
    const d = new Date(today);
    d.setDate(d.getDate() - n);
    return d.toISOString();
  };

  await api(token, "POST", "/transactions", {
    type: "expense", amount: 425000, description: "Makan siang tim",
    accountId, categoryId: expenseCat?.id, source: "manual", occurredAt: day(0),
  }, `vis-1-${Date.now()}`);
  await api(token, "POST", "/transactions", {
    type: "expense", amount: 180000, description: "Kopi Kenangan",
    accountId, categoryId: expenseCat?.id, source: "voice", occurredAt: day(1),
  }, `vis-2-${Date.now()}`);
  await api(token, "POST", "/transactions", {
    type: "expense", amount: 850000, description: "Langganan alat kerja",
    accountId, categoryId: expenseCat?.id, source: "chat", occurredAt: day(3),
  }, `vis-3-${Date.now()}`);
  await api(token, "POST", "/transactions", {
    type: "income", amount: 18500000, description: "Gaji bulanan",
    accountId, categoryId: incomeCat?.id, source: "manual", occurredAt: day(5),
  }, `vis-4-${Date.now()}`);

  try {
    await api(token, "POST", "/budgets", { period: "this_month", overallLimit: 12000000 });
  } catch (error) {
    console.log("budget seed skipped:", error.message);
  }
  try {
    const deadline = new Date(today);
    deadline.setDate(deadline.getDate() + 180);
    const created = await api(token, "POST", "/savings-goals", {
      name: "Dana Darurat 6 Bulan",
      targetAmount: 30000000,
      deadline: deadline.toISOString().slice(0, 10),
      accountId,
    });
    console.log("savings created: " + (created?.data?.id ?? JSON.stringify(created).slice(0, 120)));
  } catch (error) {
    console.log("savings seed skipped:", error.message);
  }

  return token;
}

async function waitForWeb(timeoutMs) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    try {
      const res = await fetch(`${WEB}/login`, { redirect: "manual" });
      if (res.status < 500) return;
    } catch {
      // not up yet
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  throw new Error(`web not ready at ${WEB} after ${timeoutMs}ms`);
}

async function main() {
  await waitForWeb(90000);
  const token = await seed();
  const browser = await chromium.launch({ channel: "chrome" });
  const problems = [];

  async function shoot(name, path, viewport) {
    const context = await browser.newContext({ viewport });
    const page = await context.newPage();
    page.on("console", (msg) => {
      if (msg.type() === "error") problems.push(`[console:${name}] ${msg.text()}`);
    });
    page.on("pageerror", (err) => problems.push(`[pageerror:${name}] ${err.message}`));
    await page.addInitScript((value) => {
      window.localStorage.setItem("fisca.token", value);
    }, token);
    await page.goto(`${WEB}${path}`, { waitUntil: "networkidle", timeout: 45000 });
    await page.waitForTimeout(600);
    const url = page.url();
    const h1 = (await page.locator("h1").first().textContent().catch(() => "(none)")) ?? "(none)";
    await page.screenshot({ path: `${OUT}/${name}.png`, fullPage: true });
    console.log(`shot ${name} url=${url} h1=${h1.trim().slice(0, 40)}`);
    await context.close();
  }

  const desktop = { width: 1600, height: 1000 };
  const mobile = { width: 390, height: 844 };
  for (const [name, path] of pages) await shoot(name, path, desktop);
  await shoot("home-mobile", "/home", mobile);
  await shoot("assistant-mobile", "/assistant", mobile);

  await browser.close();
  if (problems.length) {
    console.log("PROBLEMS:");
    for (const p of problems) console.log("  " + p);
    process.exitCode = 1;
  } else {
    console.log("NO CONSOLE/PAGE ERRORS");
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
