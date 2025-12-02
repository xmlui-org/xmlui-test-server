#!/usr/bin/env bash
# Enhanced help for Makefile targets

cat <<'EOF'
XMLUI Test Server - Available Make Targets

USAGE:
  make <target>                  Run a target
  make test [dir]                Run tests (optionally for specific directory)

TARGETS:
  help             Show this help message
  ensure-valid     Run tidy, test, lint, and vet (validate everything)
  deps             Download & tidy modules
  tidy             Run go mod tidy on all modules
  fmt              Format code with gofmt (+ goimports if present)
  vet              Run go vet on all modules
  lint             Run golangci-lint on all modules
  build            Build the server binary -> bin/xmlui-test-server
  run              Build and run the server
  test             Run all tests (unit + integration)
  cover            Generate coverage report -> bin/coverage.html
  clean            Remove bin/ directory and build artifacts
  doterr           Sync doterr (internal)

EXAMPLES:
  make build                     Build the server
  make test                      Run all tests
  make test xmluisvr             Run tests in xmluisvr directory
  make test xmluisvr/cfgldr      Run tests in specific package
  make ensure-valid              Validate entire project

For more details on a specific target, check the corresponding script in ./scripts/
EOF
