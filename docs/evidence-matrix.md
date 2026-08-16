# Evidence matrix

Local checks below are repository evidence, not Azure deployment evidence.

| Capability | Dated local evidence | Cloud evidence required | Status |
|---|---|---|---|
| Customer reservation | 2026-08-13 Go tests and API curl | AKS smoke workflow | local verified |
| Idempotency | 2026-08-13 `TestReserveIsIdempotentAndEmitsOutbox` | PostgreSQL integration run | local verified |
| Compensation | 2026-08-13 dependency fault test | AKS smoke run | local verified |
| Retry/backlog | 2026-08-13 backlog test and state API | NATS JetStream run | local verified |
| Trace correlation | 2026-08-13 `go test ./...` response-header assertions | OTel collector evidence | local code present |
| Analyst safety | 2026-08-16 strict schema/citation tests in `internal/analysis` | approved provider acceptance | local code present |
| Repository validation | 2026-08-16 Compose config, Terraform fmt, Markdown/SVG checks | protected workflow run | local checks present |
| SLO/alerts | Prometheus rules and Grafana JSON reviewed 2026-08-16 | measured Azure Monitor/SLO run | configuration only |
| Infrastructure | Terraform fmt/validation path reviewed 2026-08-16 | apply/smoke/destroy | not executed |

Authoritative prior hosted CI run [31732036848](https://github.com/abdalrahmanattya/incident-command-lab/actions/runs/31732036848) used the then-current single-job workflow: Go formatting/tests/race/coverage/vet, root Trivy filesystem scan, and Gitleaks. It is CI evidence only. The expanded four-gate workflow awaits its own pull-request run.
