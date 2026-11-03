// Package site renders content/ into a static documentation site: a home
// index, one page per section, one page per note (with sidebar, table of
// contents and prev/next links) and a client-side search index.
//
// Every link is relative, so the output works on GitHub Pages under
// /<repo>/, on a custom domain, and from `notesync serve` alike.
package site

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"

	"insignificantnotes/internal/config"
	"insignificantnotes/internal/frontmatter"
)

//go:embed templates/*.html static/*
var embedded embed.FS

// Marker is written into the output dir; notesync only ever clears a
// directory that contains it.
const Marker = ".notesync-site"

type TOCEntry struct {
	Level int
	ID    string
	Text  string
}

type Note struct {
	Title        string
	Section      string // section slug
	SectionTitle string
	Group        string // sub-folder display name ("" for top level)
	URL          string // site-relative, e.g. aws/module1.html
	Source       string // content-relative markdown path
	Created      time.Time
	Modified     time.Time
	Order        int
	HTML         template.HTML
	TOC          []TOCEntry
	Text         string
	Excerpt      string
	ReadMins     int
	Prev, Next   *Note

	groupKey string
}

type Group struct {
	Name  string
	Notes []*Note
}

type Section struct {
	Slug        string
	Title       string
	Description string
	Intro       template.HTML
	Groups      []*Group
	Notes       []*Note
	Updated     time.Time
	order       int
}

type Stats struct {
	Notes    int
	Sections int
	Updated  time.Time
}

type Result struct {
	Stats
	Pages    int
	Assets   int
	Warnings []string
}

type pageData struct {
	Site        config.Site
	Root        string
	Title       string
	Description string
	Kind        string
	Sections    []*Section
	Section     *Section
	Note        *Note
	Recent      []*Note
	Stats       Stats
	Year        int
}

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(gmhtml.WithHardWraps(), gmhtml.WithUnsafe()),
)

// Build renders cfg's content dir into its output dir.
func Build(cfg *config.Config) (*Result, error) {
	return BuildTo(cfg, cfg.ContentDir(), cfg.OutputDir())
}

// BuildTo renders contentDir into outDir.
func BuildTo(cfg *config.Config, contentDir, outDir string) (*Result, error) {
	res := &Result{}
	loc := cfg.Location()
	sections, warns, err := load(cfg, contentDir)
	if err != nil {
		return nil, err
	}
	res.Warnings = warns

	var all []*Note
	for _, s := range sections {
		all = append(all, s.Notes...)
		if s.Updated.After(res.Updated) {
			res.Updated = s.Updated
		}
	}
	res.Notes, res.Sections = len(all), len(sections)

	if err := prepareOut(outDir); err != nil {
		return nil, err
	}
	tmpls, err := parseTemplates(loc)
	if err != nil {
		return nil, err
	}

	recent := append([]*Note(nil), all...)
	sort.SliceStable(recent, func(i, j int) bool { return recent[i].Modified.After(recent[j].Modified) })
	if len(recent) > 8 {
		recent = recent[:8]
	}
	base := pageData{Site: cfg.Site, Sections: sections, Stats: res.Stats, Year: time.Now().In(loc).Year()}

	write := func(tmpl, rel string, d pageData) error {
		d.Root = strings.Repeat("../", strings.Count(rel, "/"))
		if tmpl == "404" {
			d.Root = cfg.Site.BaseURL
		}
		var buf bytes.Buffer
		if err := tmpls[tmpl].ExecuteTemplate(&buf, "layout", d); err != nil {
			return fmt.Errorf("render %s: %w", rel, err)
		}
		res.Pages++
		return writeFile(filepath.Join(outDir, filepath.FromSlash(rel)), buf.Bytes())
	}

	home := base
	home.Kind, home.Recent, home.Description = "home", recent, cfg.Site.Tagline
	if err := write("index", "index.html", home); err != nil {
		return nil, err
	}
	nf := base
	nf.Kind, nf.Title = "404", "Not found"
	if err := write("404", "404.html", nf); err != nil {
		return nil, err
	}
	for _, s := range sections {
		d := base
		d.Kind, d.Title, d.Section, d.Description = "section", s.Title, s, s.Description
		if err := write("section", s.Slug+"/index.html", d); err != nil {
			return nil, err
		}
		for _, n := range s.Notes {
			d := base
			d.Kind, d.Title, d.Section, d.Note, d.Description = "note", n.Title, s, n, n.Excerpt
			if err := write("note", n.URL, d); err != nil {
				return nil, err
			}
		}
	}

	if res.Assets, err = copyAssets(contentDir, outDir); err != nil {
		return nil, err
	}
	if err := copyStatic(outDir); err != nil {
		return nil, err
	}
	if err := writeSearchIndex(outDir, all); err != nil {
		return nil, err
	}
	if err := writeFile(filepath.Join(outDir, ".nojekyll"), nil); err != nil {
		return nil, err
	}
	return res, nil
}

func load(cfg *config.Config, contentDir string) ([]*Section, []string, error) {
	var warns []string
	secs := map[string]*Section{}
	folderNames := map[string]string{} // section slug -> original folder name
	groupNames := map[string]string{}  // group key -> display name
	getSec := func(slug string) *Section {
		if s, ok := secs[slug]; ok {
			return s
		}
		s := &Section{Slug: slug}
		secs[slug] = s
		return s
	}

	err := filepath.WalkDir(contentDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == contentDir && os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != contentDir && (strings.HasPrefix(name, ".") || name == "assets") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(name) != ".md" {
			return nil
		}
		rel, _ := filepath.Rel(contentDir, p)
		rel = filepath.ToSlash(rel)
		parts := strings.Split(rel, "/")
		if len(parts) < 2 {
			warns = append(warns, fmt.Sprintf("%s ignored: notes must live inside a section folder", rel))
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		meta, body := frontmatter.Parse(data)
		sec := getSec(parts[0])
		if name == "_index.md" && len(parts) == 2 {
			h, _, _, err := render(body)
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			sec.Intro = h
			if t := meta.Get("title"); t != "" {
				folderNames[sec.Slug] = t
			}
			if dsc := meta.Get("description"); dsc != "" {
				sec.Description = dsc
			}
			return nil
		}
		if strings.HasPrefix(name, "_") || meta.Get("draft") == "true" {
			return nil
		}

		n := &Note{Section: sec.Slug, Source: rel, URL: strings.TrimSuffix(rel, ".md") + ".html"}
		n.Title = strings.TrimSpace(meta.Get("title"))
		if n.Title == "" {
			n.Title, body = leadingHeading(body)
		}
		if n.Title == "" {
			n.Title = humanize(strings.TrimSuffix(name, ".md"))
		}
		n.groupKey = strings.Join(parts[1:len(parts)-1], "/")
		if folder := strings.Split(meta.Get("folder"), "/"); meta.Get("folder") != "" {
			if _, ok := folderNames[sec.Slug]; !ok {
				folderNames[sec.Slug] = folder[0]
			}
			if n.groupKey != "" && len(folder) > 1 {
				groupNames[sec.Slug+"/"+n.groupKey] = strings.Join(folder[1:], " / ")
			}
		}
		n.Created = parseTime(meta.Get("created"))
		n.Modified = parseTime(meta.Get("modified"))
		if n.Modified.IsZero() {
			if fi, err := d.Info(); err == nil {
				n.Modified = fi.ModTime()
			}
		}
		n.Order, _ = strconv.Atoi(meta.Get("order"))
		if n.HTML, n.TOC, n.Text, err = render(body); err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		n.Excerpt = excerpt(n.Text, 170)
		n.ReadMins = len(strings.Fields(n.Text))/200 + 1
		sec.Notes = append(sec.Notes, n)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	var out []*Section
	for _, s := range secs {
		if len(s.Notes) == 0 {
			continue
		}
		meta := cfg.SectionMeta(s.Slug, folderNames[s.Slug])
		s.Title = meta.Title
		if s.Description == "" {
			s.Description = meta.Description
		}
		s.order = meta.Order
		if s.order == 0 {
			s.order = 1000 + cfg.FolderOrder(s.Slug)
		}
		sort.SliceStable(s.Notes, func(i, j int) bool { return noteLess(s.Notes[i], s.Notes[j]) })
		var g *Group
		for _, n := range s.Notes {
			n.SectionTitle = s.Title
			if dn, ok := groupNames[s.Slug+"/"+n.groupKey]; ok {
				n.Group = dn
			} else {
				n.Group = strings.ReplaceAll(n.groupKey, "/", " / ")
			}
			if g == nil || g.Name != n.Group {
				g = &Group{Name: n.Group}
				s.Groups = append(s.Groups, g)
			}
			g.Notes = append(g.Notes, n)
			if n.Modified.After(s.Updated) {
				s.Updated = n.Modified
			}
		}
		for i, n := range s.Notes {
			if i > 0 {
				n.Prev = s.Notes[i-1]
			}
			if i+1 < len(s.Notes) {
				n.Next = s.Notes[i+1]
			}
		}
		out = append(out, s)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].order != out[j].order {
			return out[i].order < out[j].order
		}
		return out[i].Slug < out[j].Slug
	})
	return out, warns, nil
}

// noteLess: top-level notes before sub-folders, then explicit `order:`, then
// natural title order (module2 before module10).
func noteLess(a, b *Note) bool {
	if a.groupKey != b.groupKey {
		if a.groupKey == "" || b.groupKey == "" {
			return a.groupKey == ""
		}
		return NaturalLess(a.groupKey, b.groupKey)
	}
	if (a.Order > 0) != (b.Order > 0) {
		return a.Order > 0
	}
	if a.Order != b.Order {
		return a.Order < b.Order
	}
	return NaturalLess(strings.ToLower(a.Title), strings.ToLower(b.Title))
}

// NaturalLess compares strings treating digit runs as numbers.
func NaturalLess(a, b string) bool {
	for a != "" && b != "" {
		ad, bd := unicode.IsDigit(rune(a[0])), unicode.IsDigit(rune(b[0]))
		if ad && bd {
			i, j := digitLen(a), digitLen(b)
			na, _ := strconv.Atoi(a[:i])
			nb, _ := strconv.Atoi(b[:j])
			if na != nb {
				return na < nb
			}
			a, b = a[i:], b[j:]
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func digitLen(s string) int {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i
}

func render(src []byte) (template.HTML, []TOCEntry, string, error) {
	doc := md.Parser().Parse(text.NewReader(src))
	var toc []TOCEntry
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Heading:
			if v.Level >= 2 && v.Level <= 3 {
				if id, ok := v.AttributeString("id"); ok {
					if b, ok := id.([]byte); ok {
						toc = append(toc, TOCEntry{Level: v.Level, ID: string(b), Text: nodeText(v, src)})
					}
				}
			}
		case *ast.Link:
			v.Destination = rewriteMDLink(v.Destination)
		}
		return ast.WalkContinue, nil
	})
	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, src, doc); err != nil {
		return "", nil, "", err
	}
	h := buf.String()
	h = strings.ReplaceAll(h, "<img ", `<img loading="lazy" `)
	h = strings.ReplaceAll(h, "<table>", `<div class="table-wrap"><table>`)
	h = strings.ReplaceAll(h, "</table>", "</table></div>")
	return template.HTML(h), toc, plainText(h), nil
}

func nodeText(n ast.Node, src []byte) string {
	var b strings.Builder
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch t := c.(type) {
			case *ast.Text:
				b.Write(t.Segment.Value(src))
			case *ast.String:
				b.Write(t.Value)
			}
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(b.String())
}

// rewriteMDLink turns links to sibling .md files into links to their pages.
func rewriteMDLink(dest []byte) []byte {
	s := string(dest)
	if strings.Contains(s, "://") || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "mailto:") {
		return dest
	}
	p, frag, _ := strings.Cut(s, "#")
	if strings.HasSuffix(p, ".md") {
		p = strings.TrimSuffix(p, ".md") + ".html"
		if frag != "" {
			p += "#" + frag
		}
		return []byte(p)
	}
	return dest
}

var (
	tagRe   = regexp.MustCompile(`<[^>]+>`)
	spaceRe = regexp.MustCompile(`\s+`)
	h1Re    = regexp.MustCompile(`^#\s+(.+?)\s*#*\s*(\n|$)`)
)

func plainText(h string) string {
	t := tagRe.ReplaceAllString(h, " ")
	return strings.TrimSpace(spaceRe.ReplaceAllString(html.UnescapeString(t), " "))
}

func excerpt(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	cut := string(r[:n])
	if i := strings.LastIndex(cut, " "); i > n/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:") + "…"
}

func leadingHeading(body []byte) (string, []byte) {
	trimmed := bytes.TrimLeft(body, "\n")
	m := h1Re.FindSubmatchIndex(trimmed)
	if m == nil {
		return "", body
	}
	return string(trimmed[m[2]:m[3]]), trimmed[m[1]:]
}

func humanize(s string) string {
	s = strings.NewReplacer("-", " ", "_", " ").Replace(s)
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func parseTemplates(loc *time.Location) (map[string]*template.Template, error) {
	funcs := template.FuncMap{
		"date": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.In(loc).Format("Jan 2, 2006")
		},
		"isodate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.In(loc).Format("2006-01-02")
		},
	}
	out := map[string]*template.Template{}
	for _, name := range []string{"index", "section", "note", "404"} {
		t, err := template.New(name).Funcs(funcs).ParseFS(embedded, "templates/layout.html", "templates/"+name+".html")
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}
		out[name] = t
	}
	return out, nil
}

// prepareOut empties outDir, but only if notesync created it.
func prepareOut(outDir string) error {
	if fi, err := os.Stat(outDir); err == nil {
		if !fi.IsDir() {
			return fmt.Errorf("output path %s exists and is not a directory", outDir)
		}
		if _, err := os.Stat(filepath.Join(outDir, Marker)); err != nil {
			entries, _ := os.ReadDir(outDir)
			if len(entries) > 0 {
				return fmt.Errorf("refusing to clear %s: it was not created by notesync (no %s marker)", outDir, Marker)
			}
		}
		if err := os.RemoveAll(outDir); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, Marker), []byte("generated by notesync; safe to delete\n"), 0o644)
}

func writeFile(p string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// copyAssets mirrors every non-Markdown file from content into the site, so
// relative image links in notes keep working.
func copyAssets(contentDir, outDir string) (int, error) {
	n := 0
	err := filepath.WalkDir(contentDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == contentDir && os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() {
			if p != contentDir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) == ".md" || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		rel, _ := filepath.Rel(contentDir, p)
		if err := copyFile(p, filepath.Join(outDir, rel)); err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func copyStatic(outDir string) error {
	return fs.WalkDir(embedded, "static", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := embedded.ReadFile(p)
		if err != nil {
			return err
		}
		return writeFile(filepath.Join(outDir, "static", path.Base(p)), data)
	})
}

type searchDoc struct {
	T string   `json:"t"`
	S string   `json:"s"`
	G string   `json:"g,omitempty"`
	U string   `json:"u"`
	H []string `json:"h,omitempty"`
	X string   `json:"x"`
}

func writeSearchIndex(outDir string, notes []*Note) error {
	docs := make([]searchDoc, 0, len(notes))
	for _, n := range notes {
		var hs []string
		for _, t := range n.TOC {
			hs = append(hs, t.Text)
		}
		x := []rune(n.Text)
		if len(x) > 6000 {
			x = x[:6000]
		}
		docs = append(docs, searchDoc{T: n.Title, S: n.SectionTitle, G: n.Group, U: n.URL, H: hs, X: string(x)})
	}
	data, err := json.Marshal(docs)
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(outDir, "search-index.json"), data)
}
