package pkg

import "strings"

// 只读白名单：与 duckdb-gateway 相同的强约束，插件面板 SQL 只允许只读语句。
var readOnlyPrefixes = []string{
	"select", "show", "describe", "explain", "with",
	"from", "values", "pragma", "summarize",
}

var writeKeywords = []string{
	"insert", "update", "delete", "drop", "alter", "create",
	"attach", "detach", "call", "copy", "install", "load",
	"set", "vacuum", "checkpoint", "export", "import",
	"use", "grant", "revoke", "merge", "truncate",
}

// IsReadOnlySQL 只读白名单校验（拒绝多语句与一切写操作）。
// 先剥离注释与字符串字面量（'...' / "..." / $tag$...$tag$），避免 LIKE '%update%'
// 这类数据值误伤，再检查语句前缀与危险关键词。
func IsReadOnlySQL(sql string) bool {
	clean := strings.TrimSpace(strings.ToLower(stripQuotedAndComments(sql)))
	if clean == "" {
		return false
	}
	if strings.Contains(clean, ";") {
		return false // 禁止多语句
	}
	okPrefix := false
	for _, p := range readOnlyPrefixes {
		if clean == p || strings.HasPrefix(clean, p+" ") || strings.HasPrefix(clean, p+"\n") || strings.HasPrefix(clean, p+"\t") {
			okPrefix = true
			break
		}
	}
	if !okPrefix {
		return false
	}
	for _, kw := range writeKeywords {
		if hasWord(clean, kw) {
			return false
		}
	}
	return true
}

// stripQuotedAndComments 把注释、字符串/标识符字面量替换为空格，保留结构位置。
func stripQuotedAndComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case i+1 < len(s) && s[i:i+2] == "--": // 行注释
			for i < len(s) && s[i] != '\n' {
				i++
			}
			b.WriteByte(' ')
		case i+1 < len(s) && s[i:i+2] == "/*": // 块注释
			depth := 1
			i += 2
			for i < len(s) && depth > 0 {
				if i+1 < len(s) && s[i:i+2] == "/*" {
					depth++
					i += 2
					continue
				}
				if i+1 < len(s) && s[i:i+2] == "*/" {
					depth--
					i += 2
					continue
				}
				i++
			}
			b.WriteByte(' ')
		case c == '\'': // 单引号字符串（'' 转义）
			i++
			for i < len(s) {
				if s[i] == '\'' {
					if i+1 < len(s) && s[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
			b.WriteByte(' ')
		case c == '"': // 双引号标识符
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '"' && i+1 < len(s) && s[i+1] == '"' {
					i += 2
					continue
				}
				i++
			}
			i++
			b.WriteByte(' ')
		case c == '$': // 美元引号 $tag$...$tag$
			j := strings.IndexByte(s[i+1:], '$')
			if j >= 0 {
				tag := s[i+1 : i+1+j]
				ok := tag == "" // $$ 无标签
				for k := 0; k < len(tag); k++ {
					t := tag[k]
					if t == '_' || (t >= 'a' && t <= 'z') || (t >= 'A' && t <= 'Z') || (t >= '0' && t <= '9') {
						ok = true
						continue
					}
					ok = false
					break
				}
				if ok {
					close := "$" + tag + "$"
					rest := s[i+len(close):]
					if idx := strings.Index(rest, close); idx >= 0 {
						i += len(close) + idx + len(close)
						b.WriteByte(' ')
						continue
					}
				}
			}
			b.WriteByte(c)
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// hasWord 词边界匹配关键字（防止误伤 offset/setup/updated_at 等）。
func hasWord(s, kw string) bool {
	for i := 0; ; {
		j := strings.Index(s[i:], kw)
		if j < 0 {
			return false
		}
		pos := i + j
		before := pos == 0 || !isWordChar(s[pos-1])
		after := pos+len(kw) >= len(s) || !isWordChar(s[pos+len(kw)])
		if before && after {
			return true
		}
		i = pos + len(kw)
	}
}

func isWordChar(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}
