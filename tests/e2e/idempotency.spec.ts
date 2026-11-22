import { test, expect } from "@playwright/test";
import { api, openAs, startClean } from "./helpers";

// Plan.md §12.5 flow 4: a retry must not create a second transaction
// (PRD §23 exit 8).
test("the same idempotency key stores one transaction", async () => {
  const token = await startClean();
  const key = `e2e-idem-${Date.now()}`;
  const body = {
    type: "expense",
    amount: 77000,
    description: "E2E idempotency",
    source: "manual",
  };

  const first = await api(token, "POST", "/transactions", body, key);
  const second = await api(token, "POST", "/transactions", body, key);

  expect(first.status).toBe(201);
  expect(second.status).toBe(201);
  expect(second.body.data.transaction.id).toBe(first.body.data.transaction.id);

  const list = await api(token, "GET", "/transactions?limit=50");
  const rows = list.body.data.items as Array<{ description: string }>;
  expect(rows.filter((r) => r.description === "E2E idempotency")).toHaveLength(1);
});

test("a different body with the same key is rejected", async () => {
  const token = await startClean();
  const key = `e2e-idem-mismatch-${Date.now()}`;
  const first = await api(token, "POST", "/transactions", {
    type: "expense", amount: 1000, description: "E2E asli", source: "manual",
  }, key);
  expect(first.status, "first write must succeed before we can clash").toBe(201);

  const clash = await api(token, "POST", "/transactions", {
    type: "expense", amount: 9999, description: "E2E beda", source: "manual",
  }, key);

  expect(clash.status).toBe(409);
  expect(clash.body.error.code).toBe("IDEMPOTENCY_MISMATCH");
});

test("double submit in the form stores one row", async ({ page }) => {
  const token = await startClean();
  await openAs(page, token, "/home");
  await page.getByRole("button", { name: "Catat transaksi" }).click();
  await page.locator("#tx-amount").fill("88000");
  await page.locator("#tx-description").fill("E2E double submit");

  const submit = page.getByRole("button", { name: "Simpan transaksi" });
  await submit.click();
  // A second click while the first is in flight must be harmless: the button is
  // disabled for the duration of the request.
  await submit.click({ force: true, timeout: 2000 }).catch(() => undefined);

  await expect(page.getByText("berhasil dicatat")).toBeVisible();

  const list = await api(token, "GET", "/transactions?limit=50");
  const rows = list.body.data.items as Array<{ description: string }>;
  expect(rows.filter((r) => r.description === "E2E double submit")).toHaveLength(1);
});
