#!/bin/bash
#
# Standard build script for xmlui-test-server (no extension support)
#

set -e

# Get script directory and source variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/shared.sh"

main() {
    log "Building standard $BINARY_NAME..."
    
    # Ensure bin directory exists
    ensure_bin_dir
    
    # Clean existing binary
    rm -f "$BINARY_PATH"
    
    # Build the binary from cmd directory
    CGO_LDFLAGS="${CGO_LDFLAGS}" GOEXPERIMENT=jsonv2 go build -v -o "$BINARY_PATH" ./cmd
    
    log "Build complete! Binary: $BINARY_PATH"
}

# Run main function
main "$@"