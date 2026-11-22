import { test, expect } from "@playwright/test";
import { api, freshToken, openAs, seedExpense, startClean } from "./helpers";

// Plan.md §12.5 flow 9 / PRD §23 exit 19: user A's data must be invisible and
// untouchable for user B.
test("user B cannot see or read user A data", async ({ page }) => {
  const tokenA = await startClean();
  await seedExpense(tokenA, "Rahasia user A", 640000);
  const tokenB = await freshToken();

  // The row exists for A.
  const forA = await api(tokenA, "GET", "/transactions?limit=50");
  const rows = forA.body.data.items as Array<{ id: string; description: string }>;
  expect(rows).toHaveLength(1);
  // `!` aman karena baris di atas menjamin panjang 1; id-nya dipakai sebagai
  // path request di bawah, jadi tidak bisa diganti optional chaining tanpa
  // membuat URL "undefined" yang gagal diam-diam.
  const secretId = rows[0]!.id;

  // B's history screen shows no trace of it.
  await openAs(page, tokenB, "/history");
  await expect(page.getByRole("heading", { level: 1, name: "Riwayat" })).toBeVisible();
  await expect(page.getByText("Rahasia user A")).toHaveCount(0);
  await expect(page.getByText("Belum ada transaksi")).toBeVisible();

  // B's search does not find it either.
  await page.getByLabel("Cari transaksi").fill("Rahasia");
  await page.getByRole("button", { name: "Cari" }).click();
  await expect(page.getByText("Tidak ada transaksi yang cocok")).toBeVisible();

  // Direct API access to A's row is a 404, not a 403 that leaks existence.
  const direct = await api(tokenB, "GET", `/transactions/${secretId}`);
  expect(direct.status).toBe(404);

  // And A's row is untouched by B's attempts.
  const stillThere = await api(tokenA, "GET", `/transactions/${secretId}`);
  expect(stillThere.status).toBe(200);
  expect(stillThere.body.data.transaction.amount).toBe(640000);

  // B sees exactly zero rows.
  const forB = await api(tokenB, "GET", "/transactions?limit=50");
  expect((forB.body.data.items as unknown[]).length).toBe(0);
});
