import { test, expect } from "@playwright/test";

import { api, openAs, startClean } from "./helpers";

// ISSUE-25 #3: halaman /profile sebelumnya hanya DIKUNJUNGI untuk menekan
// tombol "Keluar" (di auth-form.spec) — isinya tidak pernah diuji.
test("profile shows the signed-in user's own data", async ({ page }) => {
  const token = await startClean();

  const me = await api(token, "GET", "/me");
  const accounts = await api(token, "GET", "/accounts");
  expect(me.status).toBe(200);

  await openAs(page, token, "/profile");
  await expect(page.getByRole("heading", { level: 1, name: "Profil & Pengaturan" })).toBeVisible();

  // Identitas harus datang dari /me, bukan dari teks statis.
  // `exact` wajib: untuk akun e2e, displayName adalah bagian sebelum "@" pada
  // email, sehingga pencarian tanpa exact cocok ke kedua elemen sekaligus.
  await expect(page.getByText(me.body.data.displayName, { exact: true })).toBeVisible();
  await expect(page.getByText(me.body.data.email, { exact: true })).toBeVisible();

  // Akun bawaan yang dibuat saat registrasi ikut terdaftar di halaman ini.
  const cash = (accounts.body.data as Array<{ name: string }>).find((a) => a.name === "Cash");
  expect(cash, "akun Cash bawaan harus ada").toBeTruthy();
  await expect(page.getByText("Cash", { exact: false }).first()).toBeVisible();

  // Kartu sesi tersedia — jalur keluarnya sendiri sudah diuji di auth-form.spec.
  await expect(page.getByRole("button", { name: "Keluar" })).toBeVisible();
});

// Halaman harus menampilkan data akun yang benar setelah akun baru dibuat,
// bukan hanya data awal saat registrasi.
test("profile lists accounts created after sign-up", async ({ page }) => {
  const token = await startClean();

  await openAs(page, token, "/profile");
  await expect(page.getByRole("heading", { level: 1, name: "Profil & Pengaturan" })).toBeVisible();

  const created = await api(token, "POST", "/accounts", {
    name: "Bank Uji E2E",
    type: "bank",
    openingBalance: 750000,
  });
  expect(created.status).toBe(201);

  // Muat ulang agar daftar akun mengambil data terbaru.
  await page.reload();
  await expect(page.getByText("Bank Uji E2E")).toBeVisible();

  const list = await api(token, "GET", "/accounts");
  const names = (list.body.data as Array<{ name: string }>).map((a) => a.name);
  expect(names).toContain("Bank Uji E2E");
  expect(names).toContain("Cash");
});
