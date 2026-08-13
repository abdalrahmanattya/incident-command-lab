import { test, expect } from '@playwright/test'

const state = { products: [{ ID: 'concert', Name: 'Concert ticket', PriceCents: 2500, Stock: 100 }], reservations: [], outbox: [], incidents: [], faults: [] }
test.beforeEach(async ({ page }) => {
  await page.route('**/api/ops/state', route => route.fulfill({ json: state }))
  await page.route('**/api/ops/incidents', route => route.fulfill({ json: { incidents: state.incidents } }))
  await page.route('**/api/ops/faults', route => route.fulfill({ json: { fault: 'dependency', enabled: true, active: ['dependency'] } }))
  await page.route('**/api/v1/reservations', route => route.fulfill({ status: 201, json: { ID: 'res-demo', Status: 'COMPENSATED', Quantity: 1, TotalCents: 2500, Failure: 'payment dependency unavailable' } }))
  await page.goto('/')
})

test('renders dashboard health and controlled chaos controls', async ({ page }) => {
  await expect(page.getByText('Reliability operator console')).toBeVisible()
  await expect(page.getByText('Service health')).toBeVisible()
  await expect(page.getByText('dependency')).toBeVisible()
  await expect(page.getByText('Local auth disabled')).toBeVisible()
})

test('dependency fault creates a compensated reservation', async ({ page }) => {
  await page.getByText('Create reservation').click()
  await expect(page.getByText('COMPENSATED')).toBeVisible()
  await expect(page.getByText('payment dependency unavailable')).toBeVisible()
})

test('incident list supports evidence and advisory analysis', async ({ page }) => {
  await page.route('**/api/ops/incidents', route => route.fulfill({ json: { incidents: [{ ID: 'inc-1', Title: 'Queue pressure', Severity: 'SEV2', Status: 'OPEN', StartedAt: new Date().toISOString(), Signals: ['backlog fault enabled'], Timeline: ['incident opened'], Runbooks: ['runbooks/reservation-dependency.md'] }] } }))
  await page.route('**/api/ops/incidents/inc-1/evidence', route => route.fulfill({ json: { IncidentID: 'inc-1', Signals: ['backlog fault enabled'], Timeline: ['incident opened'], Runbooks: ['runbooks/reservation-dependency.md'] } }))
  await page.route('**/api/ops/incidents/inc-1/analyze', route => route.fulfill({ json: { Provider: 'deterministic', Summary: 'Advisory only', Hypotheses: [], Checks: ['inspect queue'] } }))
  await page.reload(); await page.getByText('Queue pressure').click(); await expect(page.getByText('Timeline')).toBeVisible(); await page.getByText('Run advisory analysis').click(); await expect(page.getByText('Advisory only')).toBeVisible()
})
