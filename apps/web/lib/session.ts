/**
 * Session and token storage for the browser.
 *
 * FISCA mengautentikasi dirinya sendiri (ISSUE-15): API menerbitkan token sesi
 * acak, menyimpan hanya hash SHA-256-nya, dan mencabut token itu saat logout.
 * Peramban menyimpan satu token di `fisca.token`.
 *
 * Tidak ada refresh token. Masa berlaku token panjang (lihat SessionTTL di
 * backend) dan validasinya dibaca dari tabel sessions, sehingga logout langsung
 * berlaku. Bila token kedaluwarsa atau tercabut, API menjawab 401 dan
 * lib/api.ts menerjemahkannya menjadi "Sesi Anda sudah berakhir".
 *
 * Jalur Supabase (GoTrue) di bawah DISIMPAN SEBAGAI KODE DORMAN untuk
 * kemungkinan FISCA menjadi proyek SaaS kelak. Tidak ada yang memanggilnya
 * selama AUTH_MODE=session — ia sengaja tidak dihapus (ISSUE-15 konteks #3).
 */

import { useSyncExternalStore } from "react";

const TOKEN_KEY = "fisca.token";

const API_BASE_URL =
  (process.env.NEXT_PUBLIC_API_BASE_URL ?? "").replace(/\/$/, "") ||
  "http://localhost:8080/api/v1";

/** Respons `POST /auth/login` dan `POST /auth/register` (envelope `data`). */
type AuthPayload = {
  token: string;
  expiresAt: string;
  user: { id: string; email: string; displayName: string };
};

export class SessionError extends Error {
  code: string;
  constructor(message: string, code = "SESSION_ERROR") {
    super(message);
    this.name = "SessionError";
    this.code = code;
  }
}

// ---------------------------------------------------------------------------
// token
// ---------------------------------------------------------------------------

// Token changes are broadcast so components can read them without effects.
const listeners = new Set<() => void>();

function notify() {
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function getToken(): string {
  if (typeof window === "undefined") return "";
  return window.localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(token: string | null) {
  if (typeof window === "undefined") return;
  if (token && token.trim()) {
    window.localStorage.setItem(TOKEN_KEY, token.trim());
  } else {
    window.localStorage.removeItem(TOKEN_KEY);
  }
  notify();
}

export function hasToken(): boolean {
  return getToken().length > 0;
}

/**
 * Current token, read from localStorage through an external store so server
 * and client can disagree without a hydration mismatch.
 */
export function useToken(): string {
  return useSyncExternalStore(subscribe, getToken, () => "");
}

/**
 * Whether a session token exists — safe to use as a render guard. The server
 * snapshot is always `false`, so the server render and the client's first
 * (hydration) render always agree; the real value then arrives as a store
 * update right after hydration, the same mechanism as `useToken`. This is
 * what lets the views keep an `if (!authed)` guard without regenerating the
 * tree on every hydration.
 */
function serverNoToken(): boolean {
  return false;
}

export function useHasToken(): boolean {
  return useSyncExternalStore(subscribe, hasToken, serverNoToken);
}

/** Clears local credentials. Panggilan ke server dilakukan oleh signOut. */
export function clearSession() {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(TOKEN_KEY);
  notify();
}

// ---------------------------------------------------------------------------
// auth lokal (jalur aktif)
// ---------------------------------------------------------------------------

async function authRequest(
  path: string,
  options: { body?: unknown; token?: string } = {},
): Promise<AuthPayload> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (options.token) headers.Authorization = `Bearer ${options.token}`;

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}/auth/${path}`, {
      method: "POST",
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
    });
  } catch {
    throw new SessionError("Tidak bisa terhubung ke server.", "NETWORK_ERROR");
  }

  const text = await response.text();
  const body = text
    ? (JSON.parse(text) as { data?: AuthPayload; error?: { code?: string; message?: string } })
    : null;

  if (!response.ok) {
    // Pesan datang dari backend supaya "Email atau kata sandi salah" untuk
    // semua kasus gagal (AC ISSUE-15 #2) sampai utuh ke pengguna.
    throw new SessionError(
      body?.error?.message ?? "Autentikasi gagal.",
      body?.error?.code ?? String(response.status),
    );
  }
  if (!body?.data?.token) {
    throw new SessionError("Respons masuk tidak valid.", "SESSION_ERROR");
  }
  return body.data;
}

/** Masuk; melempar SessionError berisi pesan dari server bila gagal. */
export async function signIn(email: string, password: string): Promise<void> {
  const payload = await authRequest("login", { body: { email, password } });
  setToken(payload.token);
}

/**
 * Daftar; mengembalikan pesan sukses kosong bila langsung masuk, atau pesan
 * yang harus ditampilkan sebagai notice. Email terdaftar diterjemahkan jadi
 * ajakan masuk — registrasi memang tidak bisa menyembunyikan hal itu, dan
 * pengguna lebih berguna diberi tahu untuk beralih ke form masuk.
 *
 * `displayName` dikirim utuh ("Raditya Pratama"); nama depan adalah token
 * pertama yang dipakai aplikasi menyapa. Bila kosong, backend mengisinya dari
 * bagian sebelum "@" pada email.
 */
export async function signUp(
  email: string,
  password: string,
  displayName = "",
): Promise<string> {
  try {
    const payload = await authRequest("register", {
      body: { email, password, displayName },
    });
    setToken(payload.token);
    return "";
  } catch (err) {
    if (err instanceof SessionError && err.code === "CONFLICT") {
      return "Email sudah terdaftar. Silakan masuk.";
    }
    throw err;
  }
}

/**
 * Keluar: kredensial lokal dibersihkan lebih dulu, lalu sesi dicabut di
 * server. Bila panggilan server gagal (jaringan mati), token di server tetap
 * sah sampai kedaluwarsa — itu keterbatasan jujur dari logout satu-langkah;
 * pengguna tetap tercatat keluar di perangkat ini.
 */
export async function signOut(): Promise<void> {
  const token = getToken();
  clearSession();
  if (!token) return;
  try {
    await authRequest("logout", { token });
  } catch {
    // Sengaja ditelan: keluar dari perangkat ini sudah pasti terjadi.
  }
}

// ---------------------------------------------------------------------------
// Supabase Auth — DORMAN, tidak dipakai selama AUTH_MODE=session
// ---------------------------------------------------------------------------
//
// Dipertahankan sebagai jalur cadangan bila FISCA kelak menjadi proyek SaaS
// (ISSUE-15 konteks #3). Semuanya diekspor agar tetap lolos pemeriksaan
// unused-export; tidak ada import yang memakainya saat ini.

const SUPABASE_URL = (process.env.NEXT_PUBLIC_SUPABASE_URL ?? "").replace(/\/$/, "");
const SUPABASE_ANON_KEY = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY ?? "";

/** Shape of a GoTrue token response (or its nested `session` form). */
export type DormantAuthPayload = {
  access_token?: string;
  refresh_token?: string;
  expires_at?: number | string;
  session?: DormantAuthPayload;
};

export function isSupabaseConfigured(): boolean {
  return Boolean(SUPABASE_URL && SUPABASE_ANON_KEY);
}

export async function dormantSupabaseSignIn(email: string, password: string): Promise<void> {
  await dormantGotTrue("token?grant_type=password", { email, password });
}

export async function dormantSupabaseSignUp(email: string, password: string): Promise<string> {
  const body = await dormantGotTrue("signup", { email, password });
  return body.access_token ? "" : "Registrasi diterima. Cek email untuk konfirmasi, lalu masuk.";
}

async function dormantGotTrue(path: string, payload: unknown): Promise<DormantAuthPayload> {
  let response: Response;
  try {
    response = await fetch(`${SUPABASE_URL}/auth/v1/${path}`, {
      method: "POST",
      headers: { apikey: SUPABASE_ANON_KEY, "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  } catch {
    throw new SessionError("Tidak bisa terhubung ke server autentikasi.", "NETWORK_ERROR");
  }
  const text = await response.text();
  const body = text ? (JSON.parse(text) as DormantAuthPayload & { message?: string }) : null;
  if (!response.ok) {
    throw new SessionError(body?.message ?? "Autentikasi gagal.", String(response.status));
  }
  return body ?? {};
}
