PLUGIN_ID ?= esquel-duckdb-quack-datasource
EXEC ?= gpx_duckdb_quack
VERSION ?= 0.1.0

# 构建后端（darwin 测试 + linux 部署产物）
build-backend:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/$(EXEC)-darwin .
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/$(EXEC)-linux .

# 构建前端（esbuild → dist/module.js）
build-frontend:
	npm run build

# 组装完整插件目录（Grafana 可加载的结构）
dist: build-backend build-frontend
	@echo "==> 插件产物 dist/ 结构："
	@ls -la dist/
	@echo "==> 部署到 Grafana 前需把 dist 下文件放到 /var/lib/grafana/plugins/$(PLUGIN_ID)/ 并配置 allow_loading_unsigned_plugins"

clean:
	rm -rf dist

.PHONY: build-backend build-frontend dist clean
