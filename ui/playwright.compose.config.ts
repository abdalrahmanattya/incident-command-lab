import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/compose-real.spec.ts',
  use: { baseURL: 'http://127.0.0.1:8080', trace: 'retain-on-failure' },
  webServer: {
    command: 'docker compose up -d --build && until curl -fsS http://127.0.0.1:8080/healthz >/dev/null; do sleep 2; done && tail -f /dev/null',
    url: 'http://127.0.0.1:8080/healthz',
    reuseExistingServer: true,
    timeout: 180_000,
  },
})
