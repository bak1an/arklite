module github.com/bak1an/arklite

go 1.24.0

require (
	github.com/dustin/go-humanize v1.0.1
	github.com/go-sql-driver/mysql v1.9.3
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/spf13/pflag v1.0.10
	github.com/stephenafamo/bob v0.41.1
	golang.org/x/term v0.36.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/aarondl/opt v0.0.0-20250607033636-982744e1bd65 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/qdm12/reprint v0.0.0-20200326205758-722754a53494 // indirect
	github.com/stephenafamo/scan v0.7.0 // indirect
	go.uber.org/nilaway v0.0.0-20250821055425-361559d802f0 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/telemetry v0.0.0-20251008203120-078029d740a8 // indirect
	golang.org/x/tools v0.38.0 // indirect
)

tool (
	go.uber.org/nilaway/cmd/nilaway
	golang.org/x/tools/cmd/goimports
)
