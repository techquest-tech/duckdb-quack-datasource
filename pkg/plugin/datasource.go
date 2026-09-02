package plugin

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
	"github.com/techquest-tech/duckdb-quack-datasource/pkg/models"
)

// Make sure Datasource implements required interfaces.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// NewDatasource creates a new datasource instance from Grafana settings.
func NewDatasource(_ context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	cfg, err := models.LoadPluginSettings(settings)
	if err != nil {
		return nil, err
	}
	if cfg.Secrets == nil || cfg.Secrets.Token == "" {
		return nil, errors.New("quack token is required in datasource config")
	}
	return &Datasource{cfg: cfg}, nil
}

// Datasource connects to a DuckDB Quack server and executes read-only SQL.
type Datasource struct {
	cfg  *models.PluginSettings
	mu   sync.Mutex
	conn *duckcall.Conn
}

// Dispose closes the underlying Quack connection.
func (d *Datasource) Dispose() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.conn != nil {
		_ = d.conn.Close(context.Background())
		d.conn = nil
	}
}

// Query is the panel query model ({"sql": "..."}).
type Query struct {
	SQL string `json:"sql"`
}

func (q *Query) parse(raw []byte) error {
	q.SQL = "" // json.Unmarshal into a reused struct does not clear missing fields
	if len(raw) == 0 {
		return errors.New("empty query")
	}
	if err := json.Unmarshal(raw, q); err != nil {
		return fmt.Errorf("parse query: %w", err)
	}
	if strings.TrimSpace(q.SQL) == "" {
		return errors.New("empty sql")
	}
	return nil
}

// QueryData executes each query ({"sql": "..."}) against Quack and returns typed DataFrames.
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	for _, q := range req.Queries {
		var query Query
		if err := query.parse(q.JSON); err != nil {
			response.Responses[q.RefID] = backend.DataResponse{Error: err}
			continue
		}
		frame, err := d.runQuery(ctx, query.SQL)
		if err != nil {
			response.Responses[q.RefID] = backend.DataResponse{Error: err}
			continue
		}
		response.Responses[q.RefID] = backend.DataResponse{Frames: data.Frames{frame}}
	}
	return response, nil
}

// CheckHealth dials Quack and reports connectivity.
func (d *Datasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
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

func (d *Datasource) whoami(ctx context.Context) (string, error) {
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

// withConn guarantees a usable connection (lazy Dial + one reconnect on expiry).
func (d *Datasource) withConn(ctx context.Context, fn func(*duckcall.Conn) error) error {
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

func (d *Datasource) dialLocked(ctx context.Context) error {
	conn, err := duckcall.Dial(ctx, duckcall.Config{
		Endpoint: "http://" + d.cfg.Endpoint,
		Token:    d.cfg.Secrets.Token,
	})
	if err != nil {
		return fmt.Errorf("quack dial %s: %w", d.cfg.Endpoint, err)
	}
	d.conn = conn
	return nil
}

// runQuery executes a read-only SQL statement and converts it to a Grafana DataFrame.
func (d *Datasource) runQuery(ctx context.Context, sql string) (*data.Frame, error) {
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
