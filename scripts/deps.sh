#!/usr/bin/env bash
set -Eeuo pipefail
source ./scripts/shared.sh

while IFS= read -r mod; do
  (cd "$mod" && ${GO} mod download && ${GO} mod tidy)
done < <(get_module_dirs)
