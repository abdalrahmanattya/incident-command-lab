# Evidence matrix

| Capability | Local evidence | Cloud evidence | Status |
|---|---|---|---|
| Customer reservation | Go tests and API curl | AKS smoke workflow | local verified |
| Idempotency | `TestReserveIsIdempotentAndEmitsOutbox` | PostgreSQL integration run | local verified |
| Compensation | dependency fault test | AKS smoke run | local verified |
| Retry/backlog | backlog test and state API | NATS JetStream run | local verified |
| Trace correlation | `traceparent` response header | OTel collector panels | local code present |
| SLO/alerts | Prometheus rules and Grafana JSON | Azure Monitor run | configuration present |
| Analyst safety | deterministic tests and adapter gate | provider acceptance | local code present |
| Infrastructure | Terraform validate | apply/smoke/destroy | not executed |
