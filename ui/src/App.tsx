import { useCallback, useEffect, useMemo, useState } from 'react'
import { config, OperatorAuth } from './auth'
import { request, type Advisory, type Evidence, type Incident, type Reservation, type State, setAccessToken } from './api'

const faults = ['latency', 'dependency', 'backlog', 'duplicate', 'database', 'bad-release']
const auth = new OperatorAuth()
const money = (cents: number) => `€${(cents / 100).toFixed(2)}`
const date = (value?: string) => value ? new Date(value).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' }) : '—'

export function App() {
  const [ready, setReady] = useState(false)
  const [error, setError] = useState('')
  useEffect(() => { auth.initialise().then(() => auth.token()).catch((e: Error) => setError(e.message)).finally(() => setReady(true)) }, [])
  if (!ready) return <div className="center-screen"><span className="spinner" />Loading operator identity…</div>
  if (auth.enabled && !auth.account) return <div className="center-screen auth-card"><p className="eyebrow">INCIDENT COMMAND LAB</p><h1>Operator access</h1><p>Sign in with the configured Microsoft Entra tenant to open the reliability console.</p><button className="primary" onClick={() => auth.signIn()}>Sign in with Entra ID</button>{error && <p className="error">{error}</p>}</div>
  if (auth.enabled && !auth.allowed) return <div className="center-screen auth-card"><p className="eyebrow">ACCESS DENIED</p><h1>Operator group required</h1><p>Your account is signed in, but it is not a member of the configured operator group.</p></div>
  return <Console authEnabled={auth.enabled} />
}

function Console({ authEnabled }: { authEnabled: boolean }) {
  const [state, setState] = useState<State | null>(null)
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [selected, setSelected] = useState<Incident | null>(null)
  const [evidence, setEvidence] = useState<Evidence | null>(null)
  const [analysis, setAnalysis] = useState<Advisory | null>(null)
  const [reservation, setReservation] = useState<Reservation | null>(null)
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)

  const refresh = useCallback(async () => {
    try {
      const [next, listed] = await Promise.all([request<State>('/ops/state'), request<{ incidents: Incident[] }>('/ops/incidents')])
      setState({ ...next, products: next.Products ?? next.products ?? [], reservations: next.Reservations ?? next.reservations ?? [], outbox: next.Outbox ?? next.outbox ?? [], incidents: next.Incidents ?? listed.incidents ?? [], faults: next.Faults ?? next.faults ?? [] } as State)
      setIncidents(listed.incidents ?? [])
    } catch (e) { setNotice(e instanceof Error ? e.message : 'Unable to refresh operator state') }
  }, [])
  useEffect(() => { void refresh(); const timer = window.setInterval(() => void refresh(), 8000); return () => window.clearInterval(timer) }, [refresh])

  const toggleFault = async (fault: string) => {
    setBusy(true); setNotice('')
    try { await request('/ops/faults', { method: 'POST', body: JSON.stringify({ fault, enabled: !state?.faults.includes(fault) }) }); await refresh(); setNotice(`${fault} fault updated`) }
    catch (e) { setNotice(e instanceof Error ? e.message : 'Fault update failed') } finally { setBusy(false) }
  }
  const openIncident = async (incident: Incident) => {
    setSelected(incident); setAnalysis(null)
    try { setEvidence(await request<Evidence>(`/ops/incidents/${incident.ID}/evidence`)) } catch (e) { setNotice(String(e)) }
  }
  const createIncident = async (title: string, severity: string) => {
    if (!title.trim()) return
    setBusy(true)
    try { const incident = await request<Incident>('/ops/incidents', { method: 'POST', body: JSON.stringify({ title, severity }) }); await refresh(); await openIncident(incident); setNotice('Incident created') }
    catch (e) { setNotice(e instanceof Error ? e.message : 'Incident creation failed') } finally { setBusy(false) }
  }
  const analyze = async () => {
    if (!selected) return
    setBusy(true)
    try { setAnalysis(await request<Advisory>(`/ops/incidents/${selected.ID}/analyze`, { method: 'POST' })) } catch (e) { setNotice(e instanceof Error ? e.message : 'Analysis failed') } finally { setBusy(false) }
  }
  const cancel = async () => {
    if (!reservation) return
    setBusy(true)
    try { setReservation(await request<Reservation>(`/v1/reservations/${reservation.ID}/cancel`, { method: 'POST' })); await refresh(); setNotice('Reservation cancelled and stock released') } catch (e) { setNotice(e instanceof Error ? e.message : 'Recovery failed') } finally { setBusy(false) }
  }

  const activeFaultCount = state?.faults.length ?? 0
  const queue = useMemo(() => state?.outbox ?? [], [state])
  return <div className="shell">
    <header className="topbar"><div className="brand"><span className="brand-mark">⌁</span><div><p className="eyebrow">INCIDENT COMMAND LAB</p><h1>Reliability operator console</h1></div></div><div className="top-actions"><span className="mode-pill"><span className="dot" />{authEnabled ? 'Entra protected' : 'Local auth disabled'}</span><button className="quiet" onClick={() => void refresh()}>Refresh</button></div></header>
    <main className="content">
      <section className="hero"><div><p className="eyebrow">LIVE CONTROL PLANE</p><h2>Keep the reservation path honest.</h2><p>Observe the queue, inject a bounded fault, and leave a traceable recovery trail.</p></div><div className="hero-stat"><strong>{activeFaultCount}</strong><span>active faults</span></div></section>
      {notice && <div className="notice" role="status">{notice}<button aria-label="Dismiss" onClick={() => setNotice('')}>×</button></div>}
      <div className="grid two">
        <section className="panel"><PanelTitle title="Service health" kicker="01 / SIGNALS" action={<span className="healthy"><span className="dot" />nominal</span>} /><div className="health-grid"><Health label="Gateway" value="ready" /><Health label="Reservation path" value={`${state?.reservations.length ?? 0} reservations`} /><Health label="Outbox" value={`${queue.filter(e => e.Status !== 'DELIVERED').length} pending`} /><Health label="Telemetry" value="OTel connected" /></div></section>
        <section className="panel"><PanelTitle title="Queue & recovery" kicker="02 / DURABILITY" /><div className="metric-row"><Metric label="Pending" value={queue.filter(e => e.Status !== 'DELIVERED').length} /><Metric label="Delivered" value={queue.filter(e => e.Status === 'DELIVERED').length} /><Metric label="Attempts" value={queue.reduce((sum, e) => sum + (e.Attempts || 0), 0)} /></div><div className="queue-list">{queue.slice(0, 5).map(e => <div className="queue-item" key={e.ID}><span className={`status-dot ${e.Status.toLowerCase()}`} /><span>{e.Type}</span><small>{e.Status} · {e.Attempts} attempts</small></div>)}{queue.length === 0 && <Empty text="No outbox events yet" />}</div></section>
      </div>
      <div className="grid two">
        <section className="panel"><PanelTitle title="Reservation path" kicker="03 / CUSTOMER PATH" /><ReservationForm state={state} reservation={reservation} setReservation={setReservation} setNotice={setNotice} /></section>
        <section className="panel"><PanelTitle title="Reversible fault toggles" kicker="04 / CONTROLLED CHAOS" action={<span className="muted">bounded · local only</span>} /><div className="fault-list">{faults.map(fault => <label className="fault" key={fault}><span><strong>{fault}</strong><small>{faultDescription(fault)}</small></span><input type="checkbox" checked={state?.faults.includes(fault) ?? false} disabled={busy} onChange={() => void toggleFault(fault)} /><span className="switch" /></label>)}</div></section>
      </div>
      <section className="panel incidents"><PanelTitle title="Incidents" kicker="05 / COMMAND LOG" action={<IncidentForm onCreate={createIncident} busy={busy} />} /><div className="incident-layout"><div className="incident-list">{incidents.map(incident => <button className={`incident-row ${selected?.ID === incident.ID ? 'selected' : ''}`} key={incident.ID} onClick={() => void openIncident(incident)}><span className={`severity ${incident.Severity.toLowerCase()}`}>{incident.Severity}</span><span><strong>{incident.Title}</strong><small>{incident.Status} · {date(incident.StartedAt)}</small></span><span className="chevron">›</span></button>)}{incidents.length === 0 && <Empty text="No incidents. Create one after enabling a fault." />}</div><IncidentDetail incident={selected} evidence={evidence} analysis={analysis} analyze={analyze} busy={busy} /></div></section>
    </main>
    <footer><span>Deterministic advisory mode · analysis never remediates</span><span>Cloud apply is intentionally unexecuted</span></footer>
  </div>
}

function PanelTitle({ title, kicker, action }: { title: string; kicker: string; action?: React.ReactNode }) { return <div className="panel-title"><div><p className="eyebrow">{kicker}</p><h3>{title}</h3></div>{action}</div> }
function Health({ label, value }: { label: string; value: string }) { return <div className="health"><span className="dot" /><div><strong>{label}</strong><small>{value}</small></div></div> }
function Metric({ label, value }: { label: string; value: number }) { return <div className="metric"><strong>{value}</strong><small>{label}</small></div> }
function Empty({ text }: { text: string }) { return <p className="empty">{text}</p> }
function faultDescription(fault: string) { return ({ latency: 'add bounded response delay', dependency: 'force payment compensation', backlog: 'retry events before delivery', duplicate: 'surface duplicate delivery', database: 'reject persistence calls', 'bad-release': 'flag release regression' } as Record<string, string>)[fault] }

function ReservationForm({ state, reservation, setReservation, setNotice }: { state: State | null; reservation: Reservation | null; setReservation: (r: Reservation | null) => void; setNotice: (s: string) => void }) {
  const [customer, setCustomer] = useState('operator-demo'); const [product, setProduct] = useState('concert'); const [quantity, setQuantity] = useState(1); const [key, setKey] = useState(`demo-${Date.now()}`); const [lookup, setLookup] = useState('')
  const submit = async () => { try { setReservation(await request<Reservation>('/v1/reservations', { method: 'POST', headers: { 'idempotency-key': key }, body: JSON.stringify({ customer_id: customer, product_id: product, quantity }) })); setNotice('Reservation created; repeat the same key to prove idempotency') } catch (e) { setNotice(e instanceof Error ? e.message : 'Reservation failed') } }
  const find = async () => { try { setReservation(await request<Reservation>(`/v1/reservations/${lookup.trim()}`)) } catch (e) { setNotice(e instanceof Error ? e.message : 'Reservation not found') } }
  return <div className="form-stack"><div className="form-grid"><label>Customer<input value={customer} onChange={e => setCustomer(e.target.value)} /></label><label>Product<select value={product} onChange={e => setProduct(e.target.value)}>{(state?.products ?? []).map(p => <option key={p.ID} value={p.ID}>{p.Name} · {money(p.PriceCents)}</option>)}</select></label><label>Qty<input type="number" min="1" max="10" value={quantity} onChange={e => setQuantity(Number(e.target.value))} /></label><label>Idempotency key<input value={key} onChange={e => setKey(e.target.value)} /></label></div><button className="primary" onClick={() => void submit()}>Create reservation</button>{reservation && <div className="result"><div><span className={`status-chip ${reservation.Status.toLowerCase()}`}>{reservation.Status}</span><strong>{reservation.ID}</strong><small>{reservation.Failure || `${reservation.Quantity} ticket(s) · ${money(reservation.TotalCents)}`}</small></div>{reservation.Status === 'CONFIRMED' && <button className="quiet" onClick={() => void request<Reservation>(`/v1/reservations/${reservation.ID}/cancel`, { method: 'POST' }).then(setReservation).then(() => setNotice('Reservation cancelled; compensation complete'))}>Cancel / release</button>}</div>}<div className="lookup"><input placeholder="Reservation ID" value={lookup} onChange={e => setLookup(e.target.value)} /><button className="quiet" onClick={() => void find()} disabled={!lookup.trim()}>Lookup</button></div></div>
}

function IncidentForm({ onCreate, busy }: { onCreate: (title: string, severity: string) => Promise<void>; busy: boolean }) { const [title, setTitle] = useState(''); const [severity, setSeverity] = useState('SEV3'); return <form className="incident-form" onSubmit={e => { e.preventDefault(); void onCreate(title, severity); setTitle('') }}><select aria-label="Severity" value={severity} onChange={e => setSeverity(e.target.value)}><option>SEV1</option><option>SEV2</option><option>SEV3</option><option>SEV4</option></select><input aria-label="Incident title" placeholder="New incident title" value={title} onChange={e => setTitle(e.target.value)} /><button className="primary" disabled={busy || title.trim().length < 3}>Open incident</button></form> }

function IncidentDetail({ incident, evidence, analysis, analyze, busy }: { incident: Incident | null; evidence: Evidence | null; analysis: Advisory | null; analyze: () => Promise<void>; busy: boolean }) { if (!incident) return <div className="detail-placeholder"><span className="crosshair">◎</span><p>Select an incident to inspect its evidence bundle.</p></div>; return <div className="detail"><div className="detail-head"><div><span className={`severity ${incident.Severity.toLowerCase()}`}>{incident.Severity}</span><h3>{incident.Title}</h3><small>Opened {date(incident.StartedAt)} · {incident.Status}</small></div><button className="primary" onClick={() => void analyze()} disabled={busy}>Run advisory analysis</button></div><div className="evidence-grid"><EvidenceBlock title="Timeline" items={evidence?.timeline ?? incident.Timeline} numbered /><EvidenceBlock title="Signals" items={evidence?.signals ?? incident.Signals} /><EvidenceBlock title="Runbooks" items={evidence?.runbooks ?? incident.Runbooks} /></div>{analysis && <div className="analysis"><div><p className="eyebrow">ADVISORY · {analysis.provider}</p><h4>{analysis.summary}</h4></div>{analysis.hypotheses?.map(h => <div className="hypothesis" key={h.title}><span className="confidence">{Math.round(h.confidence * 100)}%</span><div><strong>{h.title}</strong><ul>{h.evidence?.map(e => <li key={e}>{e}</li>)}</ul></div></div>)}<div className="checks"><strong>Checks</strong>{analysis.checks?.map(c => <span key={c}>{c}</span>)}</div></div>}</div> }
function EvidenceBlock({ title, items, numbered }: { title: string; items: string[]; numbered?: boolean }) { return <div className="evidence"><h4>{title}</h4>{items.length ? <ol className={numbered ? '' : 'plain'}>{items.map(item => <li key={item}>{item}</li>)}</ol> : <Empty text="No evidence captured" />}</div> }
