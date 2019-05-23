#!/usr/bin/env bash

set -euo pipefail

go fmt ./{cmd,pkg}/...
go vet ./{cmd,pkg}/...
golint -set_exit_status ./{cmd,pkg}/...

cat fixtures/valid.yaml |  go run main.go