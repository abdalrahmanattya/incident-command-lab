# Contributing

Run `docker run --rm -v "$PWD":/src -w /src golang:1.25-bookworm sh -c
'gofmt -w internal cmd && go test -race ./... && go vet ./...'` before opening
a change. Use synthetic fixtures only. Changes affecting Terraform, event
delivery, identity, or model prompts require an ADR and an evidence-matrix
entry. Cloud workflows are manual and protected; contributors must not add
credentials, remotes, deployments, or billable resources.
