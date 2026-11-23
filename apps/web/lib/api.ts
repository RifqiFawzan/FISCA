/**
 * Typed client for the FISCA API (plan.md §7).
 *
 * The browser talks to the API directly; the session token issued by
 * POST /auth/login lives in localStorage. Every mutating request carries an
 * Idempotency-Key supplied by the caller.
 *
 * A 401 means the session was revoked or expired; errorMessage() turns it into
 * "Sesi Anda sudah berakhir" (AC ISSUE-15 #5).
 */

// Token storage lives in lib/session.ts (auth lokal ISSUE-15). The re-exports
// keep the original import paths working across views.
import { getToken } from "@/lib/session";
import type { components } from "./api.gen";

export { getToken, setToken, hasToken, useHasToken } from "@/lib/session";

const BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL?.replace(/\/$/, "") ??
  "http://localhost:8080/api/v1";

export class ApiError extends Error {
  status: number;
  code: string;
  details?: unknown;
  requestId?: string;

  constructor(
    message: string,
    status: number,
    code = "INTERNAL",
    details?: unknown,
    requestId?: string,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
    this.requestId = requestId;
  }
}

export function newIdempotencyKey(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `idem-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

type Envelope<T> = {
  data: T;
  meta?: { requestId?: string; nextCursor?: string | null };
};

type RequestOptions = {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  idempotencyKey?: string;
  signal?: AbortSignal;
};

async function envelope<T>(path: string, options: RequestOptions = {}): Promise<Envelope<T>> {
  const token = getToken();
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (options.body !== undefined) headers["Content-Type"] = "application/json";
  if (options.idempotencyKey) headers["Idempotency-Key"] = options.idempotencyKey;

  let response: Response;
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      cache: "no-store",
      signal: options.signal,
    });
  } catch {
    throw new ApiError(
      "Tidak bisa terhubung ke server. Periksa koneksi lalu coba lagi.",
      0,
      "NETWORK_ERROR",
    );
  }

  const text = await response.text();
  let payload: unknown = null;
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = null;
    }
  }

  if (!response.ok) {
    const failure = payload as
      | { error?: { code?: string; message?: string; details?: unknown; requestId?: string } }
      | null;
    const error = failure?.error;
    throw new ApiError(
      error?.message ?? "Terjadi kesalahan pada server.",
      response.status,
      error?.code ?? "INTERNAL",
      error?.details,
      error?.requestId,
    );
  }

  const result = (payload ?? {}) as Envelope<T>;
  return { data: result.data, meta: result.meta };
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const result = await envelope<T>(path, options);
  return result.data;
}

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    // An expired or revoked token should send the user to sign in again, not
    // show a raw provider message.
    if (error.code === "UNAUTHORIZED" || error.status === 401) {
      return "Sesi Anda sudah berakhir. Silakan masuk kembali.";
    }
    return error.message;
  }
  if (error instanceof Error) return error.message;
  return "Terjadi kesalahan yang tidak terduga.";
}

function query(params: Record<string, string | number | boolean | null | undefined>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") continue;
    search.set(key, String(value));
  }
  const raw = search.toString();
  return raw ? `?${raw}` : "";
}

// ---------------------------------------------------------------------------
// DTOs — aliased from ./api.gen.ts, generated from docs/api/openapi.yaml by
// `npm run gen:api`. The wire shapes therefore cannot drift from the Go
// contract; only the normalisations below are written by hand.
// ---------------------------------------------------------------------------

type Schema = components["schemas"];

export type User = Schema["User"];
export type Account = Schema["Account"];
export type Category = Schema["Category"];
export type Transaction = Schema["Transaction"];
export type Warning = Schema["Warning"];
export type AuditEntry = Schema["AuditEntry"];
export type TransactionDetail = Schema["TransactionDetail"];
export type Summary = Schema["Summary"];
export type CategorySpend = Schema["CategorySpend"];
export type BudgetCategoryStatus = Schema["BudgetCategoryStatus"];
export type Budget = Schema["Budget"];
export type BudgetProposal = Schema["BudgetProposal"];
export type Period = Schema["Period"];
export type MonthlyTotal = Schema["MonthlyTotal"];
export type Dashboard = Schema["Dashboard"];
export type FinancialSummary = Schema["FinancialSummary"];
export type LargestTransactions = Schema["LargestTransactions"];
export type UnusualFinding = Schema["UnusualFinding"];
export type UnusualSpending = Schema["UnusualSpending"];
export type CategoryDelta = Schema["CategoryDelta"];
export type SpendingComparison = Schema["SpendingComparison"];
export type CashFlowPoint = Schema["CashFlowPoint"];
export type CashFlow = Schema["CashFlow"];
export type SavingsProjection = Schema["SavingsProjection"];
export type SavingsGoal = Schema["SavingsGoal"];
export type PurchaseAdvisor = Schema["PurchaseAdvisor"];
export type SearchFilters = Schema["SearchFilters"];
export type AssistantCard = Schema["AssistantCard"];
export type AssistantReply = Schema["AssistantReply"];
export type Conversation = Schema["Conversation"];
export type AssistantMessage = Schema["AssistantMessage"];

/** Query parameters for GET /transactions (client side, not a wire type). */
export type TransactionFilters = {
  type?: string;
  categoryId?: string;
  accountId?: string;
  q?: string;
  period?: string;
  from?: string;
  to?: string;
  cursor?: string;
  limit?: number;
};

/**
 * A page of transactions. The server returns the next cursor in
 * `meta.nextCursor`; the client normalises an absent cursor to `null` so
 * callers never have to handle `undefined`.
 */
export type TransactionPage = {
  items: Transaction[];
  nextCursor: string | null;
};

/** Search response, normalised the same way as {@link TransactionPage}. */
export type SearchResult = {
  filters: SearchFilters;
  items: Transaction[];
  nextCursor: string | null;
};

/** POST /transactions body. */
export type CreateTransactionPayload = Schema["CreateTransactionInput"];
export type CreateTransactionResult = Schema["CreateTransactionResult"];

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

export const api = {
  me: {
    get: () => request<User>("/me"),
    update: (body: {
      displayName?: string;
      timezone?: string;
      locale?: string;
      completeOnboarding?: boolean;
    }) => request<User>("/me", { method: "PATCH", body }),
  },

  accounts: {
    list: () => request<Account[]>("/accounts"),
    create: (body: { name: string; type: string; openingBalance?: number; isDefault?: boolean }) =>
      request<Account>("/accounts", { method: "POST", body }),
    update: (id: string, body: { name?: string; type?: string }) =>
      request<Account>(`/accounts/${id}`, { method: "PATCH", body }),
    setDefault: (id: string) =>
      request<Account>(`/accounts/${id}/default`, { method: "POST" }),
    archive: (id: string) =>
      request<{ archived: boolean }>(`/accounts/${id}`, { method: "DELETE" }),
  },

  categories: {
    list: (kind?: "income" | "expense") =>
      request<Category[]>(`/categories${query({ kind })}`),
    create: (body: { name: string; kind: "income" | "expense"; icon?: string; color?: string }) =>
      request<Category>("/categories", { method: "POST", body }),
    update: (id: string, body: { name?: string; icon?: string; color?: string }) =>
      request<Category>(`/categories/${id}`, { method: "PATCH", body }),
    archive: (id: string) =>
      request<{ archived: boolean }>(`/categories/${id}`, { method: "DELETE" }),
  },

  transactions: {
    create: (body: CreateTransactionPayload, idempotencyKey = newIdempotencyKey()) =>
      request<CreateTransactionResult>("/transactions", {
        method: "POST",
        body,
        idempotencyKey,
      }),
    list: async (filters: TransactionFilters = {}) => {
      const result = await envelope<{ items: Transaction[] }>(
        `/transactions${query({ ...filters })}`,
      );
      return {
        items: result.data.items ?? [],
        nextCursor: result.meta?.nextCursor ?? null,
      } satisfies TransactionPage;
    },
    search: async (q: string, limit = 20) => {
      const result = await envelope<{
        filters: SearchFilters;
        items: Transaction[];
      }>(`/transactions/search${query({ q, limit })}`);
      return {
        filters: result.data.filters,
        items: result.data.items ?? [],
        nextCursor: result.meta?.nextCursor ?? null,
      } satisfies SearchResult;
    },
    get: (id: string) => request<TransactionDetail>(`/transactions/${id}`),
    update: (
      id: string,
      body: {
        version: number;
        description?: string;
        amount?: number;
        categoryId?: string | null;
        occurredAt?: string;
        notes?: string | null;
      },
    ) => request<Transaction>(`/transactions/${id}`, { method: "PATCH", body }),
    void: (id: string, reason: string, idempotencyKey = newIdempotencyKey()) =>
      request<Transaction>(`/transactions/${id}/void`, {
        method: "POST",
        body: { reason },
        idempotencyKey,
      }),
  },

  analytics: {
    dashboard: (period?: string) =>
      request<Dashboard>(`/analytics/dashboard${query({ period })}`),
    summary: (period?: string) =>
      request<FinancialSummary>(`/analytics/financial-summary${query({ period })}`),
    largest: (params: { period?: string; type?: string; categoryId?: string; limit?: number } = {}) =>
      request<LargestTransactions>(`/analytics/largest-transactions${query(params)}`),
    unusual: (params: { period?: string; categoryId?: string } = {}) =>
      request<UnusualSpending>(`/analytics/unusual-spending${query(params)}`),
    comparison: (period?: string) =>
      request<SpendingComparison>(`/analytics/spending-comparison${query({ period })}`),
    cashFlow: (params: { period?: string; bucket?: string } = {}) =>
      request<CashFlow>(`/analytics/cash-flow${query(params)}`),
  },

  budgets: {
    list: () => request<Budget[]>("/budgets"),
    create: (body: {
      period?: string;
      overallLimit: number;
      allocations?: { categoryId: string; limitAmount: number }[];
    }) => request<Budget>("/budgets", { method: "POST", body }),
    get: (id: string) => request<Budget>(`/budgets/${id}`),
    update: (
      id: string,
      body: {
        version: number;
        overallLimit?: number;
        allocations?: { categoryId: string; limitAmount: number }[];
      },
    ) => request<Budget>(`/budgets/${id}`, { method: "PATCH", body }),
    status: (id: string) => request<Budget>(`/budgets/${id}/status`),
  },

  proposals: {
    list: (status?: string) =>
      request<BudgetProposal[]>(`/budget-proposals${query({ status })}`),
    create: (body: {
      budgetId: string;
      changes: {
        overallLimit?: number;
        allocations?: { categoryId: string; limitAmount: number }[];
      };
      rationale?: string;
    }) => request<BudgetProposal>("/budget-proposals", { method: "POST", body }),
    approve: (id: string, baseVersion?: number, idempotencyKey = newIdempotencyKey()) =>
      request<Budget>(`/budget-proposals/${id}/approve`, {
        method: "POST",
        body: baseVersion ? { baseVersion } : {},
        idempotencyKey,
      }),
    reject: (id: string) =>
      request<BudgetProposal>(`/budget-proposals/${id}/reject`, { method: "POST" }),
  },

  savings: {
    list: () => request<SavingsGoal[]>("/savings-goals"),
    create: (body: {
      name: string;
      targetAmount: number;
      deadline: string;
      accountId: string;
    }) => request<SavingsGoal>("/savings-goals", { method: "POST", body }),
    get: (id: string, monthly?: number) =>
      request<SavingsGoal>(`/savings-goals/${id}${query({ monthly })}`),
    update: (
      id: string,
      body: {
        version: number;
        name?: string;
        targetAmount?: number;
        deadline?: string;
        accountId?: string;
      },
    ) => request<SavingsGoal>(`/savings-goals/${id}`, { method: "PATCH", body }),
    archive: (id: string) =>
      request<{ archived: boolean }>(`/savings-goals/${id}`, { method: "DELETE" }),
    advisor: (price: number) =>
      request<PurchaseAdvisor>(`/purchase-advisor${query({ price })}`),
  },

  assistant: {
    send: (body: { text: string; conversationId?: string; source?: "text" | "voice" }) =>
      request<AssistantReply>("/assistant/messages", { method: "POST", body }),
    conversations: (limit = 20) =>
      request<Conversation[]>(`/assistant/conversations${query({ limit })}`),
    conversation: (id: string) =>
      request<AssistantMessage[]>(`/assistant/conversations/${id}`),
    deleteConversation: (id: string) =>
      request<{ deleted: boolean }>(`/assistant/conversations/${id}`, { method: "DELETE" }),
  },

  speech: {
    transcribe: async (audio: Blob, filename = "rekaman.webm") => {
      const token = getToken();
      const form = new FormData();
      form.append("audio", audio, filename);
      let response: Response;
      try {
        response = await fetch(`${BASE_URL}/speech/transcriptions`, {
          method: "POST",
          headers: token ? { Authorization: `Bearer ${token}` } : undefined,
          body: form,
        });
      } catch {
        throw new ApiError("Tidak bisa terhubung ke server.", 0, "NETWORK_ERROR");
      }
      const text = await response.text();
      const payload = text ? JSON.parse(text) : null;
      if (!response.ok) {
        throw new ApiError(
          payload?.error?.message ?? "Transkripsi gagal.",
          response.status,
          payload?.error?.code ?? "INTERNAL",
          payload?.error?.details,
          payload?.error?.requestId,
        );
      }
      return payload?.data as { transcript: string; durationMs: number };
    },
  },
};
