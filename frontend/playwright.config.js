import { defineConfig, devices } from "@playwright/test";

const externalBaseURL = process.env.E2E_BASE_URL;
const liveMode = process.env.E2E_LIVE === "1";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: !liveMode,
  workers: liveMode ? 1 : undefined,
  reporter: "html",
  use: {
    baseURL: externalBaseURL ?? "http://127.0.0.1:4173",
    trace: "on-first-retry",
  },
  webServer: externalBaseURL
    ? undefined
    : {
        command: "pnpm preview --host 127.0.0.1 --port 4173",
        port: 4173,
        reuseExistingServer: true,
      },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "mobile", use: { ...devices["Pixel 7"] } },
  ],
});

