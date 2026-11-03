// Package frontmatter reads and writes the small "key: value" header at the top
// of every Markdown note. Values are written as double-quoted strings, which is
// valid YAML, so the files stay readable by other static-site tools too.
package frontmatter

import (
	"bytes"
	"strconv"
	"strings"
)

type Meta struct {
	keys []string
	vals map[string]string
}

func New() *Meta { return &Meta{vals: map[string]string{}} }

func (m *Meta) Set(k, v string) {
	if _, ok := m.vals[k]; !ok {
		m.keys = append(m.keys, k)
	}
	m.vals[k] = v
}

func (m *Meta) Get(k string) string { return m.vals[k] }
func (m *Meta) Has(k string) bool   { _, ok := m.vals[k]; return ok }
func (m *Meta) Keys() []string      { return append([]string(nil), m.keys...) }

// Parse splits data into its header and body. Files without a header return an
// empty Meta and the original data.
func Parse(data []byte) (*Meta, []byte) {
	m := New()
	s := strings.TrimPrefix(string(data), "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return m, data
	}
	rest := s[4:]
	end := -1
	if strings.HasPrefix(rest, "---\n") || rest == "---" {
		end = 0
	} else if i := strings.Index(rest, "\n---\n"); i >= 0 {
		end = i + 1
	} else if strings.HasSuffix(rest, "\n---") {
		end = len(rest) - 3
	}
	if end < 0 {
		return m, data
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(k) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		m.Set(strings.TrimSpace(k), unquote(strings.TrimSpace(v)))
	}
	body := rest[end:]
	body = strings.TrimPrefix(body, "---")
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimLeft(body, "\n")
	return m, []byte(body)
}

// Render writes a header followed by body.
func Render(m *Meta, body []byte) []byte {
	var b bytes.Buffer
	b.WriteString("---\n")
	for _, k := range m.keys {
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(strconv.Quote(m.vals[k]))
		b.WriteByte('\n')
	}
	b.WriteString("---\n\n")
	b.Write(body)
	return b.Bytes()
}

func unquote(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		if u, err := strconv.Unquote(v); err == nil {
			return u
		}
		return v[1 : len(v)-1]
	}
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		return strings.ReplaceAll(v[1:len(v)-1], "''", "'")
	}
	return v
}
