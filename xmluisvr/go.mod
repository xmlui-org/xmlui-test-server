module github.com/xmlui-org/xmlui-test-server/xmluisvr

go 1.25.3

require (
	github.com/jedib0t/go-pretty/v6 v6.7.5
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/mikeschinkel/go-cfgstore v0.4.0
	github.com/mikeschinkel/go-cliutil v0.3.0
	github.com/mikeschinkel/go-doterr v0.1.2
	github.com/mikeschinkel/go-dt v0.3.3
	github.com/mikeschinkel/go-dt/appinfo v0.2.1
	github.com/mikeschinkel/go-dt/dtglob v0.2.1
	github.com/mikeschinkel/go-dt/dtx v0.2.1
	github.com/mikeschinkel/go-jsonxtractr v0.1.1
	github.com/mikeschinkel/go-logutil v0.2.1
	github.com/mikeschinkel/go-pathvars v0.2.1
	github.com/mikeschinkel/go-rfc9457 v0.1.1
	github.com/mikeschinkel/go-sqlparams v0.1.2
	github.com/mikeschinkel/go-testutil v0.2.1
)

require (
	github.com/bmatcuk/doublestar/v4 v4.9.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)

replace github.com/mikeschinkel/go-dt => ../../../go-pkgs/go-dt
