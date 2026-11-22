import { test, expect } from "@playwright/test";
import { openAs, seedExpense, startClean } from "./helpers";

// Plan.md §12.5 flow 5 (largest transaction = database query) and flow 6
// (unusual spending states its limitations when data is insufficient).
async function seededUser() {
  const token = await startClean();
  await seedExpense(token, "E2E kecil", 12000);
  await seedExpense(token, "E2E terbesar", 987654);
  await seedExpense(token, "E2E sedang", 54000);
  return token;
}

test("largest transaction shown on reports matches the database", async ({ page }) => {
  const token = await seededUser();

  await openAs(page, token, "/reports");
  await expect(page.getByRole("heading", { level: 1, name: "Laporan" })).toBeVisible();

  const largestCard = page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Transaksi terbesar" }) });
  await expect(largestCard).toBeVisible();
  await expect(largestCard.getByText("E2E terbesar")).toBeVisible();

  // The same value comes straight from the analytics query.
  const API = process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
  const largest = await fetch(
    `${API}/analytics/largest-transactions?period=this_month&type=expense&limit=1`,
    { headers: { Authorization: `Bearer ${token}` } },
  ).then((r) => r.json());
  expect(largest.data.items[0].amount).toBe(987654);
  expect(largest.data.items[0].description).toBe("E2E terbesar");
});

test("unusual spending states the data limitation", async ({ page }) => {
  const token = await seededUser();

  await openAs(page, token, "/reports");

  const unusualCard = page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Pengeluaran tidak biasa" }) });
  await expect(unusualCard).toBeVisible();
  // A single month of history is not enough for a median+MAD baseline.
  await expect(unusualCard.getByText("Data belum cukup")).toBeVisible();
});
