# Threat model

## Assets

Reservation state, customer identifiers, payment-simulation events, incident
evidence, traces, and model prompts are the protected assets. Availability and
correct stock accounting are the primary safety properties.

## Trust boundaries

The customer API is untrusted input; event consumers are separate process
boundaries; PostgreSQL/NATS are infrastructure boundaries; and an AI provider
is an external processor. Uploaded telemetry and incident text are treated as
data, not instructions. The model adapter receives a minimized evidence
bundle, is advisory only, and cannot call tools or mutate state.

## Controls

- Request size limits, strict JSON decoding, bounded quantities and title
  validation prevent common input abuse.
- Idempotency keys and a unique database key prevent duplicate reservations.
- Transactional outbox state, retry limits, and DLQ status prevent silent loss.
- Compensation returns stock after simulated payment failure.
- W3C trace correlation supports evidence without copying secrets to logs.
- Containers run non-root with dropped capabilities and read-only filesystems.
- Cloud uses managed identity, Key Vault RBAC, private network planning,
  protected OIDC workflows, and short-lived ephemeral acceptance environments.
- Synthetic fixtures are mandatory for tests; no production data is accepted.

## Residual risks

The in-memory runtime is not durable and is single-process. Cloud PostgreSQL
private networking, secret rotation, and AKS policy must be validated in the
manual acceptance run. The local anonymous Grafana viewer is for localhost
only and must not be exposed publicly. A model can produce incorrect advice;
operators must verify every hypothesis against telemetry and runbooks.
