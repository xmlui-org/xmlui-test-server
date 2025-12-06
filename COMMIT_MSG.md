refactor: Change package `common` to `localsvr`

- Rename `xmluisvr/common/` package directory to
  `xmluisvr/localsvr/` to better reflect package purpose and
  avoid generic naming.
- Update all import statements across codebase from
  `github.com/xmlui-org/xmlui-test-server/xmluisvr/common` to
  `github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr` in
  files including `api.go`, `endpoint.go`, `payload_args.go`,
  `api_config_v2.go`, `root_config_v1.go`, `database.go`,
  `sqlite3.go`, `server.go`, and test files.
- Update dependency versions in `go.mod` and `go.sum` files
  across `cmd/`, `test/`, and `xmluisvr/` modules:
  `go-cfgstore` v0.3.0 → v0.4.0, `go-cliutil` v0.2.1 → v0.3.0,
  `go-dt` v0.3.1 → v0.3.3.
- Bump internal version reference from v0.4.1 to v0.5.0 in
  `cmd/go.mod` and `test/go.mod`.
