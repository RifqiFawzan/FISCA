import { defineConfig } from "@playwright/test";

/**
 * E2E tests for the nine mandatory flows in plan.md §12.5 (PRD §22.4).
 * They run against the running Docker stack (web :3000, api :8080) and use the
 * system Chrome instead of downloading a browser.
 *
 *   npm run stack:up && npm run test:e2e
 */
const WEB_URL = process.env.E2E_WEB_URL ?? "http://localhost:3000";

export default defineConfig({
  testDir: "./tests/e2e",
  timeout: 60_000,
  expect: { timeout: 10_000 },
  // Test users are shared, so the suite runs serially.
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  globalSetup: "./tests/e2e/global-setup.ts",
  globalTeardown: "./tests/e2e/global-teardown.ts",
  use: {
    baseURL: WEB_URL,
    channel: "chrome",
    launchOptions: {
      // Provides a fake microphone so the voice flow can record; permission is
      // still requested per context, so the denial flow keeps working.
      args: ["--use-fake-device-for-media-stream"],
    },
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    { name: "desktop", use: { viewport: { width: 1280, height: 720 } } },
    { name: "mobile", use: { viewport: { width: 360, height: 800 } } },
  ],
});
