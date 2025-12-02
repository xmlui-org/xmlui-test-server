#!/usr/bin/env bash
set -Eeuo pipefail
source ./scripts/shared.sh

echo "Cleaning build artifacts..."
rm -rf "${BIN_DIR}"
rm -f ext.tar.gz
rm -rf xmlui-test-server-build
echo "Removed ${BIN_DIR}/ and build artifacts"
