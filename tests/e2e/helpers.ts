import { execSync } from "node:child_process";
import { randomUUID } from "node:crypto";
import type { Page } from "@playwright/test";

/** Base URLs of the running Docker stack. */
export const API_BASE =
  process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
export const DB_CONTAINER = "fisca-db";

/** Kata sandi untuk semua akun E2E; domain .test tidak pernah dikirim email. */
const E2E_PASSWORD = "kata-sandi-e2e-123";

/**
 * Registers a brand-new account through the real auth flow and returns its
 * session token. Every test provisions its own user so it starts with a full
 * rate-limit bucket: the limiter is keyed by user id, and sharing one user
 * across the suite made later tests trip 429 (they are not what E2E verifies —
 * rate limiting has its own unit tests).
 *
 * Register juga membuat akun & kategori bawaan, jadi tidak perlu dipanggil
 * terpisah. Inilah jalur AUTH_MODE=session yang sebenarnya — tidak ada lagi
 * format token khusus test yang dipercaya backend (ISSUE-15).
 */
export async function freshToken(): Promise<string> {
  const email = `e2e-${randomUUID()}@fisca.test`;
  const response = await fetch(`${API_BASE}/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password: E2E_PASSWORD }),
  });
  const text = await response.text();
  if (response.status !== 201) {
    throw new Error(`register ${email} failed: ${response.status} ${text}`);
  }
  const body = JSON.parse(text);
  return body.data.token as string;
}

/**
 * Removes accounts created by E2E runs — only theirs, by the `e2e-` prefix
 * they register with.
 *
 * Polanya harus sempit. Membersihkan seluruh `@fisca.test` akan ikut menyentuh
 * akun smoke (`smoke-a`, `smoke-b`), dan penghapusan itu gagal: kategori yang
 * dialokasikan ke budget dirujuk `budget_categories.category_id` dengan
 * `ON DELETE RESTRICT`. Akun smoke justru perlu disisakan — smoke idempoten,
 * jadi ia memakai akun yang sama tiap run.
 *
 * Everything a user owns — sessions included — cascades from the users row.
 */
export function resetTestUsers(): void {
  dbQuery(`DELETE FROM users WHERE email LIKE 'e2e-%@fisca.test'`);
}

/** Runs a SQL statement through psql inside the database container. */
export function dbQuery(sql: string): string {
  const command = `docker exec ${DB_CONTAINER} psql -U fisca -d fisca -t -A -c "${sql}"`;
  return execSync(command, { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim();
}

type ApiResult<T = any> = { status: number; body: T };

/** Calls the API directly, for setup and for asserting server-side state. */
export async function api<T = any>(
  token: string,
  method: string,
  path: string,
  body?: unknown,
  idempotencyKey?: string,
): Promise<ApiResult<T>> {
  const headers: Record<string, string> = { Authorization: `Bearer ${token}` };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (idempotencyKey) headers["Idempotency-Key"] = idempotencyKey;

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  return { status: response.status, body: text ? JSON.parse(text) : null };
}

/**
 * Sanity check: the freshly registered token is accepted by the API. Provisioning
 * itself happens during register, so this only fails fast when auth breaks.
 */
export async function provision(token: string): Promise<void> {
  const result = await api(token, "GET", "/me");
  if (result.status !== 200) {
    throw new Error(`provision ${token} failed: ${result.status} ${JSON.stringify(result.body)}`);
  }
}

/**
 * Clean starting point for one test: wipes previous E2E users and provisions a
 * brand-new one. Playwright re-runs file-level beforeAll hooks after a failure,
 * so each test owns its setup instead of inheriting it.
 */
export async function startClean(): Promise<string> {
  resetTestUsers();
  const token = await freshToken();
  await provision(token);
  return token;
}

type Account = { id: string; isDefault: boolean; balance: number };

/** Returns the id of the user's default (first) account. */
export async function defaultAccountId(token: string): Promise<string> {
  const result = await api(token, "GET", "/accounts");
  const accounts = result.body.data as Account[];
  const account = accounts.find((a) => a.isDefault) ?? accounts[0];
  if (!account) throw new Error("no account provisioned");
  return account.id;
}

/** Records one expense directly through the API (setup, not the flow under test). */
export async function seedExpense(
  token: string,
  description: string,
  amount: number,
): Promise<string> {
  const accountId = await defaultAccountId(token);
  const result = await api(token, "POST", "/transactions", {
    type: "expense",
    amount,
    description,
    accountId,
    source: "manual",
  }, `e2e-${description}-${amount}-${Date.now()}`);
  if (result.status !== 201) {
    throw new Error(`seed expense failed: ${result.status} ${JSON.stringify(result.body)}`);
  }
  return result.body.data.transaction.id as string;
}

/** Creates the budget when missing and forces it to the wanted limit. */
export async function ensureBudget(token: string, limit: number): Promise<string> {
  const list = await api(token, "GET", "/budgets");
  const existing = (list.body.data as Array<{ id: string; overallLimit: number; version: number }>)[0];
  if (existing) {
    if (existing.overallLimit !== limit) {
      const patched = await api(token, "PATCH", `/budgets/${existing.id}`, {
        version: existing.version,
        overallLimit: limit,
      });
      if (patched.status !== 200) {
        throw new Error(`budget patch failed: ${patched.status} ${JSON.stringify(patched.body)}`);
      }
    }
    return existing.id;
  }
  const created = await api(token, "POST", "/budgets", {
    period: "this_month",
    overallLimit: limit,
  });
  if (created.status !== 201) {
    throw new Error(`budget create failed: ${created.status} ${JSON.stringify(created.body)}`);
  }
  return created.body.data.id;
}

/**
 * Opens a page already signed in as the given user. The token is written to
 * localStorage before any page script runs, which is where lib/session.ts reads
 * it from.
 */
export async function openAs(page: Page, token: string, path: string): Promise<void> {
  await page.addInitScript((value: string) => {
    window.localStorage.setItem("fisca.token", value);
  }, token);
  await page.goto(path);
}
