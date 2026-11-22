import { test, expect } from "@playwright/test";
import { api, dbQuery, ensureBudget, openAs, seedExpense, startClean } from "./helpers";

// Plan.md §12.5 flow 7 (proposal → approve → budget changes + audit log) and
// flow 8 (over-budget transaction is stored and warns, PRD §23 exit 16).
test("over-budget transaction is stored with a warning", async ({ page }) => {
  const token = await startClean();
  await ensureBudget(token, 100000);

  await openAs(page, token, "/assistant");
  await page.getByLabel("Tulis pesan").fill("catat beli laptop 900rb");
  await page.getByRole("button", { name: "Kirim pesan" }).click();

  // The transaction is stored AND a warning is shown — never blocked.
  await expect(page.getByText("sudah dicatat").first()).toBeVisible();
  await expect(page.getByText("terlampaui").first()).toBeVisible();

  const transactions = await api(token, "GET", "/transactions?limit=50");
  const rows = transactions.body.data.items as Array<{ description: string; amount: number }>;
  const stored = rows.find((r) => r.amount === 900000);
  expect(stored, "over-budget transaction must be stored").toBeTruthy();
});

test("budget proposal needs approval, then changes the budget and writes audit", async ({ page }) => {
  const token = await startClean();
  await ensureBudget(token, 100000);

  const before = await api(token, "GET", "/budgets");
  const versionBefore = before.body.data[0].version;
  expect(before.body.data[0].overallLimit).toBe(100000);

  // The assistant only proposes; nothing changes yet.
  await openAs(page, token, "/assistant");
  await page.getByLabel("Tulis pesan").fill("ubah budget jadi 2jt");
  await page.getByRole("button", { name: "Kirim pesan" }).click();
  await expect(page.getByText("Usulan budget baru")).toBeVisible();

  const pending = await api(token, "GET", "/budgets");
  expect(pending.body.data[0].overallLimit, "proposal must not apply by itself").toBe(100000);

  // Approve from the budget screen.
  await openAs(page, token, "/budget");
  await expect(page.getByText("Menunggu")).toBeVisible();
  await page.getByRole("button", { name: "Setujui" }).click();
  await expect(page.getByText("Proposal disetujui")).toBeVisible();

  const after = await api(token, "GET", "/budgets");
  expect(after.body.data[0].overallLimit).toBe(2000000);
  expect(after.body.data[0].version).toBeGreaterThan(versionBefore);

  // The approval is recorded in the audit trail.
  const approvals = dbQuery(
    "SELECT count(*) FROM audit_logs WHERE entity_type = 'budget' AND action = 'approve_proposal'",
  );
  expect(Number(approvals)).toBeGreaterThanOrEqual(1);
});

// ISSUE-23: status budget punya tiga tahap — Normal / Mendekati Limit / Over
// Limit — dan warna bar progres memakai statusForPercent yang sama, sehingga
// bar dan badge mustahil berbeda. Sebelumnya hanya dua tahap dan logikanya
// ditulis inline terduplikasi.
//
// Satu user per kasus karena startClean() menghapus seluruh user E2E lebih dulu.
test("budget badge renders all three tones", async ({ page }) => {
  const cases = [
    { limit: 1_000_000, spend: 0, badge: "0% Normal" },
    { limit: 10_000, spend: 8_500, badge: "85% Mendekati Limit" },
    { limit: 10_000, spend: 20_000, badge: "200% Over Limit" },
  ];

  for (const c of cases) {
    const token = await startClean();
    await ensureBudget(token, c.limit);
    if (c.spend > 0) await seedExpense(token, `tone-${c.spend}`, c.spend);

    await openAs(page, token, "/budget");
    await expect(page.getByText(c.badge)).toBeVisible();

    // Persentase yang dikembalikan API harus sama dengan yang dirender, supaya
    // pembulatan di lapisan mana pun tidak menyembunyikan selisih.
    const list = await api(token, "GET", "/budgets");
    expect(`${Math.round(list.body.data[0].percent * 100)}%`).toBe(c.badge.split(" ")[0]);
  }
});
