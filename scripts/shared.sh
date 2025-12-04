#!/usr/bin/env bash
set -Eeuo pipefail

# Shared defaults; override via env if needed.
BIN="${BIN:-xmlui-localsvr}"
BIN_DIR="${BIN_DIR:-bin}"
CMD_DIR="${CMD_DIR:-cmd}"
PKG_DIR="${PKG_DIR:-xmluisvr}"
ITEST_DIR="${ITEST_DIR:-test}"

GO="${GO:-go}"
CGO="${CGO:-1}"   # sqlite3 requires CGO
RACE="${RACE:-1}"
GOEXPERIMENT="${GOEXPERIMENT:-jsonv2}"

# Legacy aliases for backwards compatibility
BINARY_NAME="${BIN}"
BINARY_PATH="${BIN_DIR}/${BIN}"
GO_SQLITE3_PATH="xmluisvr/sqlite3"

# Version metadata (override in CI if desired)
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo v0.0.0-dev)}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 0000000)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

# ldflags; append extra with LDFLAGS_APPEND if needed
LDFLAGS_BASE="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE} -X main.builtBy=make"
LDFLAGS="${LDFLAGS:-${LDFLAGS_BASE} ${LDFLAGS_APPEND:-}}"

LOCALBIN="${LOCALBIN:-${BIN_DIR}/${BIN}}"

# SQLite settings
SQLITE_VERSION="3450200"
SQLITE_YEAR="2024"
SQLITE_CFLAGS="-DSQLITE_ENABLE_LOAD_EXTENSION -DSQLITE_ALLOW_LOAD_EXTENSION"

# CGO linker flags
# Suppress LC_DYSYMTAB warnings on macOS (see: https://github.com/golang/go/issues/61229)
# This is a known cosmetic warning with Apple's ld-prime linker in Xcode 15+
# Will be fixed in Go 1.26 (https://github.com/golang/go/issues/75274)
CGO_LDFLAGS="${CGO_LDFLAGS:--Xlinker -w}"

# Extension settings
EXTENSION_VERSION="v1.2.0"
STEAMPIPE_EXTENSION="steampipe_sqlite_github.so"

# Build directories
BUILD_DIR="xmlui-test-server-build"
SQLITE_INSTALL_DIR="${BUILD_DIR}/sqlite-install"

# Build flags
BUILD_TAGS="sqlite3_load_extension"

# Platform detection
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$ARCH" in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64)
            ARCH="arm64"
            ;;
    esac
    
    export OS ARCH
}

# Helper functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

error() {
    echo "[ERROR] $*" >&2
    exit 1
}

ensure_bin_dir() {
    if [[ ! -d "$BIN_DIR" ]]; then
        log "Creating $BIN_DIR directory..."
        mkdir -p "$BIN_DIR"
    fi
}

check_macos() {
    if [[ "$OS" != "darwin" ]]; then
        error "This operation is specifically for macOS"
    fi
}

check_macos_arm() {
    if [[ "$OS" != "darwin" || "$ARCH" != "arm64" ]]; then
        error "This operation is specifically for macOS ARM (Apple Silicon)"
    fi
}

check_macos_intel() {
    if [[ "$OS" != "darwin" || "$ARCH" != "amd64" ]]; then
        error "This operation is specifically for macOS Intel (x86_64)"
    fi
}

check_patched_sqlite3() {
    if [[ ! -d "$GO_SQLITE3_PATH" ]]; then
        error "Patched go-sqlite3 not found at $GO_SQLITE3_PATH"
    fi
}

# Discover all module roots (dirs containing go.mod), ignoring vendor/.git/bin/build
get_module_dirs() {
  find . -type f -name go.mod -print0 \
  | xargs -0 -n1 dirname \
  | sed 's#^\./##' \
  | grep -Ev '(^|/)(vendor|\.git|'"${BIN_DIR:-bin}"'|'"${BUILD_DIR:-build}"')(/|$)' \
  | sort -u
}

# Test directory discovery
discover_test_directories() {
    local base_dirs=()

    # Find directories with Go files that have test files or could have tests
    while IFS= read -r -d '' dir; do
        # Skip vendor, .git, and build directories
        if [[ "$dir" == *"vendor"* || "$dir" == *".git"* || "$dir" == *"${BUILD_DIR}"* ]]; then
            continue
        fi

        # Check if directory contains Go files (*.go) and has test potential
        if find "$dir" -maxdepth 1 -name "*.go" | grep -q .; then
            # Convert absolute path to relative and add ./
            local rel_dir=$(realpath --relative-to=. "$dir" 2>/dev/null || echo "$dir")
            base_dirs+=("./$rel_dir/...")
        fi
    done < <(find . -type d -print0)

    # Remove duplicates and sort
    printf '%s\n' "${base_dirs[@]}" | sort -u
}

# Get test directories - either discovered or from arguments
get_test_directories() {
    if [[ $# -gt 0 ]]; then
        # Use provided directories
        for dir in "$@"; do
            # Ensure ./ prefix and /... suffix for Go module paths
            if [[ "$dir" != ./* ]]; then
                dir="./$dir"
            fi
            if [[ "$dir" != */... ]]; then
                dir="$dir/..."
            fi
            echo "$dir"
        done
    else
        # Discover directories automatically
        discover_test_directories
    fi
}

# Initialize platform detection when sourced
detect_platform