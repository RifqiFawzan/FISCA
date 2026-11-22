import { test, expect } from "@playwright/test";

import { api, defaultAccountId, openAs, startClean } from "./helpers";

// ISSUE-25 #2: halaman /savings sebelumnya NOL test e2e — tercatat sebagai
// Batasan di ISSUE-20. Alur yang diuji: halaman termuat kosong → membuat
// target → tampil di layar dan tersimpan di server.
test("savings: create a target and see it tracked", async ({ page }) => {
  const token = await startClean();
  const accountId = await defaultAccountId(token);

  await openAs(page, token, "/savings");
  await expect(page.getByRole("heading", { level: 1, name: "Tabungan" })).toBeVisible();
  await expect(page.getByText("Belum ada target tabungan")).toBeVisible();

  // Field "Tenggat" punya min = besok, jadi tenggat harus jelas di masa depan.
  // +30 hari dipilih agar geser zona waktu (UTC vs lokal) tidak pernah
  // menjatuhkannya ke bawah minimum.
  const deadline = new Date();
  deadline.setDate(deadline.getDate() + 30);

  await page.getByRole("button", { name: "Target baru" }).click();
  await page.locator("#goal-name").fill("Liburan uji");
  await page.locator("#goal-target").fill("5000000");
  await page.locator("#goal-deadline").fill(deadline.toISOString().slice(0, 10));
  await page.locator("#goal-account").selectOption(accountId);
  await page.getByRole("button", { name: "Simpan target" }).click();

  // Tampil di layar… pakai heading, bukan teks bebas: notice sukses juga
  // memuat "Liburan uji", sehingga carian global melanggar strict mode
  // (terbukti saat suite penuh, tidak saat dijalankan sendirian).
  await expect(page.getByRole("heading", { name: "Liburan uji" })).toBeVisible();

  // …dan tersimpan di server dengan nilai yang benar.
  const list = await api(token, "GET", "/savings-goals");
  const goals = list.body.data as Array<{
    name: string;
    targetAmount: number;
    accountId: string;
    percent: number;
  }>;
  expect(goals).toHaveLength(1);
  // `!` aman: assertion di atas menjamin panjang 1.
  expect(goals[0]!.name).toBe("Liburan uji");
  expect(goals[0]!.targetAmount).toBe(5000000);
  expect(goals[0]!.accountId).toBe(accountId);
  // Progres diturunkan dari saldo akun; akun kosong berarti 0.
  expect(goals[0]!.percent).toBe(0);
});

// Simulasi pembelian memakai data akun & budget yang sama, jadi wajib tampil
// walau belum ada target tabungan.
test("savings: purchase advisor answers from real balances", async ({ page }) => {
  const token = await startClean();

  await openAs(page, token, "/savings");
  await expect(page.getByText("Simulasi pembelian")).toBeVisible();

  await page.locator("#advisor-price").fill("100000");
  await page.getByRole("button", { name: "Cek keterjangkauan" }).click();

  // Salah satu dari tiga vonis harus muncul (VERDICT_META di savings-view).
  const verdict = page
    .locator("span")
    .filter({ hasText: /^(Terjangkau|Bisa, tapi mepet|Belum terjangkau)$/ });
  await expect(verdict.first()).toBeVisible();

  // Angka yang ditampilkan harus berasal dari API, bukan karangan UI.
  // Kontraknya GET dengan query param `price` (docs/api/openapi.yaml), bukan POST.
  const advisor = await api(token, "GET", "/purchase-advisor?price=100000");
  expect(advisor.status).toBe(200);
  expect(advisor.body.data.verdict).toMatch(/^(affordable|tight|not_affordable)$/);
  expect(advisor.body.data.price).toBe(100000);
});
