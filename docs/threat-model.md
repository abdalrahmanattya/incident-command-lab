# Threat model

## Assets and boundaries

Reservation state, customer identifiers, payment-simulation events, incident evidence, traces, and model prompts are protected assets. The customer API is untrusted input; event consumers, PostgreSQL/NATS, and the remote model provider are separate boundaries. Incident text and telemetry are data, not instructions. The analyst receives a minimized evidence bundle, is advisory-only, and cannot call tools or mutate state.

## Implemented controls

- Request limits, strict JSON decoding, bounded quantities/title validation, idempotency keys, and unique database keys.
- Transactional outbox state, retry limits, max-delivery/DLQ status, and compensation after simulated payment failure.
- W3C trace correlation without copying secrets into logs.
- Non-root, read-only filesystem, and dropped-capability controls are asserted only for the gateway and operator-console workloads.
- Synthetic fixtures only; no production data is accepted.

## Planned controls and current gaps

- Key Vault is provisioned with RBAC enabled, but has no secrets, role assignments, or runtime consumption in this repository. Key Vault/CSI integration is a pre-deployment blocker.
- Workload identity/OIDC/federation and service-account annotations are provisioned as infrastructure direction, but application consumption is incomplete. The current planned secret path is workflow-created Kubernetes Secrets for `database` and `operator-auth` during SMOKE.
- Full workload hardening, private networking, secret rotation, recovery/restore evidence, and real metric/SLO/error-budget validation remain pre-deployment blockers.
- Prometheus rules and Grafana JSON are configuration, not measured SLO/error-budget evidence.

## Residual risks

The in-memory runtime is single-process and not durable. The local anonymous Grafana viewer must remain localhost-only. Model output can be incorrect; operators must verify hypotheses against supplied evidence, telemetry, and runbooks. Azure availability, production security posture, and disaster recovery require a separately approved cloud acceptance run.
