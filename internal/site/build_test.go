package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"insignificantnotes/internal/config"
)

func write(t *testing.T, root, rel, data string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) (*config.Config, string, string) {
	content := t.TempDir()
	out := filepath.Join(t.TempDir(), "site")
	fm := func(title, folder string) string {
		return "---\ntitle: \"" + title + "\"\nfolder: \"" + folder + "\"\nmodified: \"2025-12-26T10:14:44.000Z\"\nsource: \"apple-notes\"\n---\n\n"
	}
	write(t, content, "aws/module1.md", fm("module1", "aws")+"one\n")
	write(t, content, "aws/module10.md", fm("module10", "aws")+"ten\n")
	write(t, content, "aws/module2.md", fm("module2", "aws")+"two\n![](assets/0123456789abcdef.png)\n")
	write(t, content, "aws/assets/0123456789abcdef.png", "png")
	write(t, content, "aws/course/domain1.md", fm("Domain1", "aws/course")+"![](../assets/0123456789abcdef.png)\n")
	write(t, content, "java/oops.md", fm("oops", "Java")+"intro\n\n## Classes\n\n```\nclass A {}\n```\n\n### Inner\n\nSee [threads](threads.md).\n")
	write(t, content, "java/threads.md", "# Threads by hand\n\nhand-written note\n")
	write(t, content, "java/_index.md", "Section intro text.\n")
	cfg := &config.Config{Site: config.Site{Title: "Test Notes", BaseURL: "/repo/"},
		Source:   config.Source{Folders: []string{"aws", "Java"}},
		Sections: map[string]config.Section{"aws": {Title: "AWS", Description: "cloud stuff"}}}
	cfg.ApplyDefaults()
	return cfg, content, out
}

func TestBuild(t *testing.T) {
	cfg, content, out := fixture(t)
	res, err := BuildTo(cfg, content, out)
	if err != nil {
		t.Fatal(err)
	}
	if res.Notes != 6 || res.Sections != 2 {
		t.Fatalf("stats = %+v", res.Stats)
	}
	for _, f := range []string{"index.html", "404.html", ".nojekyll", "search-index.json", "static/style.css", "static/app.js",
		"aws/index.html", "aws/module1.html", "aws/course/domain1.html", "java/oops.html", "aws/assets/0123456789abcdef.png"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(out, rel))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	sec := read("aws/index.html")
	i1, i2, i10 := strings.Index(sec, "module1<"), strings.Index(sec, "module2<"), strings.Index(sec, "module10<")
	if !(i1 >= 0 && i1 < i2 && i2 < i10) {
		t.Errorf("natural sort broken: %d %d %d", i1, i2, i10)
	}
	if !strings.Contains(sec, "cloud stuff") {
		t.Error("section description missing")
	}

	deep := read("aws/course/domain1.html")
	if !strings.Contains(deep, `href="../../static/style.css"`) {
		t.Error("relative root wrong for nested page")
	}
	if !strings.Contains(deep, `src="../assets/0123456789abcdef.png"`) {
		t.Error("nested image path not preserved")
	}

	oops := read("java/oops.html")
	for _, want := range []string{`id="classes"`, `href="#classes"`, `href="#inner"`, `href="threads.html"`, "<pre><code>", "Next"} {
		if !strings.Contains(oops, want) {
			t.Errorf("oops.html missing %q", want)
		}
	}
	if !strings.Contains(read("java/threads.html"), "Threads by hand") {
		t.Error("title from leading # heading not used")
	}
	if !strings.Contains(read("java/index.html"), "Section intro text.") {
		t.Error("_index.md intro not rendered")
	}
	if !strings.Contains(read("404.html"), `href="/repo/static/style.css"`) {
		t.Error("404 page must use absolute base_url")
	}
	home := read("index.html")
	if strings.Index(home, ">AWS<") > strings.Index(home, ">Java<") {
		t.Error("sections should follow source.folders order")
	}

	var idx []map[string]any
	if err := json.Unmarshal([]byte(read("search-index.json")), &idx); err != nil || len(idx) != 6 {
		t.Errorf("search index: %v (%d docs)", err, len(idx))
	}
}

func TestBuildRefusesForeignOutputDir(t *testing.T) {
	cfg, content, _ := fixture(t)
	out := t.TempDir()
	write(t, out, "important.txt", "do not delete")
	if _, err := BuildTo(cfg, content, out); err == nil {
		t.Fatal("must refuse to clear a directory it did not create")
	}
	if _, err := os.Stat(filepath.Join(out, "important.txt")); err != nil {
		t.Fatal("foreign file deleted")
	}
}

func TestNaturalLess(t *testing.T) {
	if !NaturalLess("module2", "module10") || NaturalLess("module10", "module2") || !NaturalLess("a", "b") {
		t.Error("NaturalLess")
	}
}
