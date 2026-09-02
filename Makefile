PLUGIN_ID ?= panarm-duckdb-datasource
EXEC ?= gpx_duckdb_quack
VERSION ?= 0.1.0

# 构建后端（darwin 测试 + linux 部署产物）
build-backend:
	GOWORK=off CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/$(EXEC)-darwin ./cmd
	GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/$(EXEC)-linux ./cmd

# 构建前端（esbuild → dist/module.js）
build-frontend:
	npm run build

# 组装可分发插件包（Grafana 规范：module.js 在插件根目录 + LICENSE/README）
package: build-backend build-frontend
	rm -rf dist/package/$(PLUGIN_ID) && mkdir -p dist/package/$(PLUGIN_ID)
	cp src/plugin.json dist/package/$(PLUGIN_ID)/
	cp LICENSE README.md dist/package/$(PLUGIN_ID)/
	cp -r img dist/package/$(PLUGIN_ID)/
	cp dist/module.js dist/package/$(PLUGIN_ID)/module.js
	cp dist/$(EXEC)-linux dist/package/$(PLUGIN_ID)/$(EXEC)
	chmod +x dist/package/$(PLUGIN_ID)/$(EXEC)
	cd dist/package && zip -r ../$(PLUGIN_ID)-$(VERSION).zip $(PLUGIN_ID)/
	@echo "==> dist/$(PLUGIN_ID)-$(VERSION).zip 就绪"

clean:
	rm -rf dist

.PHONY: build-backend build-frontend package clean
