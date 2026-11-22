import { test, expect } from "@playwright/test";
import { randomUUID } from "node:crypto";

import { API_BASE, api } from "./helpers";

// Plan.md §12.5 tidak punya alur khusus untuk form masuk, dan selama ini e2e
// selalu menulis token langsung ke localStorage lewat openAs() — sehingga form
// ini tidak pernah tersentuh test. ISSUE-22 menutup celah itu.
//
// Panggilan auth berharga: bucket rate limit endpoint auth sengaja ketat
// (RATE_LIMIT_RPS/4, lihat internal/server/server.go). Karena itu ketiga test
// disusung hemat — hanya test pertama yang benar-benar mendaftar.

const PASSWORD = "kata-sandi-e2e-123";

function newEmail(): string {
  // Prefix `e2e-` membuat akun ini terhapus oleh resetTestUsers saat teardown.
  return `e2e-${randomUUID()}@fisca.test`;
}

/** Membuat akun lewat API sebagai persiapan — lebih hemat daripada lewat form. */
async function registerViaApi(email: string, password: string): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, displayName: "Persiapan Uji" }),
  });
  if (res.status !== 201) {
    throw new Error(`persiapan register gagal: ${res.status} ${await res.text()}`);
  }
}

async function enterSignupMode(page: import("@playwright/test").Page): Promise<void> {
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "Masuk" })).toBeVisible();
  await page.getByRole("button", { name: "Belum punya akun? Daftar" }).click();
  await expect(page.getByRole("heading", { name: "Buat akun" })).toBeVisible();
}

// Satu alur penuh: daftar → sapaan memakai nama depan → keluar → masuk kembali.
// Menggabungkan ketiganya menjaga jumlah panggilan auth tetap minim.
test("sign up greets by first name, then sign out and back in", async ({ page }) => {
  const email = newEmail();
  await enterSignupMode(page);

  await page.locator("#signup-firstname").fill("Nadia");
  await page.locator("#signup-lastname").fill("Putri");
  await page.locator("#login-email").fill(email);
  await page.locator("#login-password").fill(PASSWORD);
  await page.locator("#signup-confirm").fill(PASSWORD);
  await page.getByRole("button", { name: "Daftar", exact: true }).click();

  await expect(page).toHaveURL(/\/home/);
  await expect(page.getByText("Halo, Nadia")).toBeVisible();

  // Nama lengkap tersimpan di server, bukan hanya di layar.
  const token = await page.evaluate(() => localStorage.getItem("fisca.token"));
  expect(token, "token sesi harus tersimpan setelah pendaftaran").toBeTruthy();
  const me = await api(token!, "GET", "/me");
  expect(me.status).toBe(200);
  expect(me.body.data.displayName).toBe("Nadia Putri");
  expect(me.body.data.email).toBe(email);

  // Keluar: sesi dicabut di server, lalu form masuk kembali menampilkan sapaan.
  await page.goto("/profile");
  await page.getByRole("button", { name: "Keluar" }).click();
  await expect(page).toHaveURL(/\/login/);
  expect(await page.evaluate(() => localStorage.getItem("fisca.token"))).toBeNull();

  await page.locator("#login-email").fill(email);
  await page.locator("#login-password").fill(PASSWORD);
  await page.getByRole("button", { name: "Masuk", exact: true }).click();
  await expect(page).toHaveURL(/\/home/);
  await expect(page.getByText("Halo, Nadia")).toBeVisible();
});

// Semua kasus gagal harus berhenti SEBELUM request terkirim: tidak ada token
// yang terbit, dan halaman tetap di /login.
test("sign up blocks invalid input before sending anything", async ({ page }) => {
  await enterSignupMode(page);

  // Discope ke alert: teks yang sama juga muncul sebagai hint di bawah field
  // ("Minimal 8 karakter"), sehingga carian global melanggar strict mode.
  const expectBlocked = async (message: string) => {
    await page.getByRole("button", { name: "Daftar", exact: true }).click();
    await expect(page.getByRole("alert").filter({ hasText: message })).toBeVisible();
    await expect(page).toHaveURL(/\/login/);
    expect(await page.evaluate(() => localStorage.getItem("fisca.token"))).toBeNull();
  };

  // 1) konfirmasi sandi tidak cocok
  await page.locator("#signup-firstname").fill("Nadia");
  await page.locator("#login-email").fill(newEmail());
  await page.locator("#login-password").fill(PASSWORD);
  await page.locator("#signup-confirm").fill("kata-sandi-berbeda-999");
  await expectBlocked("Kata sandi tidak sama");

  // 2) nama depan kosong
  await page.locator("#signup-firstname").fill("");
  await page.locator("#signup-confirm").fill(PASSWORD);
  await expectBlocked("Nama depan wajib");

  // 3) kata sandi terlalu pendek
  await page.locator("#signup-firstname").fill("Nadia");
  await page.locator("#login-password").fill("abc1234");
  await page.locator("#signup-confirm").fill("abc1234");
  await expectBlocked("minimal 8 karakter");
});

// Tombol mata harus mengubah hanya field-nya sendiri — dua field sandi di mode
// daftar tidak boleh ikut terbuka.
test("password eye reveals only its own field", async ({ page }) => {
  await enterSignupMode(page);

  const primary = page.locator("#login-password");
  const confirm = page.locator("#signup-confirm");
  const eyes = page.locator('button[aria-label="Lihat kata sandi"]');

  await expect(primary).toHaveAttribute("type", "password");
  await expect(confirm).toHaveAttribute("type", "password");
  await expect(eyes).toHaveCount(2);
  await expect(eyes.first()).toHaveAttribute("aria-pressed", "false");

  // Target sentuh 44px sesuai kebutuhan ponsel (DESIGN.md §105).
  const box = await eyes.first().boundingBox();
  expect(box!.height).toBeGreaterThanOrEqual(44);
  expect(box!.width).toBeGreaterThanOrEqual(44);

  await eyes.first().click();
  await expect(primary).toHaveAttribute("type", "text");
  await expect(confirm).toHaveAttribute("type", "password");
  await expect(page.locator('button[aria-label="Sembunyikan kata sandi"]')).toHaveCount(1);
  await expect(page.locator('button[aria-label="Sembunyikan kata sandi"]')).toHaveAttribute(
    "aria-pressed",
    "true",
  );

  // Menekan lagi menyembunyikannya kembali.
  await page.locator('button[aria-label="Sembunyikan kata sandi"]').click();
  await expect(primary).toHaveAttribute("type", "password");
});

// Penolakan dari SERVER, bukan validasi klien: akunnya nyata, sandinya salah.
test("sign in shows the server message for a wrong password", async ({ page }) => {
  const email = newEmail();
  await registerViaApi(email, PASSWORD);

  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "Masuk" })).toBeVisible();
  await page.locator("#login-email").fill(email);
  await page.locator("#login-password").fill("kata-sandi-salah-999");
  await page.getByRole("button", { name: "Masuk", exact: true }).click();

  // Pesan seragam untuk semua kasus gagal, agar tidak membocorkan apakah
  // email terdaftar (AC ISSUE-15 #2).
  await expect(
    page.getByRole("alert").filter({ hasText: "Email atau kata sandi salah" }),
  ).toBeVisible();
  await expect(page).toHaveURL(/\/login/);
  expect(await page.evaluate(() => localStorage.getItem("fisca.token"))).toBeNull();
});

// Email terdaftar: registrasi tidak bisa menyembunyikannya, jadi pengguna
// diarahkan untuk masuk — lewat notice, bukan error.
test("sign up points to sign in when the email already exists", async ({ page }) => {
  const email = newEmail();
  await registerViaApi(email, PASSWORD);

  await enterSignupMode(page);
  await page.locator("#signup-firstname").fill("Nadia");
  await page.locator("#login-email").fill(email);
  await page.locator("#login-password").fill(PASSWORD);
  await page.locator("#signup-confirm").fill(PASSWORD);
  await page.getByRole("button", { name: "Daftar", exact: true }).click();

  await expect(page.getByText("Email sudah terdaftar. Silakan masuk.")).toBeVisible();
  await expect(page).toHaveURL(/\/login/);
  expect(await page.evaluate(() => localStorage.getItem("fisca.token"))).toBeNull();
});

// ISSUE-25 #4: AC ISSUE-15 #5 menuntut "Sesi Anda sudah berakhir" saat token
// kedaluwarsa/tidak sah. Sebelumnya hanya diverifikasi manual di browser.
test("an invalid token shows the expired-session message", async ({ page }) => {
  // addInitScript berjalan sebelum skrip aplikasi, jadi token sudah terpasang
  // tepat saat lib/session.ts membacanya.
  await page.addInitScript(() => {
    window.localStorage.setItem("fisca.token", "token-tidak-sah-sudah-lewat-masa-berlaku");
  });

  await page.goto("/home");
  await expect(page.getByText("Sesi Anda sudah berakhir")).toBeVisible();
  await expect(page.getByText("Silakan masuk kembali")).toBeVisible();
});

// ISSUE-25 #5: mata password pada mode MASUK (satu field) — sebelumnya yang
// diuji hanya mode daftar (dua field).
test("password eye works in sign-in mode", async ({ page }) => {
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "Masuk" })).toBeVisible();

  const password = page.locator("#login-password");
  const eye = page.locator('button[aria-label="Lihat kata sandi"]');

  // Mode masuk hanya punya satu field sandi, jadi satu tombol mata.
  await expect(eye).toHaveCount(1);
  await expect(password).toHaveAttribute("type", "password");

  await eye.click();
  await expect(password).toHaveAttribute("type", "text");
  await expect(page.locator('button[aria-label="Sembunyikan kata sandi"]')).toHaveAttribute(
    "aria-pressed",
    "true",
  );

  await page.locator('button[aria-label="Sembunyikan kata sandi"]').click();
  await expect(password).toHaveAttribute("type", "password");
});
