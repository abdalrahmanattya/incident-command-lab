# Runbook: reservation dependency failure

Owner: service operator. Evidence owner: incident commander. Exit approver: service owner.

1. Confirm the incident and capture `/ops/incidents/{id}/evidence` plus trace IDs.
2. Inspect outbox retry and DLQ depth using the local state endpoint or the approved acceptance evidence bundle.
3. Enable the dependency fault only in local mode, or use the protected acceptance gate; never toggle production state from model output.
4. Create a synthetic reservation and verify stock returned with status `COMPENSATED`.
5. Clear the fault, process the retry, and repeat the idempotency-key check.

Evidence: sanitized request IDs, reservation status, stock count, retry/DLQ state, and incident timeline. Exit criteria: compensation and idempotency checks pass, no unexpected DLQ growth remains, and the owner records the decision. Prometheus/Grafana configuration alone is not proof that an SLO or error budget recovered.
