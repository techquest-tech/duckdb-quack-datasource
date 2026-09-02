// Package duckdbquack 实现 DuckDB Quack Grafana 后端数据源。
package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/mehrabr/duckcall"
	"github.com/mehrabr/duckcall/wire"
)

// PluginID 插件唯一标识（也用于 grafana.ini allow_loading_unsigned_plugins）。
const PluginID = "esquel-duckdb-datasource"

// Settings 是插件配置（数据源配置页填写）。
// endpoint/token 存 secureJsonData；tablePrefix/queryTimeoutMS 存 jsonData。
type Settings struct {
	Endpoint       string `json:"endpoint"`
	Token          string `json:"token"`
	TablePrefix    string `json:"tablePrefix"`
	QueryTimeoutMS int64  `json:"queryTimeoutMS"`
}

// Query 是面板查询模型（{"sql": "..."}）。
type Query struct {
	SQL string `json:"sql"`
}

// NewDatasource 是 Grafana 后端实例工厂（每次数据源配置变更重建实例）。
func NewDatasource(_ context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	ds, err := newFromSettings(settings)
	if err != nil {
		return nil, err
	}
	return ds, nil
}

func newFromSettings(s backend.DataSourceInstanceSettings) (*DuckDBDatasource, error) {
	cfg := Settings{
		Endpoint:       "localhost:9494",
		QueryTimeoutMS: 30000,
	}
	if err := cfg.read(s); err != nil {
		return nil, err
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "localhost:9494"
	}
	if cfg.Token == "" {
		return nil, errors.New("quack token is required in datasource config")
	}
	if cfg.QueryTimeoutMS <= 0 {
		cfg.QueryTimeoutMS = 30000
	}
	return &DuckDBDatasource{cfg: cfg}, nil
}

// read 解析 jsonData 与 secureJsonData（token 在 secure）。
func (c *Settings) read(s backend.DataSourceInstanceSettings) error {
	if len(s.JSONData) > 0 {
		var jd struct {
			Endpoint       string `json:"endpoint"`
			TablePrefix    string `json:"tablePrefix"`
			QueryTimeoutMS int64  `json:"queryTimeoutMS"`
		}
		if err := json.Unmarshal(s.JSONData, &jd); err != nil {
			return fmt.Errorf("parse jsonData: %w", err)
		}
		if jd.Endpoint != "" {
			c.Endpoint = jd.Endpoint
		}
		if jd.TablePrefix != "" {
			c.TablePrefix = jd.TablePrefix
		}
		if jd.QueryTimeoutMS > 0 {
			c.QueryTimeoutMS = jd.QueryTimeoutMS
		}
	}
	if len(s.DecryptedSecureJSONData) > 0 {
		if v, ok := s.DecryptedSecureJSONData["token"]; ok {
			c.Token = v
		}
	}
	return nil
}

// DuckDBDatasource 实现 backend.QueryDataHandler + backend.CheckHealthHandler。
type DuckDBDatasource struct {
	cfg  Settings
	mu   sync.Mutex
	conn *duckcall.Conn
}

// QueryData 处理面板查询：每个查询 {"sql": "..."} → 只读校验 → Quack 执行 → DataFrame。
func (d *DuckDBDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	resp := backend.NewQueryDataResponse()
	for _, q := range req.Queries {
		var query Query
		if err := query.parse(q.JSON); err != nil {
			resp.Responses[q.RefID] = backend.DataResponse{Error: err}
			continue
		}
		frame, err := d.runQuery(ctx, query.SQL)
		if err != nil {
			resp.Responses[q.RefID] = backend.DataResponse{Error: err}
			continue
		}
		resp.Responses[q.RefID] = backend.DataResponse{Frames: data.Frames{frame}}
	}
	return resp, nil
}

func (q *Query) parse(raw []byte) error {
	q.SQL = "" // json.Unmarshal 到复用对象不会清零缺失字段
	if len(raw) == 0 {
		return errors.New("empty query")
	}
	if err := jsonUnmarshal(raw, q); err != nil {
		return fmt.Errorf("parse query: %w", err)
	}
	if strings.TrimSpace(q.SQL) == "" {
		return errors.New("empty sql")
	}
	return nil
}

// CheckHealth 探测 Quack 连通性。
func (d *DuckDBDatasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	server, err := d.whoami(ctx)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "quack connection failed: " + err.Error(),
		}, nil
	}
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "connected to " + server,
	}, nil
}

func (d *DuckDBDatasource) whoami(ctx context.Context) (string, error) {
	var whoami string
	err := d.withConn(ctx, func(conn *duckcall.Conn) error {
		res, err := conn.Query(ctx, "FROM whoami()")
		if err != nil {
			return err
		}
		defer res.Close(ctx)
		for ch, err := range res.Chunks(ctx) {
			if err != nil {
				return err
			}
			if ch.RowCount() > 0 && ch.ColumnCount() > 0 {
				if v, ok := ch.Column(0).Value(0).(string); ok {
					whoami = v
				}
			}
		}
		return nil
	})
	return whoami, err
}

// withConn 保证可用连接（惰性 Dial + 会话过期重连一次）。
func (d *DuckDBDatasource) withConn(ctx context.Context, fn func(*duckcall.Conn) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.conn == nil {
		if err := d.dialLocked(ctx); err != nil {
			return err
		}
	}
	err := fn(d.conn)
	if err != nil && errors.Is(err, wire.ErrConnectionExpired) {
		d.conn = nil
		if err2 := d.dialLocked(ctx); err2 != nil {
			return err2
		}
		return fn(d.conn)
	}
	return err
}

func (d *DuckDBDatasource) dialLocked(ctx context.Context) error {
	conn, err := duckcall.Dial(ctx, duckcall.Config{
		Endpoint: "http://" + d.cfg.Endpoint,
		Token:    d.cfg.Token,
	})
	if err != nil {
		return fmt.Errorf("quack dial %s: %w", d.cfg.Endpoint, err)
	}
	d.conn = conn
	return nil
}

// runQuery 执行只读 SQL 并转换为 Grafana DataFrame。
func (d *DuckDBDatasource) runQuery(ctx context.Context, sql string) (*data.Frame, error) {
	if !IsReadOnlySQL(sql) {
		return nil, errors.New("only read-only SQL (SELECT/SHOW/DESCRIBE/EXPLAIN/WITH/...) is allowed")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.cfg.QueryTimeoutMS)*time.Millisecond)
	defer cancel()

	var frame *data.Frame
	err := d.withConn(ctx, func(conn *duckcall.Conn) error {
		res, err := conn.Query(ctx, sql)
		if err != nil {
			return err
		}
		defer res.Close(ctx)

		schema := res.Schema()
		frame = data.NewFrame("duckdb")
		cols := make([][]any, len(schema.Columns))
		for i := range cols {
			cols[i] = make([]any, 0)
		}
		for ch, err := range res.Chunks(ctx) {
			if err != nil {
				return err
			}
			for row := 0; row < ch.RowCount(); row++ {
				for col := 0; col < ch.ColumnCount(); col++ {
					cols[col] = append(cols[col], ch.Column(col).Value(row))
				}
			}
		}
		// 按列类型转 typed 字段并加入 frame（Grafana 需要具体类型才能绘图/聚合）
		for i := range schema.Columns {
			typed := convertColumn(cols[i], schema.Columns[i].Type)
			frame.Fields = append(frame.Fields, data.NewField(schema.Columns[i].Name, nil, typed))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return frame, nil
}

func jsonUnmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
