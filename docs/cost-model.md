# Cost model

## Local

Docker Compose uses local CPU, memory, disk, and image bandwidth only. Remove
the Compose volumes after a lab if the data is no longer needed.

## Azure acceptance

The Terraform baseline creates AKS (two small system nodes), ACR Basic,
PostgreSQL Flexible Server Burstable B1ms, Log Analytics, Key Vault, VNet, and
diagnostics. Actual cost depends on region and runtime. The protected workflow
must capture the provider cost estimate before apply, use a short acceptance
window, and run destroy immediately after smoke tests. No permanent hosting is
part of this repository.
