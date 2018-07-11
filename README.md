# Cloud-cost-tracker

This program can be used to extract and push usage costs from AWS and Azure to InfluxDB.

## Set up

`go get github.com/lentzi90/cloud-cost-tracker/...`

Get dependencies: `./downloadDependencies.sh`.

Generate mocks by executing `go generate ./...`.
Tests can then be run with `go test ./...`.
