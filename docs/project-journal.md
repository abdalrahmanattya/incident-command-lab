# Project journal

## Current resume point

Operator console and public-readiness remediation accepted locally. Next
bounded action is a separately approved Azure acceptance run; cloud apply
remains unexecuted.

## Log

- 2026-08-13 — Created independent repository and credential-free Go runtime.
- 2026-08-13 — Added idempotency, outbox event states, retries, compensation,
  fault injection, incident evidence, deterministic analyst and trace headers.
- 2026-08-13 — Added local observability, kind, Azure Terraform, protected
  workflow surfaces and public safety documentation.
- 2026-08-13 — Verified Docker image build, HTTP smoke/idempotency drill, fault
  drill, Go race tests/vet, Terraform init/validate, main branch, and local
  AGENTS exclusion. Also verified `docker compose config` and Terraform
  validation without warnings; apply was not attempted.
- 2026-08-13 — Review remediation: wired PostgreSQL transactional reservation
  and outbox, NATS JetStream publishing/ack worker with max-delivery DLQ state,
  OTLP traces/metrics, strict Ollama/Azure HTTP adapters, Compose health-gated
  integrated runtime, and database-backed smoke evidence.
- 2026-08-13 — Final integration remediation: repository-backed faults,
  cancellation and state snapshot; durable processed-event keys and NATS
  Nak/max-delivery DLQ path; Tempo/Loki/Promtail/Grafana configuration; AKS
  workload identity, ACR pull, PostgreSQL private networking, migration schema,
  and immutable-image workflow wiring. Compose fault/recovery and telemetry
  configuration checks passed; cloud apply remains unexecuted.
- 2026-08-13 — Role-service remediation: gateway, payment, and notification
  now have distinct binaries and Compose containers. Migration is health-gated;
  payment/notification outcomes are persisted and filtered JetStream consumers
  process events. Integrated Compose evidence recorded seven payment outcomes,
  seven notification outcomes, and fourteen processed events. AKS worker,
  NATS, OTel, and migration manifests plus azurerm remote-state initialization
  were added.
- 2026-08-13 — Operator console remediation: added React/Vite responsive console
  with health, queue, reservations, reversible faults, incident evidence,
  deterministic analysis, compensation/recovery, local auth-disabled mode, and
  Entra SPA/operator-group cloud mode. Added `GET /ops/incidents` and contract
  test, nginx same-origin UI proxy, Compose UI service, AKS UI manifest, and
  immutable backend/UI image workflow wiring. Added landscape system and
  planned Azure architecture diagrams with visible README embeds. Cloud apply
  remains unexecuted; this is the exact resume point for Compose/UI/browser
  verification.
- 2026-08-13 — Acceptance verification completed: clean Compose same-origin
  Playwright exercised dependency compensation, fault disable/recovery,
  confirmed idempotency, incident create/list/evidence, and deterministic
  analysis; Go JWT tests covered missing, forged, wrong issuer/audience,
  expired, wrong-group, and valid RS256/JWKS tokens; Go test/race/vet, UI
  typecheck/build/component/E2E, Terraform, YAML, Compose, shell, and SVG
  contracts passed. Both landscape diagrams were rendered as full-canvas PNGs
  with sips and visually inspected; cloud SVG includes embedded official Azure
  icon assets and attribution. Azure apply/deployment and authenticated cloud
  smoke remain intentionally unexecuted.
