#!/usr/bin/env bash
set -Eeuo pipefail
source ./scripts/shared.sh

mkdir -p "${BIN_DIR}"

# Coverage over auto-discovered unit-test packages unless args provided
pkgs=()
while IFS= read -r pkg; do
  pkgs+=("$pkg")
done < <(get_test_directories "$@")

if [[ ${#pkgs[@]} -eq 0 ]]; then
  echo "No packages for coverage."; exit 0
fi

CGO_ENABLED="${CGO}" GOEXPERIMENT="${GOEXPERIMENT}" ${GO} test -coverprofile="${BIN_DIR}/coverage.unit.out" "${pkgs[@]}"
${GO} tool cover -func="${BIN_DIR}/coverage.unit.out" || true
${GO} tool cover -html="${BIN_DIR}/coverage.unit.out" -o "${BIN_DIR}/coverage.html" || true
echo "Open ${BIN_DIR}/coverage.html"
