# duckdb-quack-datasource

Grafana **backend datasource** that connects directly to a DuckDB server over
the [Quack remote protocol](https://duckdb.org/docs/current/core_extensions/quack) —
no intermediate HTTP gateway, no embedded DuckDB engine, pure Go (no CGO) via
[duckcall](https://github.com/mehrabr/duckcall).

Panel SQL is executed by the plugin process directly over Quack; results are
returned as typed Grafana DataFrames (time / number / string columns), so time
series, table and stat panels work like any native datasource.

Why not the official [grafana-duckdb-datasource](https://github.com/motherduckdb/grafana-duckdb-datasource)?
It only supports local files and MotherDuck cloud — and a local DuckDB file is
exclusively locked by its owning process (`quack_serve`), so multi-process reads
are impossible. This plugin connects over Quack, coexisting with any number of
Quack writers/readers.

## Features

- Read-only SQL whitelist enforced in the plugin (SELECT/SHOW/DESCRIBE/EXPLAIN/WITH/...;
  INSERT/UPDATE/DROP/ATTACH/CALL/... and multi-statement are rejected)
- Lazy connect + automatic reconnect (`wire.ErrConnectionExpired`)
- Configurable query timeout and table prefix
- Simple frontend: config form (endpoint / token / prefix / timeout) + SQL query editor

## Layout

```
cmd/            Go backend entry (datasource.Manage)
pkg/            datasource.go (QueryData / CheckHealth / connection mgmt)
                sqlcheck.go (read-only whitelist) · convert.go (DuckDB → DataFrame)
src/            Frontend (ConfigEditor / QueryEditor), bundled with esbuild
plugin.json     Plugin metadata (id=panarm-duckdb-datasource, backend)
img/logo.svg
```

## Build

```bash
npm install && npm run build        # frontend → dist/module.js
make build-backend                  # backend → dist/gpx_duckdb_quack-{linux,darwin}
```

For a distributable plugin package, assemble a folder named after the plugin id
(`panarm-duckdb-datasource/`) containing `plugin.json`, `module.js` at the root,
`gpx_duckdb_quack` (linux amd64), `img/`, `LICENSE`, `README.md`.

## Install into Grafana

The plugin must be **signed** to be picked up by Grafana's plugin loader
(manual directory placement of unsigned plugins is not honored by recent
Grafana releases). Sign it via the
[Grafana plugin signing service](https://grafana.com/docs/grafana/latest/developers/plugins/sign-a-plugin/)
(the plugin id prefix `esquel` must match your grafana.com organization name),
then install from the catalog:

```bash
grafana cli plugins install panarm-duckdb-datasource
```

## Configure the datasource

- **Quack endpoint**: `host:port` of the Quack server (e.g. `localhost:9494`)
- **Token**: the Quack auth token (stored in Grafana's secureJsonData)
- **Table prefix** (optional): prepended to table names when you omit it in SQL
- **Query timeout** (ms, default 30000)

## Query example

```sql
SELECT started_at AS time, count(*) AS value
FROM monitor_tracing
WHERE started_at > $__timeFilter(started_at)
GROUP BY 1 ORDER BY 1
```

- `time` + `value` columns → time series panel; any columns → table/stat panels
- Only read-only statements are allowed (see pkg/sqlcheck.go)

## Test

```bash
go test ./pkg/                                          # whitelist / conversion / framework
QUACK_LIVE_ENDPOINT=127.0.0.1:9495 QUACK_LIVE_TOKEN=<t> go test ./pkg/ -run TestLiveQueryData -v
# real E2E against a local quack server
```

## License

MIT — see [LICENSE](https://github.com/techquest-tech/duckdb-quack-datasource/blob/v0.1.3/LICENSE).
