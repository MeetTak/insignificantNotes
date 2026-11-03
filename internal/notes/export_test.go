package notes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"insignificantnotes/internal/config"
	"insignificantnotes/internal/frontmatter"
)

// fakeSource stands in for Apple Notes.
type fakeSource struct {
	list    Listing
	bodies  map[string]string
	fetched []string
}

func (f *fakeSource) List(context.Context) (*Listing, error) { return &f.list, nil }
func (f *fakeSource) Body(_ context.Context, id string) (string, error) {
	f.fetched = append(f.fetched, id)
	return f.bodies[id], nil
}

func newCfg(t *testing.T) *config.Config {
	c := &config.Config{Root: t.TempDir(), Source: config.Source{
		Folders:        []string{"aws", "Java"},
		ExcludeFolders: []string{"other", "others", "notes", "acad"},
		SkipTags:       []string{"#private"},
	}}
	c.ApplyDefaults()
	return c
}

func note(id, name string, folder []string, modified string) NoteMeta {
	return NoteMeta{ID: id, Name: name, Folder: folder, Created: "2025-01-0" + id + "T00:00:00Z", Modified: modified}
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestExportLifecycle(t *testing.T) {
	cfg := newCfg(t)
	src := &fakeSource{
		list: Listing{Found: []string{"aws", "Java"}, Notes: []NoteMeta{
			note("1", "module1", []string{"aws"}, "m1"),
			note("2", "Domain1", []string{"aws", "course"}, "m1"),
			note("3", "oops", []string{"Java"}, "m1"),
			note("4", "leak", []string{"aws", "others"}, "m1"), // excluded subfolder
			note("5", "school", []string{"acad"}, "m1"),        // excluded top level
			note("6", "secret plans", []string{"Java"}, "m1"),  // tagged private
		}},
		bodies: map[string]string{
			"1": `<div><h1>module1</h1></div><div>hello <b>aws</b></div><div><img src="data:image/png;base64,iVBORw0KGgo="></div>`,
			"2": `<div><h1>Domain1</h1></div><div>domain text</div><div><img src="data:image/png;base64,iVBORw0KGgo="></div>`,
			"3": `<div><h1>oops</h1></div><div>classes</div>`,
			"6": `<div><h1>secret plans</h1></div><div>#private</div>`,
		},
	}
	ex := &Exporter{Cfg: cfg, Source: src}
	rep, err := ex.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Count("add") != 3 {
		t.Fatalf("want 3 adds, got %+v", rep.Changes)
	}
	for _, id := range src.fetched {
		if id == "4" || id == "5" {
			t.Fatalf("note %s from an excluded folder was read", id)
		}
	}
	content := cfg.ContentDir()
	m1 := read(t, filepath.Join(content, "aws", "module1.md"))
	meta, body := frontmatter.Parse([]byte(m1))
	if meta.Get("note_id") != "1" || meta.Get("source") != SourceTag || meta.Get("folder") != "aws" {
		t.Errorf("front matter = %v", meta.Keys())
	}
	if !strings.Contains(string(body), "hello **aws**") || !strings.Contains(string(body), "](assets/") {
		t.Errorf("body = %q", body)
	}
	d1 := read(t, filepath.Join(content, "aws", "course", "domain1.md"))
	if !strings.Contains(d1, "](../assets/") {
		t.Errorf("sub-folder note should reference ../assets/: %q", d1)
	}
	if _, err := os.Stat(filepath.Join(content, "java", "secret-plans.md")); err == nil {
		t.Error("#private note was exported")
	}
	assets, _ := os.ReadDir(filepath.Join(content, "aws", "assets"))
	if len(assets) != 1 {
		t.Errorf("identical images should share one asset, got %d", len(assets))
	}

	// Second run with no changes fetches nothing.
	src.fetched = nil
	rep, err = ex.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(src.fetched) != 0 || rep.Unchanged != 3 {
		t.Errorf("incremental run fetched %v, unchanged=%d", src.fetched, rep.Unchanged)
	}

	// Rename note 3, delete note 2, edit note 1 (dropping its image).
	src.list.Notes = []NoteMeta{
		note("1", "module1", []string{"aws"}, "m2"),
		note("3", "OOP basics", []string{"Java"}, "m2"),
	}
	src.bodies["1"] = `<div><h1>module1</h1></div><div>no image now</div>`
	src.bodies["3"] = `<div><h1>OOP basics</h1></div><div>classes</div>`
	rep, err = ex.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Count("rename") != 1 || rep.Count("remove") != 1 || rep.Count("update") != 1 {
		t.Errorf("changes = %+v", rep.Changes)
	}
	if _, err := os.Stat(filepath.Join(content, "java", "oops.md")); err == nil {
		t.Error("old file not removed after rename")
	}
	if _, err := os.Stat(filepath.Join(content, "java", "oop-basics.md")); err != nil {
		t.Error("renamed file missing")
	}
	if _, err := os.Stat(filepath.Join(content, "aws", "assets")); err == nil {
		t.Error("unreferenced assets (and the empty dir) should be cleaned up")
	}
}

func TestExportRefusesMassDelete(t *testing.T) {
	cfg := newCfg(t)
	p := filepath.Join(cfg.ContentDir(), "aws", "x.md")
	os.MkdirAll(filepath.Dir(p), 0o755)
	m := frontmatter.New()
	m.Set("note_id", "9")
	m.Set("source", SourceTag)
	os.WriteFile(p, frontmatter.Render(m, nil), 0o644)
	ex := &Exporter{Cfg: cfg, Source: &fakeSource{}}
	if _, err := ex.Run(context.Background()); err == nil {
		t.Fatal("an empty listing must not delete existing notes")
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal("note was deleted")
	}
}

func TestHandWrittenFilesAreKept(t *testing.T) {
	cfg := newCfg(t)
	p := filepath.Join(cfg.ContentDir(), "aws", "_index.md")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte("intro\n"), 0o644)
	src := &fakeSource{list: Listing{Found: []string{"aws"}, Notes: []NoteMeta{note("1", "a", []string{"aws"}, "m")}},
		bodies: map[string]string{"1": "<div>x</div>"}}
	if _, err := (&Exporter{Cfg: cfg, Source: src}).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Error("hand-written file was removed")
	}
}

func TestPlanPathsAreUnique(t *testing.T) {
	ps := planPaths([]NoteMeta{
		note("1", "Notes", []string{"aws"}, ""),
		note("2", "notes", []string{"aws"}, ""),
		note("3", "notes", []string{"aws", "course"}, ""),
	})
	got := map[string]string{}
	for _, p := range ps {
		got[p.note.ID] = filepath.ToSlash(p.rel)
	}
	if got["1"] != "aws/notes.md" || got["2"] != "aws/notes-2.md" || got["3"] != "aws/course/notes.md" {
		t.Errorf("paths = %v", got)
	}
}

func TestSkipTag(t *testing.T) {
	tags := []string{"#private", "#nopublish"}
	cases := map[string]string{
		"<div>#private</div>":              "#private",
		"<div>draft #NoPublish here</div>": "#nopublish",
		"<div>#privateer ship</div>":       "",
		"<div>nothing</div>":               "",
	}
	for body, want := range cases {
		if got := skipTag(body, tags); got != want {
			t.Errorf("skipTag(%q) = %q, want %q", body, got, want)
		}
	}
}
