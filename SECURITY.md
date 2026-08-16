# Security policy and pre-deployment checklist

This service accepts security reports through a private maintainer channel before public disclosure. Do not include credentials, customer data, or cloud identifiers in reports. The Compose stack is localhost-only synthetic data; never expose its anonymous Grafana viewer or local database to the Internet.

Before Azure acceptance, confirm protected OIDC/environment approval, remote-state ownership, current calculator estimate and budget threshold, private networking, Key Vault/CSI integration, workload identity application consumption, full workload hardening, secret rotation, recovery/restore evidence, real SLO/error-budget measurements, sanitized SMOKE evidence, and DESTROY verification. Key Vault RBAC being enabled is not evidence that secrets are provisioned or consumed.

The current planned secret path is workflow-created Kubernetes Secrets for `database` and `operator-auth` during SMOKE. Model output is untrusted advisory text and cannot execute remediation.
