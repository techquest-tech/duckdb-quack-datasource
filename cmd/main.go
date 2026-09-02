// duckdb-quack-datasource：Grafana 后端数据源插件，纯 Quack 直连 DuckDB。
//
// 与官方 grafana-duckdb-datasource（只支持本地文件 / MotherDuck 云）不同，
// This plugin connects directly to a DuckDB Quack server over the Quack
// remote protocol (pure Go duckcall, no CGO, no HTTP gateway). Queries are
// read-only (whitelist enforced in sqlcheck.go).
package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"summationsolutions.com/duckdb-quack-datasource/pkg"
)

func main() {
	if err := datasource.Manage(pkg.PluginID, pkg.NewDatasource, datasource.ManageOpts{}); err != nil {
		log.DefaultLogger.Error("duckdb-quack-datasource failed", "error", err)
		os.Exit(1)
	}
}
