import { API_BASE, resetTestUsers } from "./helpers";

/**
 * Wipes the E2E users before the run so every project (desktop, mobile) starts
 * from a clean database, and fails fast when the stack is not running.
 */
export default async function globalSetup(): Promise<void> {
  try {
    const health = await fetch("http://localhost:8080/health");
    if (!health.ok) throw new Error(`health status ${health.status}`);
  } catch (error) {
    throw new Error(
      `Stack Docker tidak berjalan di ${API_BASE}. Jalankan \`npm run stack:up\` dulu. (${String(error)})`,
    );
  }
  resetTestUsers();
}
