# Runbook: bad-release rollback

Freeze rollout, compare the reservation `release` marker with the incident
timeline, and preserve the evidence bundle. Roll back the deployment to the
last known-good immutable image digest. Verify health/readiness, reservation
idempotency, compensation, and notification delivery. Keep the incident open
until the SLO panels recover. The `bad-release` fault is a marker for a
controlled local scenario; it never changes a real deployment.
