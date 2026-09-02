package pkg

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/mehrabr/duckcall/codec"
)

func TestIsReadOnlySQL(t *testing.T) {
	allowed := []string{
		"SELECT count(*) FROM uat_monitor_tracing",
		"SELECT * FROM t WHERE uri LIKE '%update%' LIMIT 10",
		"SELECT 'drop table x' AS msg",
		"WITH c AS (SELECT 1) SELECT * FROM c",
		"SELECT started_at, optionname FROM prd_monitor_tracing WHERE started_at > now() - INTERVAL 1 HOUR",
	}
	for _, sql := range allowed {
		if !IsReadOnlySQL(sql) {
			t.Errorf("应放行: %q", sql)
		}
	}
	blocked := []string{
		"INSERT INTO t VALUES (1)",
		"DROP TABLE t",
		"ATTACH 'quack:x' AS y",
		"SELECT 1; DROP TABLE t",
		"CALL quack_serve('x')",
		"SET memory_limit='10GB'",
	}
	for _, sql := range blocked {
		if IsReadOnlySQL(sql) {
			t.Errorf("应拦截: %q", sql)
		}
	}
}

func TestConvertColumn(t *testing.T) {
	// 数值列
	nums := convertColumn([]any{int64(1), nil, int64(3)}, LogicalType{ID: codec.TypeBigint})
	n, ok := nums.([]*int64)
	if !ok || len(n) != 3 || n[0] == nil || *n[0] != 1 || n[1] != nil {
		t.Errorf("int col wrong: %#v", nums)
	}
	// 时间列
	ts := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	tcol := convertColumn([]any{ts, nil}, LogicalType{ID: codec.TypeTimestampTZ})
	tv, ok := tcol.([]*time.Time)
	if !ok || len(tv) != 2 || tv[0] == nil || !tv[0].Equal(ts) || tv[1] != nil {
		t.Errorf("time col wrong: %#v", tcol)
	}
	// BLOB → base64
	bcol := convertColumn([]any{[]byte{0xDE, 0xAD}}, LogicalType{ID: codec.TypeBlob})
	bv, ok := bcol.([]*string)
	if !ok || bv[0] == nil || *bv[0] != "3q0=" {
		t.Errorf("blob col wrong: %#v", bcol)
	}
	// 字符串
	scol := convertColumn([]any{"a", nil}, LogicalType{ID: codec.TypeVarchar})
	sv, ok := scol.([]*string)
	if !ok || sv[0] == nil || *sv[0] != "a" {
		t.Errorf("varchar col wrong: %#v", scol)
	}
}

func TestParseQuery(t *testing.T) {
	q := Query{}
	if err := q.parse([]byte(`{"sql":"SELECT 1"}`)); err != nil {
		t.Fatal(err)
	}
	if q.SQL != "SELECT 1" {
		t.Errorf("sql = %q", q.SQL)
	}
	if err := q.parse([]byte(`{}`)); err == nil {
		t.Error("empty sql 应报错")
	}
	if err := q.parse(nil); err == nil {
		t.Error("nil 应报错")
	}
}

// TestQueryData 构造一个只读查询请求，验证走通 QueryData（无真实服务器时
// runQuery 会因 dial 失败返回错误，但框架路径已覆盖）。
func TestQueryData(t *testing.T) {
	ds := &DuckDBDatasource{cfg: Settings{Endpoint: "127.0.0.1:1", Token: "x", QueryTimeoutMS: 2000}}
	_, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{{RefID: "A", JSON: []byte(`{"sql":"SELECT count(*) FROM t"}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}
}
