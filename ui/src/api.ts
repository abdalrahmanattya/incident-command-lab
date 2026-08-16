export type Product = { ID: string; Name: string; PriceCents: number; Stock: number }
export type Reservation = { ID: string; IdempotencyKey: string; ProductID: string; CustomerID: string; Status: string; Quantity: number; TotalCents: number; Release: string; Failure?: string; CreatedAt: string; UpdatedAt: string }
export type Event = { ID: string; Type: string; AggregateID: string; Status: string; Attempts: number; LastError?: string; CreatedAt: string }
export type Incident = { ID: string; Title: string; Status: string; Severity: string; StartedAt: string; EndedAt?: string; Signals: string[]; Timeline: string[]; Runbooks: string[] }
export type State = { products: Product[]; reservations: Reservation[]; outbox: Event[]; incidents: Incident[]; faults: string[]; Products?: Product[]; Reservations?: Reservation[]; Outbox?: Event[]; Incidents?: Incident[]; Faults?: string[] }
export type Evidence = { incident_id: string; timeline: string[]; signals: string[]; runbooks: string[] }
export type Advisory = { provider: string; summary: string; hypotheses: { title: string; confidence: number; evidence: string[] }[]; checks: string[] }

const root = '/api'
let accessToken = ''
export function setAccessToken(token: string) { accessToken = token }
export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  headers.set('accept', 'application/json')
  if (init?.body) headers.set('content-type', 'application/json')
  if (accessToken) headers.set('authorization', `Bearer ${accessToken}`)
  const response = await fetch(`${root}${path}`, { ...init, headers })
  if (!response.ok) throw new Error(await response.text() || `${response.status} ${response.statusText}`)
  return response.status === 204 ? (undefined as T) : response.json() as Promise<T>
}
