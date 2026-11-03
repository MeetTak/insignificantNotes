package frontmatter

import "testing"

func TestRoundTrip(t *testing.T) {
	m := New()
	m.Set("title", `config for vm "testPub"`)
	m.Set("folder", "aws/course")
	out := Render(m, []byte("body line\n"))
	got, body := Parse(out)
	if got.Get("title") != `config for vm "testPub"` || got.Get("folder") != "aws/course" {
		t.Errorf("meta = %v", got.vals)
	}
	if string(body) != "body line\n" {
		t.Errorf("body = %q", body)
	}
	if k := got.Keys(); len(k) != 2 || k[0] != "title" {
		t.Errorf("key order lost: %v", k)
	}
}

func TestParseVariants(t *testing.T) {
	m, body := Parse([]byte("---\ntitle: plain value\norder: 3\nq: 'it''s'\n---\n# Hi\n"))
	if m.Get("title") != "plain value" || m.Get("order") != "3" || m.Get("q") != "it's" {
		t.Errorf("meta = %v", m.vals)
	}
	if string(body) != "# Hi\n" {
		t.Errorf("body = %q", body)
	}
	m, body = Parse([]byte("no header\n"))
	if len(m.Keys()) != 0 || string(body) != "no header\n" {
		t.Error("file without header should pass through")
	}
	m, _ = Parse([]byte("\ufeff---\r\ntitle: x\r\n---\r\n"))
	if m.Get("title") != "x" {
		t.Error("BOM / CRLF not handled")
	}
}
