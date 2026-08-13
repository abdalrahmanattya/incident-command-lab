# Incident Command Lab

Incident Command Lab is a fault-injectable ticket-reservation system and
operations console for inspecting and operating distributed reliability
scenarios. It contains a Go gateway plus catalogue, reservation, payment-simulator,
and notification service entry points; a transactional-outbox-shaped event flow;
idempotency; retries and backlog/DLQ simulation; saga compensation; and an
advisory incident evidence analyst.

![System architecture](docs/diagrams/system-architecture.svg)

![Planned Azure architecture](docs/diagrams/cloud-architecture.svg)

Incident Command Lab is a small, intentionally fault-injectable distributed
systems service and operator console. It is an operator-facing reliability
system, not a production payment system.

## Why it exists

Operational response is easier to inspect when failure, evidence, recovery,
and limits are visible together. The console makes those relationships
replayable without cloud credentials, while the planned Azure boundary remains
explicit.

## What it does

Customers can inspect products, create an idempotent reservation, retrieve it,
and cancel it. Operators can enable bounded faults, inspect state and create an
incident. An incident evidence bundle includes a timeline, signals, and
versioned runbook references. `/ops/incidents/{id}/analyze` uses a deterministic
analyst by default. Ollama and Azure OpenAI are adapter selections only; remote
analysis is disabled unless an operator explicitly enables it and the adapter
is connected to a reviewed gateway. Analysis never performs remediation.

The default runtime is in-memory so unit tests work without cloud,
database, broker, or model credentials. When `DATABASE_URL` and `NATS_URL` are
set, the gateway uses PostgreSQL for the atomic reservation/idempotency/outbox
transaction and NATS JetStream for durable event publication, retries, and
max-delivery worker acknowledgement. Docker Compose adds PostgreSQL, NATS
JetStream, OpenTelemetry Collector, Prometheus, Loki, Tempo, and Grafana for
local integration services. The Azure deployment definition is Terraform for
AKS, ACR, PostgreSQL Flexible Server, Key Vault, managed
identity, network, and monitoring. It is manual and ephemeral by design.

## How it works

The Go gateway coordinates catalogue and reservation behaviour. A reservation
is idempotent and writes a transactional-outbox-shaped event sequence. Compose
adds PostgreSQL and NATS JetStream so workers can publish, retry, acknowledge,
and route max-delivery messages to a DLQ. Payment and notification are
simulators; dependency failure compensates stock. OTel correlation is exported
when configured. The deterministic analyst returns cited hypotheses and checks
but never changes system state.

The React operator console reads health, queue, reservation, fault, incident,
evidence, runbook, analysis, and recovery surfaces. Local mode disables auth
explicitly. Cloud mode uses Microsoft Entra SPA MSAL and requires the
configured operator group; the backend applies the same group gate to `/ops`.

## Run it

Requirements: Go 1.23+, or Docker. No credentials are needed.

```sh
go run ./cmd/gateway
curl http://localhost:8080/v1/catalogue
curl -X POST http://localhost:8080/v1/reservations \
  -H 'content-type: application/json' -H 'idempotency-key: demo-1' \
  -d '{"customer_id":"customer-1","product_id":"concert","quantity":2}'
```

The operational workflow is reproducible:

```sh
curl -X POST http://localhost:8080/ops/faults -H 'content-type: application/json' -d '{"fault":"dependency","enabled":true}'
curl -X POST http://localhost:8080/ops/incidents -H 'content-type: application/json' -d '{"title":"Payment dependency degraded","severity":"SEV2"}'
go test ./...
```

For the full operator and observability runtime, run `docker compose up --build`
and open `http://localhost:8080`. Grafana is at
`http://localhost:3000` (anonymous local viewer, no production data). The
Compose broker/database are integration dependencies; the Go process remains
usable without them.

## Guided demo

1. Open the console and confirm the health cards show the local path.
2. Enable `dependency`, create a reservation, and observe `COMPENSATED` plus
   stock-release recovery.
3. Disable the fault, create the same reservation twice with one idempotency
   key, and verify the same reservation ID is returned.
4. Enable `backlog`, create an incident, inspect its signals/timeline/runbook,
   then run deterministic advisory analysis. Analysis is read-only.
5. Disable every fault and cancel a confirmed reservation. Compose cleanup is
   `docker compose down -v`.

## Tests and verification

`go test ./...`, `go test -race ./...`, and `go vet ./...` cover the Go
contract. The console uses `npm run typecheck`, `npm run build`, `npm test`,
and Playwright E2E when a browser is installed. `terraform fmt -check`,
`terraform init -backend=false`, and `terraform validate` cover the planned
Azure shape. CI also checks Trivy, Gitleaks, immutable image provenance/SBOM,
Compose configuration, Kubernetes YAML, links, and SVG XML.

## Azure and security

Terraform and AKS manifests describe a manual, ephemeral Azure deployment with
AKS workload identity, ACR digest-pinned images, private PostgreSQL/DNS,
Key Vault RBAC, Entra operator-group access, and Azure Monitor diagnostics.
Cloud apply, smoke, and destroy require protected workflow approval and were
not executed for this repository state. The protected cloud workflow requires
the Entra API audience/client ID, SPA client ID, tenant ID, operator group ID,
and a short-lived `AZURE_SMOKE_BEARER` secret for the port-forwarded smoke;
smoke fails closed when that bearer is missing.

## API surface

Customer: `GET /v1/catalogue`, `POST /v1/reservations`,
`GET /v1/reservations/{id}`, `POST /v1/reservations/{id}/cancel`.

Operator: `POST /ops/faults`, `GET /ops/state`, `POST /ops/incidents`,
`GET /ops/incidents`,
`GET /ops/incidents/{id}`, `GET /ops/incidents/{id}/evidence`, and
`POST /ops/incidents/{id}/analyze`.

Every response includes a W3C `traceparent` and `x-trace-id`. OTLP traces and
metrics export to the Collector when `OTEL_EXPORTER_OTLP_ENDPOINT` is set;
JSON logs carry the correlation identifiers. Faults are `latency`, `dependency`,
`backlog`, `duplicate`, `database`, and `bad-release`; each is bounded and
reversible through the operator endpoint.

## Repository map

- `internal/app`: reservation domain, outbox state, faults, incidents.
- `internal/server`: HTTP contract and validation.
- `internal/observability`: W3C correlation middleware.
- `internal/analysis`: deterministic and remote-adapter boundary.
- `deploy/`: kind, observability, and local container configuration.
- `terraform/azure`: production-shaped, manual Azure infrastructure.
- `docs/`: architecture, ADRs, threat model, runbooks, cost and evidence.

## Safety and limitations

This is a local reliability system, not a payment processor. Payments and
notifications are simulations. Synthetic data only. The in-memory mode is
single-process; integrated durability is validated by Compose and the AKS
workflow described in `docs/development.md`. Cloud workflows are manual,
authenticated, protected, and require environment approval for plan, apply,
smoke, and destroy. No credentials are stored in this repository. This project
does not validate Azure availability, production security posture, payment
correctness, or disaster recovery until a separately approved acceptance run.

## Azure deployment method and status

The exact deployment path is the protected, manually dispatched workflow in
`.github/workflows/cloud.yml`, described in [`docs/development.md`](docs/development.md).
It runs `terraform init` with the approved remote-state inputs, validates and
plans `terraform/azure`, and only permits apply, smoke, or destroy after the
`cloud-acceptance` environment and the required typed confirmation. The apply
path builds immutable backend and UI images in ACR, renders the AKS manifests,
and the smoke path verifies the deployed services through a port-forwarded
gateway. Azure apply, smoke, and destroy have not been executed for this
repository state.
