package pkg

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/mehrabr/duckcall"
)

// execLive 经插件的连接管理执行任意 SQL（测试辅助）。
func (d *DuckDBDatasource) execLive(ctx context.Context, sql string) error {
	return d.withConn(ctx, func(conn *duckcall.Conn) error {
		res, err := conn.Query(ctx, sql)
		if err != nil {
			return err
		}
		defer res.Close(ctx)
		for _, err := range res.Chunks(ctx) {
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// TestLiveQueryData 真实 E2E：连本地 quack server 执行 QueryData，
// 验证 SQL→Frame 全链路（时间/数值/字符串列类型转换）。
func TestLiveQueryData(t *testing.T) {
	endpoint := os.Getenv("QUACK_LIVE_ENDPOINT")
	token := os.Getenv("QUACK_LIVE_TOKEN")
	if endpoint == "" || token == "" {
		t.Skip("set QUACK_LIVE_ENDPOINT/QUACK_LIVE_TOKEN")
	}
	ds := &DuckDBDatasource{cfg: Settings{Endpoint: endpoint, Token: token, QueryTimeoutMS: 10000}}

	prefix := "e2e_panel"
	// 清表保证断言确定性（execLive 直连执行，不受只读校验约束）
	if err := ds.execLive(context.Background(), "DROP TABLE IF EXISTS "+prefix); err != nil {
		t.Fatal(err)
	}
	if err := ds.execLive(context.Background(),
		"CREATE TABLE "+prefix+" (t TIMESTAMPTZ, v BIGINT, s VARCHAR)"); err != nil {
		t.Fatal(err)
	}
	if err := ds.execLive(context.Background(),
		"INSERT INTO "+prefix+" VALUES (now(), 42, 'hello'), (now() + INTERVAL 1 MINUTE, 43, 'world')"); err != nil {
		t.Fatal(err)
	}

	// 调试：直接 runQuery 看 schema
	frame1, err1 := ds.runQuery(context.Background(), "SELECT t, v, s FROM e2e_panel ORDER BY t")
	if err1 != nil {
		t.Logf("runQuery err: %v", err1)
	} else if frame1 != nil {
		t.Logf("runQuery direct: %d rows x %d cols, fields=%v", frame1.Rows(), len(frame1.Fields), frameFieldNames(frame1))
	}

	resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{{
			RefID: "A",
			JSON:  json.RawMessage(`{"sql":"SELECT t, v, s FROM e2e_panel ORDER BY t"}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	dr := resp.Responses["A"]
	if dr.Error != nil {
		t.Fatal(dr.Error)
	}
	if len(dr.Frames) != 1 {
		t.Fatalf("frames = %d", len(dr.Frames))
	}
	f := dr.Frames[0]
	t.Logf("frame: %d rows x %d cols", f.Rows(), len(f.Fields))
	if f.Rows() != 2 {
		t.Fatalf("rows = %d, want 2", f.Rows())
	}
	checks := []struct {
		idx  int
		ftyp data.FieldType
		val  any
	}{
		{0, data.FieldTypeNullableTime, nil},
		{1, data.FieldTypeNullableInt64, int64(42)},
		{2, data.FieldTypeNullableString, "hello"},
	}
	for _, c := range checks {
		if f.Fields[c.idx].Type() != c.ftyp {
			t.Errorf("col%d type = %v, want %v", c.idx, f.Fields[c.idx].Type(), c.ftyp)
		}
	}
	if v, ok := f.Fields[1].At(0).(*int64); !ok || v == nil || *v != 42 {
		t.Errorf("v[0] = %#v", f.Fields[1].At(0))
	}
	if s, ok := f.Fields[2].At(1).(*string); !ok || s == nil || *s != "world" {
		t.Errorf("s[1] = %#v", f.Fields[2].At(1))
	}
}

func frameFieldNames(f *data.Frame) []string {
	var names []string
	for _, fd := range f.Fields {
		names = append(names, fd.Name)
	}
	return names
}
