#!/usr/bin/env bash
set -Eeuo pipefail
source ./scripts/shared.sh

gofmt -s -w .
if command -v goimports >/dev/null 2>&1; then
  goimports -w .
else
  echo "tip: go install golang.org/x/tools/cmd/goimports@latest"
fi
