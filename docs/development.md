# Development

## Local checks

```sh
gofmt -d internal cmd
go test ./...
go test -race -cover ./...
go vet ./...
cd ui && npm ci && npm run typecheck && npm test && npm run build
cd .. && bash scripts/validate-repository.sh
```

The default Go server uses deterministic in-memory adapters when no connection URLs are set. Compose sets `DATABASE_URL` and `NATS_URL`, so the gateway uses PostgreSQL transactions and JetStream publication/worker behavior. The schema in `deploy/postgres` is applied at database startup.

## Fault drill and evidence

Enable one fault at a time with `POST /ops/faults`, create a reservation with an `Idempotency-Key`, inspect `/ops/state`, create an incident, and fetch its evidence. Record the request IDs and response bodies as sanitized evidence. `backlog` remains retryable until the third attempt; `dependency` compensates stock; `database` fails closed; `bad-release` is recorded in incident evidence. Runbooks identify owners, evidence, and exit criteria; no model output executes a step.

## Cloud boundary and gates

`terraform/azure` requires an explicitly selected subscription and protected GitHub OIDC environment. The cloud workflow is manual and uses `PLAN`, `APPLY`, `SMOKE`, and `DESTROY` gates with a concurrency lock. APPLY runs Terraform and builds immutable backend/UI images in ACR. SMOKE obtains AKS credentials, creates workflow-controlled `database` and `operator-auth` Secrets, resolves image digests, renders/applies manifests, runs migration/rollout checks, and performs the port-forwarded smoke. Capture only sanitized outputs and destroy the resource group after acceptance.
