# Runbook: reservation dependency failure

1. Confirm the incident and capture `/ops/incidents/{id}/evidence`.
2. Check trace IDs across reservation, payment, and notification events.
3. Inspect outbox retry and DLQ depth in the dashboard.
4. Enable the dependency fault only in the local runtime to reproduce
   compensation.
5. Verify stock returned and reservation status is `COMPENSATED`.
6. Clear the fault, process the retry, and run a new idempotency check.
7. Close the incident only after the error-budget and backlog alerts recover.

No runbook step executes automatically from model output.
