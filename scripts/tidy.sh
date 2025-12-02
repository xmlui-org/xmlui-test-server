#!/usr/bin/env bash
set -Eeuo pipefail
source ./scripts/shared.sh

echo "Running go mod tidy on all go.mod files..."
while IFS= read -r mod; do
  echo "  → $mod"
  (cd "$mod" && ${GO} mod tidy)
done < <(get_module_dirs)
echo "Done!"
