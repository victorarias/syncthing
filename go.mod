module github.com/victorarias/syncthing

go 1.26.2

require (
	github.com/d4l3k/messagediff v1.2.1
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/jackpal/gateway v1.2.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mattn/go-sqlite3 v1.14.50
	github.com/miscreant/miscreant.go v0.0.0-20200214223636-26d376326b75
	github.com/pierrec/lz4/v4 v4.1.29
	github.com/prometheus/client_golang v1.24.1
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9
	github.com/shirou/gopsutil/v4 v4.26.7
	github.com/syncthing/notify v0.0.0-20250528144937-c7027d4f7465
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d
	github.com/thejerf/suture/v4 v4.0.6
	github.com/wlynxg/anet v0.0.5
	golang.org/x/crypto v0.55.0
	golang.org/x/sys v0.47.0
	golang.org/x/text v0.41.0
	google.golang.org/protobuf v1.36.12
	modernc.org/sqlite v1.57.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/calmh/xdr v1.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.10.2 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.12.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.1 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/telemetry v0.0.0-20260811182544-a038080d80e5 // indirect
	golang.org/x/tools v0.49.0 // indirect
	modernc.org/libc v1.74.4 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

tool (
	github.com/calmh/xdr/cmd/genxdr
	github.com/maxbrunsfeld/counterfeiter/v6
	golang.org/x/tools/cmd/goimports
)
