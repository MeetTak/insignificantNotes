// Package tests holds the publishing gate: checks that run against the real
// content/ directory. They run in `go test ./...`, in `make check`, and in
// GitHub Actions before anything is deployed, so a leaked key, an excluded
// folder or a broken link stops the site from going live.
package tests

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"insignificantnotes/internal/config"
	"insignificantnotes/internal/frontmatter"
	"insignificantnotes/internal/scan"
	"insignificantnotes/internal/site"
)

func loadConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load("..")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// contentFiles returns repo-relative paths of every file in content/.
func contentFiles(t *testing.T, cfg *config.Config) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(cfg.ContentDir(), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(cfg.Root, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	if os.IsNotExist(err) || (err == nil && len(out) == 0) {
		t.Skip("content/ is empty; run `notesync export` first")
	}
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestNoSecretsInContent is the secret gate: HIGH findings fail the build,
// warnings are logged (run `go test ./tests -v` to see them).
func TestNoSecretsInContent(t *testing.T) {
	cfg := loadConfig(t)
	allow, err := scan.LoadAllowlist(cfg.AllowlistPath())
	if err != nil {
		t.Fatal(err)
	}
	sc := scan.New(allow, cfg.Scan.IgnorePaths, cfg.Scan.KeywordPaths)
	warnings := 0
	for _, rel := range contentFiles(t, cfg) {
		data, err := os.ReadFile(filepath.Join(cfg.Root, rel))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range sc.ScanContent(rel, data) {
			if f.Severity == scan.High {
				t.Errorf("HIGH %s:%d %s: %s (fp:%s). Remove it from the note, or allowlist the fingerprint in %s if it is a false positive",
					f.Path, f.Line, f.Description, f.Shown(), f.Fingerprint, cfg.Scan.Allowlist)
			} else {
				warnings++
				t.Logf("warning %s:%d %s: %s (fp:%s)", f.Path, f.Line, f.RuleID, f.Shown(), f.Fingerprint)
			}
		}
	}
	if warnings > 0 {
		t.Logf("%d warning(s); review them, then `notesync scan --accept-warnings`", warnings)
	}
}

// TestExcludedFoldersNeverPublished guards the privacy rule: nothing from an
// excluded Apple Notes folder may ever land in content/.
func TestExcludedFoldersNeverPublished(t *testing.T) {
	cfg := loadConfig(t)
	for _, rel := range contentFiles(t, cfg) {
		parts := strings.Split(strings.TrimPrefix(rel, cfg.Paths.Content+"/"), "/")
		for _, dir := range parts[:len(parts)-1] {
			if cfg.IsExcluded(dir) {
				t.Errorf("%s lives under excluded folder %q", rel, dir)
			}
		}
		if !strings.HasSuffix(rel, ".md") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(cfg.Root, rel))
		m, _ := frontmatter.Parse(data)
		if folder := m.Get("folder"); folder != "" {
			segs := strings.Split(folder, "/")
			if !cfg.IsIncluded(segs[0]) {
				t.Errorf("%s comes from %q, which is not in source.folders", rel, folder)
			}
			for _, s := range segs {
				if cfg.IsExcluded(s) {
					t.Errorf("%s comes from excluded folder %q", rel, folder)
				}
			}
		}
	}
}

// TestNotesAreWellFormed checks titles exist and every image reference
// points at a file that exists.
func TestNotesAreWellFormed(t *testing.T) {
	cfg := loadConfig(t)
	img := regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)\)`)
	for _, rel := range contentFiles(t, cfg) {
		if !strings.HasSuffix(rel, ".md") || strings.HasPrefix(filepath.Base(rel), "_") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(cfg.Root, rel))
		m, body := frontmatter.Parse(data)
		if m.Get("source") == "apple-notes" && strings.TrimSpace(m.Get("title")) == "" {
			t.Errorf("%s has no title", rel)
		}
		for _, match := range img.FindAllSubmatch(body, -1) {
			ref := string(match[1])
			if strings.Contains(ref, "://") {
				continue
			}
			if _, err := os.Stat(filepath.Join(cfg.Root, filepath.Dir(rel), filepath.FromSlash(ref))); err != nil {
				t.Errorf("%s references missing image %s", rel, ref)
			}
		}
	}
}

// TestSiteHasNoBrokenLinks builds the site into a temp dir and checks every
// relative href/src resolves to a generated file.
func TestSiteHasNoBrokenLinks(t *testing.T) {
	cfg := loadConfig(t)
	contentFiles(t, cfg) // skip when empty
	out := filepath.Join(t.TempDir(), "site")
	if _, err := site.BuildTo(cfg, cfg.ContentDir(), out); err != nil {
		t.Fatal(err)
	}
	link := regexp.MustCompile(`(?:href|src)="([^"]+)"`)
	filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".html" || filepath.Base(p) == "404.html" {
			return err
		}
		data, _ := os.ReadFile(p)
		for _, m := range link.FindAllStringSubmatch(string(data), -1) {
			ref := m[1]
			if strings.Contains(ref, "://") || strings.HasPrefix(ref, "#") || strings.HasPrefix(ref, "mailto:") ||
				strings.HasPrefix(ref, "data:") || strings.HasPrefix(ref, "/") {
				continue
			}
			ref = strings.SplitN(strings.SplitN(ref, "#", 2)[0], "?", 2)[0]
			if _, err := os.Stat(filepath.Join(filepath.Dir(p), filepath.FromSlash(ref))); err != nil {
				rel, _ := filepath.Rel(out, p)
				t.Errorf("%s: broken link %s", rel, m[1])
			}
		}
		return nil
	})
}
