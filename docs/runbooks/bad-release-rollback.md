# Runbook: bad-release rollback

Owner: release operator. Evidence owner: incident commander. Exit approver: service owner.

Freeze further rollout, compare the reservation `release` marker with the incident timeline, and preserve sanitized evidence. For an approved cloud acceptance only, select the last known-good immutable image digest through the protected workflow and roll back the deployment. Verify health/readiness, reservation idempotency, compensation, and notification delivery with synthetic requests.

Evidence: selected digest, rollout status, request IDs, reservation outcomes, and rollback timestamps. Exit criteria: readiness and synthetic behavior pass, no new retry/DLQ regression is observed, and the owner records whether to resume or destroy. The `bad-release` fault is a local marker; it never changes a real deployment automatically.
