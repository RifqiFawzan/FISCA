import { test, expect } from "@playwright/test";
import { api, openAs, startClean } from "./helpers";

// Plan.md §12.5 flow 2 (voice expense → STT → parse → stored) and PRD §23
// exit 18 (voice permission denial falls back to chat).
//
// The real transcription provider needs OPENAI_API_KEY (ISSUE-16, BLOCKED), so
// the success path mocks the STT endpoint and the denial path runs without the
// microphone permission. What is verified here is the UI wiring end to end.
test("voice permission denial falls back to chat", async ({ page }) => {
  const token = await startClean();

  await openAs(page, token, "/assistant");
  await expect(page.getByText("Halo, saya Asisten FISCA")).toBeVisible();

  // No microphone permission in this context → getUserMedia rejects.
  await page.getByRole("button", { name: "Rekam suara" }).click();
  await expect(page.getByText("Mikrofon tidak tersedia atau izin ditolak")).toBeVisible();

  // Chat keeps working (PRD exit 18).
  await page.getByLabel("Tulis pesan").fill("catat uji fallback 12rb");
  await page.getByRole("button", { name: "Kirim pesan" }).click();
  await expect(page.getByText("sudah dicatat").first()).toBeVisible();

  const transactions = await api(token, "GET", "/transactions?limit=50");
  const rows = transactions.body.data.items as Array<{ description: string }>;
  expect(rows.some((r) => r.description.toLowerCase().includes("uji fallback"))).toBe(true);
});

test.describe("with microphone", () => {
  test.use({ permissions: ["microphone"] });

  test("voice expense is transcribed, parsed and stored", async ({ page }) => {
    const token = await startClean();

    await page.route("**/speech/transcriptions", (route) =>
      route.fulfill({
        json: { data: { transcript: "catat makan siang 35rb", durationMs: 1200 } },
      }),
    );

    await openAs(page, token, "/assistant");
    await expect(page.getByText("Halo, saya Asisten FISCA")).toBeVisible();

    // Record a short clip with the fake device, then stop.
    await page.getByRole("button", { name: "Rekam suara" }).click();
    await page.waitForTimeout(400);
    await page.getByRole("button", { name: "Hentikan rekaman" }).click();

    // The transcript lands in the composer for the user to review.
    await expect(page.getByLabel("Tulis pesan")).toHaveValue("catat makan siang 35rb");
    await page.getByRole("button", { name: "Kirim pesan" }).click();

    await expect(page.getByText("sudah dicatat").first()).toBeVisible();

    const transactions = await api(token, "GET", "/transactions?limit=50");
    const rows = transactions.body.data.items as Array<{ description: string; amount: number }>;
    const stored = rows.find((r) => r.description === "Makan siang");
    expect(stored, "voice expense must be stored as a real row").toBeTruthy();
    expect(stored!.amount).toBe(35000);
  });
});
