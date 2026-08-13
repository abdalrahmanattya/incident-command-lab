# ADR 0001: deterministic local runtime

Status: accepted — 2026-08-13

The local v1 uses an in-memory domain store and deterministic analyst so every
workflow is testable without credentials or external services. PostgreSQL,
NATS JetStream, and cloud adapters remain explicit integration boundaries. This
keeps the default demo reproducible while preserving the durable schema and
deployment shape for an acceptance run.
