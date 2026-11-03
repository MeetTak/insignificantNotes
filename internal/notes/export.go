package notes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"insignificantnotes/internal/config"
	"insignificantnotes/internal/convert"
	"insignificantnotes/internal/frontmatter"
)

// SourceTag marks files the exporter owns. Files without it are hand-written
// and never touched or deleted by export.
const SourceTag = "apple-notes"

type Change struct {
	Action string // add, update, rename, remove, skip
	Title  string
	Path   string // content-relative
	Detail string
}

type Report struct {
	Changes   []Change
	Unchanged int
	Warnings  []string
}

func (r *Report) add(action, title, path, detail string) {
	r.Changes = append(r.Changes, Change{action, title, path, detail})
}

// Count returns how many changes have the given action.
func (r *Report) Count(action string) int {
	n := 0
	for _, c := range r.Changes {
		if c.Action == action {
			n++
		}
	}
	return n
}

// Written reports whether export changed anything on disk.
func (r *Report) Written() bool {
	for _, c := range r.Changes {
		if c.Action != "skip" {
			return true
		}
	}
	return false
}

type Exporter struct {
	Cfg    *config.Config
	Source Source
	Force  bool // re-export unchanged notes too
	DryRun bool
	// Progress, when set, is called before each note body is fetched.
	Progress func(done, total int, title string)
}

type existingNote struct {
	Rel      string
	Modified string
}

type plan struct {
	note NoteMeta
	rel  string
}

// Run performs one sync from Apple Notes into the content directory.
func (e *Exporter) Run(ctx context.Context) (*Report, error) {
	rep := &Report{}
	contentDir := e.Cfg.ContentDir()

	list, err := e.Source.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, f := range e.Cfg.Source.Folders {
		if !containsNorm(list.Found, f) {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("folder %q was not found in Apple Notes", f))
		}
	}
	for _, s := range list.Skipped {
		rep.add("skip", s.Path, "", s.Reason)
	}

	existing, err := indexExisting(contentDir)
	if err != nil {
		return nil, err
	}

	// Defence in depth: drop anything under an excluded folder even if the
	// script returned it.
	var notes []NoteMeta
	for _, n := range list.Notes {
		if len(n.Folder) == 0 || !e.Cfg.IsIncluded(n.Folder[0]) || anyExcluded(e.Cfg, n.Folder) {
			continue
		}
		notes = append(notes, n)
	}
	if len(notes) == 0 && len(existing) > 0 && !e.Force {
		return nil, fmt.Errorf("Apple Notes returned no notes for %v but %d exported notes exist; refusing to delete them (re-run with --force if this is intended)", e.Cfg.Source.Folders, len(existing))
	}

	cache := loadCache(e.Cfg.CachePath())
	newCache := map[string]string{}
	plans := planPaths(notes)
	keep := map[string]bool{}
	var todo []plan
	for _, p := range plans {
		if p.note.Locked {
			rep.add("skip", p.note.Name, p.rel, "locked note")
			continue
		}
		if tag, ok := cache[p.note.ID]; ok && !e.Force && strings.HasPrefix(tag, p.note.Modified+"|") {
			newCache[p.note.ID] = tag
			rep.add("skip", p.note.Name, p.rel, "tagged "+strings.SplitN(tag, "|", 2)[1])
			continue
		}
		ex, had := existing[p.note.ID]
		if had && !e.Force && ex.Modified == p.note.Modified && ex.Rel == p.rel {
			rep.Unchanged++
			keep[p.note.ID] = true
			continue
		}
		todo = append(todo, p)
	}

	for i, p := range todo {
		n := p.note
		ex, had := existing[n.ID]
		action := "add"
		if had {
			action = "update"
			if ex.Rel != p.rel {
				action = "rename"
			}
		}
		if e.DryRun {
			keep[n.ID] = true
			rep.add(action, n.Name, p.rel, "")
			continue
		}
		if e.Progress != nil {
			e.Progress(i+1, len(todo), n.Name)
		}
		body, err := e.Source.Body(ctx, n.ID)
		if err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: could not read note body: %v", n.Name, err))
			keep[n.ID] = had
			continue
		}
		if tag := skipTag(body, e.Cfg.Source.SkipTags); tag != "" {
			newCache[n.ID] = n.Modified + "|" + tag
			rep.add("skip", n.Name, p.rel, "tagged "+tag)
			continue
		}
		if err := e.write(p, body, rep); err != nil {
			return nil, err
		}
		if had && ex.Rel != p.rel {
			os.Remove(filepath.Join(contentDir, ex.Rel))
		}
		keep[n.ID] = true
		rep.add(action, n.Name, p.rel, "")
	}

	ids := make([]string, 0, len(existing))
	for id := range existing {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if keep[id] {
			continue
		}
		ex := existing[id]
		if !e.DryRun {
			if err := os.Remove(filepath.Join(contentDir, ex.Rel)); err != nil && !os.IsNotExist(err) {
				return nil, err
			}
		}
		rep.add("remove", strings.TrimSuffix(filepath.Base(ex.Rel), ".md"), ex.Rel, "deleted, moved out of a published folder, locked or tagged private")
	}

	if !e.DryRun {
		if err := saveCache(e.Cfg.CachePath(), newCache); err != nil {
			rep.Warnings = append(rep.Warnings, "could not save export cache: "+err.Error())
		}
		if _, err := CleanupAssets(contentDir); err != nil {
			return nil, err
		}
		removeEmptyDirs(contentDir)
	}
	return rep, nil
}

func (e *Exporter) write(p plan, body string, rep *Report) error {
	contentDir := e.Cfg.ContentDir()
	abs := filepath.Join(contentDir, p.rel)
	depth := strings.Count(filepath.ToSlash(p.rel), "/") // 1 for section/note.md
	sink := &assetSink{
		dir:       filepath.Join(contentDir, strings.SplitN(filepath.ToSlash(p.rel), "/", 2)[0], "assets"),
		relPrefix: strings.Repeat("../", depth-1),
		maxWidth:  e.Cfg.Source.ImageMaxWidth,
	}
	res, err := convert.ToMarkdown(body, convert.Options{Title: p.note.Name, Images: sink})
	if err != nil {
		return fmt.Errorf("%s: convert: %w", p.note.Name, err)
	}
	for _, w := range res.Warnings {
		rep.Warnings = append(rep.Warnings, p.note.Name+": "+w)
	}
	m := frontmatter.New()
	m.Set("title", strings.TrimSpace(p.note.Name))
	m.Set("folder", strings.Join(p.note.Folder, "/"))
	m.Set("created", p.note.Created)
	m.Set("modified", p.note.Modified)
	m.Set("note_id", p.note.ID)
	m.Set("source", SourceTag)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	return writeAtomic(abs, frontmatter.Render(m, []byte(res.Markdown)))
}

// planPaths gives every note a stable, unique content-relative path:
// <section>/<subfolder…>/<slug>.md. Collisions get -2, -3… in creation order.
func planPaths(notes []NoteMeta) []plan {
	sorted := append([]NoteMeta(nil), notes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Created != b.Created {
			return a.Created < b.Created
		}
		return a.ID < b.ID
	})
	taken := map[string]bool{}
	var out []plan
	for _, n := range sorted {
		parts := make([]string, len(n.Folder))
		for i, f := range n.Folder {
			parts[i] = config.Slugify(f)
		}
		dir := filepath.Join(parts...)
		base := config.Slugify(n.Name)
		rel := filepath.Join(dir, base+".md")
		for k := 2; taken[strings.ToLower(rel)]; k++ {
			rel = filepath.Join(dir, base+"-"+strconv.Itoa(k)+".md")
		}
		taken[strings.ToLower(rel)] = true
		out = append(out, plan{n, rel})
	}
	return out
}

func indexExisting(contentDir string) (map[string]existingNote, error) {
	out := map[string]existingNote{}
	err := filepath.WalkDir(contentDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && p == contentDir {
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() || filepath.Ext(p) != ".md" {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		m, _ := frontmatter.Parse(data)
		if m.Get("source") != SourceTag || m.Get("note_id") == "" {
			return nil
		}
		rel, _ := filepath.Rel(contentDir, p)
		out[m.Get("note_id")] = existingNote{Rel: rel, Modified: m.Get("modified")}
		return nil
	})
	return out, err
}

func anyExcluded(cfg *config.Config, path []string) bool {
	for _, p := range path {
		if cfg.IsExcluded(p) {
			return true
		}
	}
	return false
}

func containsNorm(list []string, s string) bool {
	for _, l := range list {
		if config.Norm(l) == config.Norm(s) {
			return true
		}
	}
	return false
}

var tagStrip = regexp.MustCompile(`<[^>]+>`)

// skipTag returns the first skip tag (e.g. "#private") present in the body.
func skipTag(body string, tags []string) string {
	text := strings.ToLower(tagStrip.ReplaceAllString(body, " "))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		re := regexp.MustCompile(`(^|[^\w#])` + regexp.QuoteMeta(t) + `($|[^\w-])`)
		if re.MatchString(text) {
			return t
		}
	}
	return ""
}

// The export cache remembers notes skipped because of a skip tag, keyed by
// note id → "<modified>|<tag>", so unchanged private notes are never re-read.
func loadCache(path string) map[string]string {
	out := map[string]string{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &out)
	}
	return out
}

func saveCache(path string, m map[string]string) error {
	if len(m) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, data)
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// assetSink writes images to <section>/assets/<sha256-16>.<ext>.
type assetSink struct {
	dir       string
	relPrefix string
	maxWidth  int
}

var mimeExt = map[string]string{
	"image/png": "png", "image/jpeg": "jpg", "image/jpg": "jpg", "image/gif": "gif",
	"image/webp": "webp", "image/svg+xml": "svg", "image/heic": "heic", "image/tiff": "tiff",
}

func (s *assetSink) SaveImage(mime string, data []byte) (string, error) {
	ext, ok := mimeExt[strings.ToLower(mime)]
	if !ok {
		return "", fmt.Errorf("unsupported image type %q", mime)
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])[:16]
	// Browsers cannot show HEIC/TIFF; convert them to JPEG/PNG with sips.
	outExt := ext
	if ext == "heic" {
		outExt = "jpg"
	} else if ext == "tiff" {
		outExt = "png"
	}
	name := hash + "." + outExt
	dst := filepath.Join(s.dir, name)
	if _, err := os.Stat(dst); err == nil {
		return s.relPrefix + "assets/" + name, nil
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", err
	}
	if outExt != ext {
		src := filepath.Join(s.dir, hash+"."+ext+".tmp")
		if err := os.WriteFile(src, data, 0o644); err != nil {
			return "", err
		}
		defer os.Remove(src)
		format := map[string]string{"jpg": "jpeg", "png": "png"}[outExt]
		if err := exec.Command("sips", "-s", "format", format, src, "--out", dst).Run(); err != nil {
			return "", fmt.Errorf("convert %s to %s with sips: %w", ext, outExt, err)
		}
	} else if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", err
	}
	if s.maxWidth > 0 && outExt != "svg" && outExt != "gif" {
		shrink(dst, s.maxWidth)
	}
	return s.relPrefix + "assets/" + name, nil
}

var widthRe = regexp.MustCompile(`pixelWidth:\s*(\d+)`)

// shrink downsizes wide images in place with macOS sips (best effort).
func shrink(path string, maxWidth int) {
	if _, err := exec.LookPath("sips"); err != nil {
		return
	}
	out, err := exec.Command("sips", "-g", "pixelWidth", path).Output()
	if err != nil {
		return
	}
	m := widthRe.FindSubmatch(out)
	if m == nil {
		return
	}
	if w, _ := strconv.Atoi(string(m[1])); w > maxWidth {
		exec.Command("sips", "--resampleWidth", strconv.Itoa(maxWidth), path).Run()
	}
}

var assetRef = regexp.MustCompile(`assets/([0-9a-f]{16}\.[a-z0-9]+)`)
var assetName = regexp.MustCompile(`^[0-9a-f]{16}\.[a-z0-9]+$`)

// CleanupAssets deletes exporter-generated images no Markdown file references.
// Only hash-named files are considered, so hand-added assets are safe.
func CleanupAssets(contentDir string) ([]string, error) {
	sections, err := os.ReadDir(contentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var removed []string
	for _, sec := range sections {
		if !sec.IsDir() {
			continue
		}
		secDir := filepath.Join(contentDir, sec.Name())
		refs := map[string]bool{}
		filepath.WalkDir(secDir, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && filepath.Ext(p) == ".md" {
				if data, err := os.ReadFile(p); err == nil {
					for _, m := range assetRef.FindAllSubmatch(data, -1) {
						refs[string(m[1])] = true
					}
				}
			}
			return nil
		})
		assets, err := os.ReadDir(filepath.Join(secDir, "assets"))
		if err != nil {
			continue
		}
		for _, a := range assets {
			if a.IsDir() || !assetName.MatchString(a.Name()) || refs[a.Name()] {
				continue
			}
			p := filepath.Join(secDir, "assets", a.Name())
			if err := os.Remove(p); err != nil {
				return removed, err
			}
			removed = append(removed, p)
		}
	}
	return removed, nil
}

func removeEmptyDirs(root string) {
	var dirs []string
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && p != root {
			dirs = append(dirs, p)
		}
		return nil
	})
	for i := len(dirs) - 1; i >= 0; i-- {
		os.Remove(dirs[i]) // fails (harmlessly) when not empty
	}
}
