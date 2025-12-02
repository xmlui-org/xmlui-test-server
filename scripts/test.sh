#!/usr/bin/env bash
set -Eeuo pipefail
source ./scripts/shared.sh

flags=()
[[ "${RACE}" == "1" ]] && flags+=("-race")

# Accept optional directory args; otherwise discover all (including integration tests)
pkgs=()
if [[ $# -gt 0 ]]; then
  while IFS= read -r pkg; do
    pkgs+=("$pkg")
  done < <(get_test_directories "$@")
else
  # Discover all test directories
  while IFS= read -r pkg; do
    pkgs+=("$pkg")
  done < <(get_test_directories)

  # Ensure integration test directory is included if it exists
  if [[ -d "${ITEST_DIR}" ]] && [[ ! " ${pkgs[*]} " =~ " ./${ITEST_DIR}/... " ]]; then
    pkgs+=( "./${ITEST_DIR}/..." )
  fi
fi

if [[ ${#pkgs[@]} -eq 0 ]]; then
  echo "No testable directories found."; exit 0
fi

CGO_ENABLED="${CGO}" CGO_LDFLAGS="${CGO_LDFLAGS}" GOEXPERIMENT="${GOEXPERIMENT}" ${GO} test "${flags[@]}" "${pkgs[@]}"
