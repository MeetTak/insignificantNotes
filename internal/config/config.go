// Package config loads notesync.json, the single source of truth for which
// Apple Notes folders are published, how the site looks and how the secret
// scanner behaves.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// FileName is the config file notesync looks for, walking up from the cwd.
const FileName = "notesync.json"

// alwaysExcluded folders are never read, whatever the config says.
var alwaysExcluded = []string{"recently deleted"}

type Config struct {
	Site     Site               `json:"site"`
	Source   Source             `json:"source"`
	Sections map[string]Section `json:"sections"`
	Paths    Paths              `json:"paths"`
	Scan     Scan               `json:"scan"`

	// Root is the absolute directory that contains notesync.json.
	Root string `json:"-"`
}

type Site struct {
	Title    string `json:"title"`
	Tagline  string `json:"tagline"`
	Author   string `json:"author"`
	RepoURL  string `json:"repo_url"`
	BaseURL  string `json:"base_url"`
	Timezone string `json:"timezone"`
}

type Source struct {
	Account        string   `json:"account"`
	Folders        []string `json:"folders"`
	ExcludeFolders []string `json:"exclude_folders"`
	SkipTags       []string `json:"skip_tags"`
	ImageMaxWidth  int      `json:"image_max_width"`
}

type Section struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Order       int    `json:"order"`
}

type Paths struct {
	Content string `json:"content"`
	Output  string `json:"output"`
}

type Scan struct {
	Allowlist      string   `json:"allowlist"`
	IgnorePaths    []string `json:"ignore_paths"`
	KeywordPaths   []string `json:"keyword_paths"`
	FailOnWarnings bool     `json:"fail_on_warnings"`
}

// Norm normalises a folder name for comparisons: trimmed and lower-cased, so
// "Others " and "others" are the same folder.
func Norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// Find walks upward from start until it finds notesync.json.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(dir, FileName)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%s not found (searched upward from %s)", FileName, start)
		}
		dir = parent
	}
}

// Load finds and loads the config that governs start.
func Load(start string) (*Config, error) {
	p, err := Find(start)
	if err != nil {
		return nil, err
	}
	return LoadFile(p)
}

// LoadFile parses one config file. Unknown keys are rejected so typos surface.
func LoadFile(p string) (*Config, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c Config
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	abs, err := filepath.Abs(filepath.Dir(p))
	if err != nil {
		return nil, err
	}
	c.Root = abs
	c.ApplyDefaults()
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	return &c, nil
}

// ApplyDefaults fills empty optional fields.
func (c *Config) ApplyDefaults() {
	if c.Site.Title == "" {
		c.Site.Title = "Notes"
	}
	if c.Site.BaseURL == "" {
		c.Site.BaseURL = "/"
	}
	if !strings.HasSuffix(c.Site.BaseURL, "/") {
		c.Site.BaseURL += "/"
	}
	if c.Paths.Content == "" {
		c.Paths.Content = "content"
	}
	if c.Paths.Output == "" {
		c.Paths.Output = "site"
	}
	if c.Scan.Allowlist == "" {
		c.Scan.Allowlist = ".secretsallow"
	}
	if c.Scan.KeywordPaths == nil {
		c.Scan.KeywordPaths = []string{c.Paths.Content + "/**"}
	}
	if c.Sections == nil {
		c.Sections = map[string]Section{}
	}
}

// Validate rejects configs that are ambiguous or could damage the repo.
func (c *Config) Validate() error {
	if len(c.Source.Folders) == 0 {
		return errors.New("source.folders is empty: list at least one Apple Notes folder to publish")
	}
	for _, f := range c.Source.Folders {
		if c.IsExcluded(f) {
			return fmt.Errorf("folder %q is in both source.folders and source.exclude_folders (exclusion wins; remove one)", f)
		}
	}
	for name, p := range map[string]string{"paths.content": c.Paths.Content, "paths.output": c.Paths.Output} {
		cp := filepath.Clean(p)
		if cp == "." || filepath.IsAbs(cp) || cp == ".." || strings.HasPrefix(cp, ".."+string(filepath.Separator)) || cp == ".git" {
			return fmt.Errorf("%s must be a sub-directory of the repo, got %q", name, p)
		}
	}
	if filepath.Clean(c.Paths.Content) == filepath.Clean(c.Paths.Output) {
		return errors.New("paths.content and paths.output must differ")
	}
	if c.Site.Timezone != "" {
		if _, err := time.LoadLocation(c.Site.Timezone); err != nil {
			return fmt.Errorf("site.timezone: %w", err)
		}
	}
	return nil
}

// IsExcluded reports whether a folder name must never be read.
func (c *Config) IsExcluded(name string) bool {
	n := Norm(name)
	for _, e := range alwaysExcluded {
		if n == e {
			return true
		}
	}
	for _, e := range c.Source.ExcludeFolders {
		if n == Norm(e) {
			return true
		}
	}
	return false
}

// IsIncluded reports whether a top-level folder is published.
func (c *Config) IsIncluded(name string) bool {
	if c.IsExcluded(name) {
		return false
	}
	n := Norm(name)
	for _, f := range c.Source.Folders {
		if n == Norm(f) {
			return true
		}
	}
	return false
}

// ExcludeList is every excluded folder name (config plus built-ins).
func (c *Config) ExcludeList() []string {
	out := append([]string{}, alwaysExcluded...)
	return append(out, c.Source.ExcludeFolders...)
}

func (c *Config) ContentDir() string    { return filepath.Join(c.Root, c.Paths.Content) }
func (c *Config) OutputDir() string     { return filepath.Join(c.Root, c.Paths.Output) }
func (c *Config) AllowlistPath() string { return filepath.Join(c.Root, c.Scan.Allowlist) }

// CachePath is the git-ignored export cache (notes skipped by tag).
func (c *Config) CachePath() string { return filepath.Join(c.Root, ".notesync-cache.json") }

// Location is the timezone used to print dates on the site.
func (c *Config) Location() *time.Location {
	if c.Site.Timezone == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(c.Site.Timezone)
	if err != nil {
		return time.Local
	}
	return loc
}

// SectionMeta returns display metadata for a section slug, falling back to
// fallbackTitle (usually the original Apple Notes folder name).
func (c *Config) SectionMeta(slug, fallbackTitle string) Section {
	s := c.Sections[slug]
	if s.Title == "" {
		s.Title = fallbackTitle
	}
	if s.Title == "" {
		s.Title = titleCase(slug)
	}
	return s
}

// FolderOrder is the position of a section in source.folders, or len+1.
func (c *Config) FolderOrder(slug string) int {
	for i, f := range c.Source.Folders {
		if Slugify(f) == slug {
			return i
		}
	}
	return len(c.Source.Folders) + 1
}

// Slugify turns a note or folder name into a stable, URL-safe file name.
func Slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "note"
	}
	if len(out) > 80 {
		out = strings.Trim(out[:80], "-")
	}
	return out
}

func titleCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	for i, p := range parts {
		r := []rune(p)
		r[0] = unicode.ToUpper(r[0])
		parts[i] = string(r)
	}
	return strings.Join(parts, " ")
}
