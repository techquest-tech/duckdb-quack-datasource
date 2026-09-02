package plugin

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/mehrabr/duckcall/codec"
)

// convertColumn 把解码后的原始值列按 DuckDB 类型转换为 Grafana 可用的
// typed 切片（可空指针切片，nil = SQL NULL）。
func convertColumn(vals []any, lt LogicalType) any {
	switch lt.ID {
	case codec.TypeBoolean:
		out := make([]*bool, len(vals))
		for i, v := range vals {
			if v != nil {
				if b, ok := v.(bool); ok {
					out[i] = &b
				}
			}
		}
		return out
	case codec.TypeTinyint, codec.TypeSmallint, codec.TypeInteger, codec.TypeBigint,
		codec.TypeUTinyint, codec.TypeUSmallint, codec.TypeUInteger, codec.TypeUBigint:
		out := make([]*int64, len(vals))
		for i, v := range vals {
			if v != nil {
				switch n := v.(type) {
				case int64:
					out[i] = &n
				case int32:
					x := int64(n)
					out[i] = &x
				case int:
					x := int64(n)
					out[i] = &x
				case uint64:
					x := int64(n)
					out[i] = &x
				}
			}
		}
		return out
	case codec.TypeFloat, codec.TypeDouble:
		out := make([]*float64, len(vals))
		for i, v := range vals {
			if v != nil {
				switch n := v.(type) {
				case float64:
					out[i] = &n
				case float32:
					x := float64(n)
					out[i] = &x
				case int64:
					x := float64(n)
					out[i] = &x
				}
			}
		}
		return out
	case codec.TypeTimestampSec, codec.TypeTimestampMS, codec.TypeTimestamp,
		codec.TypeTimestampNS, codec.TypeTimestampTZ, codec.TypeDate:
		out := make([]*time.Time, len(vals))
		for i, v := range vals {
			if t, ok := v.(time.Time); ok {
				tt := t
				out[i] = &tt
			}
		}
		return out
	case codec.TypeBlob:
		out := make([]*string, len(vals))
		for i, v := range vals {
			if b, ok := v.([]byte); ok {
				s := base64.StdEncoding.EncodeToString(b)
				out[i] = &s
			}
		}
		return out
	default: // varchar / 其他（interval/uuid/嵌套等转字符串）
		out := make([]*string, len(vals))
		for i, v := range vals {
			if v != nil {
				s := fmt.Sprint(v)
				out[i] = &s
			}
		}
		return out
	}
}

// LogicalType 别名，避免 datasource.go 直接依赖 codec 包类型签名（测试友好）。
type LogicalType = codec.LogicalType
