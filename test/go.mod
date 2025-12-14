module test

go 1.25.3

require (
	github.com/mikeschinkel/go-cfgstore v0.4.1
	github.com/mikeschinkel/go-cliutil v0.3.0
	github.com/mikeschinkel/go-dt v0.3.3
	github.com/mikeschinkel/go-dt/appinfo v0.2.1
	github.com/mikeschinkel/go-fsfix v0.2.2
	github.com/mikeschinkel/go-rfc9457 v0.1.1
	github.com/mikeschinkel/go-sqlparams v0.1.2
	github.com/mikeschinkel/go-testutil v0.2.1
	github.com/xmlui-org/xmlui-test-server/xmluisvr v0.5.0
)

require (
	github.com/lib/pq v1.10.9 // indirect
	github.com/mattn/go-sqlite3 v1.14.32 // indirect
	github.com/mikeschinkel/go-doterr v0.1.2 // indirect
	github.com/mikeschinkel/go-dt/dtx v0.2.1 // indirect
	github.com/mikeschinkel/go-jsonxtractr v0.1.1 // indirect
	github.com/mikeschinkel/go-logutil v0.2.1 // indirect
	github.com/mikeschinkel/go-pathvars v0.2.1 // indirect
)

replace github.com/xmlui-org/xmlui-test-server/xmluisvr => ../xmluisvr

replace github.com/mikeschinkel/go-cfgstore => ../../../go-pkgs/go-cfgstore

replace github.com/mikeschinkel/go-cfgstore/cstest => ../../../go-pkgs/go-cfgstore/cstest

replace github.com/mikeschinkel/go-cfgstore/test => ../../../go-pkgs/go-cfgstore/test
