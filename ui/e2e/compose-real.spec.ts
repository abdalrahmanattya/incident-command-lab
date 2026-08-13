import { test, expect } from '@playwright/test'

test('real Compose operator recovery and incident flow', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByText('Service health')).toBeVisible()
  const dependency = page.locator('label.fault').filter({ hasText: 'dependency' }).locator('input')
  await dependency.locator('xpath=..').click()
  await expect(dependency).toBeChecked()
  await page.getByRole('button', { name: 'Create reservation' }).click()
  await expect(page.getByText('COMPENSATED')).toBeVisible()

  await dependency.locator('xpath=..').click()
  await expect(dependency).not.toBeChecked()
  const key = page.locator('label').filter({ hasText: 'Idempotency key' }).locator('input')
  await key.fill('browser-recovery-1')
  await page.getByRole('button', { name: 'Create reservation' }).click()
  await expect(page.getByText('CONFIRMED')).toBeVisible()
  const reservationId = await page.locator('.result strong').innerText()
  await page.getByRole('button', { name: 'Create reservation' }).click()
  await expect(page.locator('.result strong')).toHaveText(reservationId)

  await page.getByLabel('Incident title').fill('Browser recovery drill')
  await page.getByRole('button', { name: 'Open incident' }).click()
  await expect(page.getByRole('heading', { name: 'Browser recovery drill' })).toBeVisible()
  await page.getByRole('button', { name: 'Run advisory analysis' }).click()
  await expect(page.locator('.analysis')).toBeVisible()
})
