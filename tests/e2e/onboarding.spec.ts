import { test, expect } from "@playwright/test";
import { api, defaultAccountId, openAs, startClean } from "./helpers";

// Plan.md §12.5 flow 1: onboarding → cash account + default categories →
// manual expense → dashboard changes.
test("onboarding provisions defaults, manual expense updates the dashboard", async ({ page }) => {
  const token = await startClean();

  // First visit provisions profile, Cash account and default categories.
  await openAs(page, token, "/home");
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  // The balance label exists twice by design: hero card (below xl) and the KPI
  // strip (xl and up). Assert the one actually visible at this viewport.
  await expect(page.getByText(/Total saldo/).locator("visible=true").first()).toBeVisible();

  const accounts = await api(token, "GET", "/accounts");
  const cash = (accounts.body.data as Array<{ name: string; isDefault: boolean }>)
    .find((a) => a.name === "Cash");
  expect(cash, "default Cash account must be provisioned").toBeTruthy();
  expect(cash!.isDefault).toBe(true);

  const categories = await api(token, "GET", "/categories");
  expect((categories.body.data as unknown[]).length).toBeGreaterThanOrEqual(9);

  // Record an expense through the real form.
  await page.getByRole("button", { name: "Catat transaksi" }).click();
  await page.locator("#tx-amount").fill("45000");
  await page.locator("#tx-description").fill("E2E onboarding");
  await page.getByRole("button", { name: "Simpan transaksi" }).click();

  // The UI only reports success after the server confirms.
  await expect(page.getByText("berhasil dicatat")).toBeVisible();
  await expect(page.getByText("E2E onboarding").first()).toBeVisible();

  const transactions = await api(token, "GET", "/transactions?limit=50");
  const rows = transactions.body.data.items as Array<{ description: string; amount: number }>;
  expect(rows.filter((r) => r.description === "E2E onboarding")).toHaveLength(1);

  // Dashboard numbers now include the new expense.
  const dashboard = await api(token, "GET", "/analytics/dashboard?period=this_month");
  expect(dashboard.body.data.summary.expense).toBeGreaterThanOrEqual(45000);
});

// ISSUE-25 #6: DistributionBars di beranda — pill persentasenya harus sama
// dengan yang dikembalikan API. Sebelumnya komponen ini hanya diverifikasi
// lewat Lighthouse, e2e regresi, dan pemeriksaan DOM (docs/qa.md §11).
test("home distribution pills match the API", async ({ page }) => {
  const token = await startClean();
  const accountId = await defaultAccountId(token);

  const categories = await api(token, "GET", "/categories");
  const expenseCats = (categories.body.data as Array<{ id: string; kind: string; name: string }>)
    .filter((c) => c.kind === "expense")
    .slice(0, 2);
  expect(expenseCats.length, "butuh minimal 2 kategori expense").toBeGreaterThanOrEqual(2);

  // Dua pengeluaran pada kategori BERBEDA supaya bar-nya benar-benar bertumpuk,
  // bukan satu segmen penuh.
  for (const [index, category] of expenseCats.entries()) {
    const created = await api(
      token,
      "POST",
      "/transactions",
      {
        type: "expense",
        amount: index === 0 ? 60000 : 40000,
        description: `Distribusi ${index}`,
        accountId,
        categoryId: category.id,
        source: "manual",
      },
      `dist-${index}-${Date.now()}`,
    );
    expect(created.status, `buat transaksi kategori ${category.name}`).toBe(201);
  }

  await openAs(page, token, "/home");
  const card = page.locator("section").filter({ hasText: "Kategori terbesar" });
  await expect(card).toBeVisible();

  const dashboard = await api(token, "GET", "/analytics/dashboard?period=this_month");
  const top = dashboard.body.data.topCategories as Array<{ name: string; percent: number }>;
  expect(top.length, "harus ada kategori untuk didistribusikan").toBeGreaterThan(0);

  for (const category of top) {
    const pill = card.locator("ul li").filter({ hasText: category.name });
    await expect(pill, `pill "${category.name}" harus tampil`).toBeVisible();
    await expect(pill).toContainText(`${Math.round(category.percent * 100)}%`);
  }
});
