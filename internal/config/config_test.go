package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testConfig() *Config {
	c := &Config{Source: Source{
		Folders:        []string{"aws", "cloud", "Java"},
		ExcludeFolders: []string{"other", "others", "notes", "acad"},
	}}
	c.ApplyDefaults()
	return c
}

func TestExclusionIsCaseAndSpaceInsensitive(t *testing.T) {
	c := testConfig()
	for _, name := range []string{"Others ", "others", "OTHER", " acad", "ACAD", "Notes", "Recently Deleted"} {
		if !c.IsExcluded(name) {
			t.Errorf("%q should be excluded", name)
		}
		if c.IsIncluded(name) {
			t.Errorf("%q must never be included", name)
		}
	}
	for _, name := range []string{"aws", "java", "Java", " cloud "} {
		if !c.IsIncluded(name) {
			t.Errorf("%q should be included", name)
		}
	}
	if c.IsIncluded("blog") {
		t.Error("folders not listed are not published")
	}
}

func TestValidate(t *testing.T) {
	c := testConfig()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Source.Folders = append(c.Source.Folders, "Acad")
	if err := c.Validate(); err == nil {
		t.Error("a folder both included and excluded must be rejected")
	}
	c = testConfig()
	c.Paths.Output = "."
	if err := c.Validate(); err == nil {
		t.Error("output dir = repo root must be rejected")
	}
	c = testConfig()
	c.Paths.Output = "../elsewhere"
	if err := c.Validate(); err == nil {
		t.Error("output dir outside the repo must be rejected")
	}
	c = testConfig()
	c.Source.Folders = nil
	if err := c.Validate(); err == nil {
		t.Error("empty folder list must be rejected")
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, FileName)
	os.WriteFile(p, []byte(`{"source":{"folders":["aws"]},"sitee":{}}`), 0o644)
	if _, err := LoadFile(p); err == nil || !strings.Contains(err.Error(), "sitee") {
		t.Errorf("want unknown-field error, got %v", err)
	}
}

func TestLoadFindsConfigUpward(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, FileName), []byte(`{"source":{"folders":["aws"]}}`), 0o644)
	sub := filepath.Join(dir, "content", "aws")
	os.MkdirAll(sub, 0o755)
	c, err := Load(sub)
	if err != nil {
		t.Fatal(err)
	}
	if c.Paths.Content != "content" || c.Paths.Output != "site" || c.Site.BaseURL != "/" {
		t.Errorf("defaults not applied: %+v", c.Paths)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"config for vm(testPub/8100)": "config-for-vm-testpub-8100",
		"Java":                        "java",
		"  ":                          "note",
		"sonatype nexus installation": "sonatype-nexus-installation",
		"Spring Boot 3!":              "spring-boot-3",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
