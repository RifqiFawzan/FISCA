import { test, expect } from "@playwright/test";
import { api, openAs, startClean } from "./helpers";

// Plan.md §12.5 flow 3: ambiguous input → draft → clarification → stored.
// The draft must not touch balances, reports or budgets before it is resolved
// (PRD §23 exit 7).
test("ambiguous input becomes a draft, the reply completes it", async ({ page }) => {
  const token = await startClean();

  await openAs(page, token, "/assistant");
  await expect(page.getByText("Halo, saya Asisten FISCA")).toBeVisible();

  // "catat 25rb" has an amount but no description → clarification, no row yet.
  await page.getByLabel("Tulis pesan").fill("catat 25rb");
  await page.getByRole("button", { name: "Kirim pesan" }).click();
  // Balasan melewati pemanggilan AI yang boleh memakan waktu sampai
  // AI_TIMEOUT_SECONDS server (20s); default expect10s terlalu ketat dan
  // membuat spesifikasi ini flaky saat latensi provider naik.
  await expect(page.getByText("Butuh konfirmasi")).toBeVisible({ timeout: 25000 });

  const before = await api(token, "GET", "/transactions?limit=50");
  expect((before.body.data.items as unknown[]).length, "draft must not create a row").toBe(0);

  const accounts = await api(token, "GET", "/accounts");
  const balances = (accounts.body.data as Array<{ balance: number }>).map((a) => a.balance);
  expect(balances.every((b) => b === 0), "draft must not change the balance").toBe(true);

  // The reply fills the missing description and saves.
  await page.getByLabel("Tulis pesan").fill("kopi susu uji");
  await page.getByRole("button", { name: "Kirim pesan" }).click();
  await expect(page.getByText("sudah dicatat").first()).toBeVisible({ timeout: 25000 });

  const after = await api(token, "GET", "/transactions?limit=50");
  const rows = after.body.data.items as Array<{ description: string; amount: number }>;
  expect(rows).toHaveLength(1);
  // Optional chaining supaya bila assertion di atas kendur, kegagalannya terbaca
  // sebagai selisih nilai — bukan TypeError "cannot read property of undefined".
  expect(rows[0]?.description).toBe("Kopi susu uji");
  expect(rows[0]?.amount).toBe(25000);
});
