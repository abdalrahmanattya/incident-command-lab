# Runbook: protected Azure acceptance

Owner: platform operator. Evidence owner: incident commander. Exit approver: service owner and budget owner.

## PLAN

Confirm the selected subscription, region, remote-state owner, current calculator estimate, pre-approved budget threshold, and protected `cloud-acceptance` environment. Review Terraform changes and image/resource scope. Exit criteria: expected resources only, cost owner approval recorded, and the plan artifact is sanitized.

## APPLY

Run Terraform and build immutable backend/UI images in ACR. Do not claim deployment from a successful plan or image build. Exit criteria: Terraform apply completes, image digests are recorded, and the owner approves proceeding to SMOKE.

## SMOKE

Obtain AKS credentials, create the `database` and `operator-auth` Kubernetes Secrets from protected workflow inputs, resolve image digests, render and apply manifests, run migration and rollout/readiness checks, and execute the port-forwarded synthetic smoke. Capture only sanitized request IDs, statuses, digests, and timestamps. Exit criteria: migration, rollout, reservation/idempotency, compensation, analyst-read-only, and health checks pass.

## DESTROY

Destroy the ephemeral resource group and verify Terraform state/output cleanup. Record retained resources explicitly if policy requires them. Exit criteria: no unapproved acceptance resources remain and the budget owner signs off.

Key Vault RBAC being enabled is not evidence of configured secrets or runtime consumption. Workload identity application consumption, Key Vault/CSI integration, full workload hardening, measured SLO/error-budget evidence, recovery validation, and the complete cloud acceptance remain pre-deployment blockers.
