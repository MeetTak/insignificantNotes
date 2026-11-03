// Package scan is notesync's secret scanner. It runs before anything is
// committed, pushed or deployed and looks for API keys, tokens, passwords and
// personal details that should not be published.
//
// HIGH findings block a commit/push/deploy. WARN findings are printed for a
// human to review and can be accepted into the allowlist.
package scan

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Severity int

const (
	Warn Severity = 1
	High Severity = 2
)

func (s Severity) String() string {
	switch s {
	case High:
		return "HIGH"
	case Warn:
		return "WARN"
	}
	return "NONE"
}

// InlineAllow on a line silences every finding on that line.
const InlineAllow = "notesync:allow"

type Rule struct {
	ID          string
	Description string
	Severity    Severity
	Pattern     *regexp.Regexp
	// Group is the capture group holding the secret (0 = whole match).
	Group int
	// Refine can re-grade a match; returning 0 drops it.
	Refine func(secret string) Severity
	// LineScoped rules flag a whole line, only when no other rule matched it,
	// and only in files matching Scanner.KeywordPaths.
	LineScoped bool
	// ContentOnly rules (personal/infrastructure details) only run on files
	// matching Scanner.KeywordPaths, i.e. the published notes.
	ContentOnly bool
}

type Finding struct {
	RuleID      string
	Description string
	Severity    Severity
	Path        string
	Line        int
	Col         int
	Secret      string
	Fingerprint string
}

// Shown is the matched text as printed: HIGH secrets are redacted.
func (f Finding) Shown() string {
	if f.Severity == High {
		return Redact(f.Secret)
	}
	s := f.Secret
	if len(s) > 60 {
		s = s[:57] + "..."
	}
	return s
}

type Scanner struct {
	Rules        []Rule
	Allow        *Allowlist
	Ignore       []string // globs of files never scanned
	KeywordPaths []string // globs where LineScoped/ContentOnly rules apply (empty = everywhere)
}

func New(allow *Allowlist, ignore, keywordPaths []string) *Scanner {
	return &Scanner{Rules: DefaultRules(), Allow: allow, Ignore: ignore, KeywordPaths: keywordPaths}
}

func (s *Scanner) Ignored(path string) bool {
	path = filepath.ToSlash(path)
	for _, g := range s.Ignore {
		if MatchGlob(g, path) {
			return true
		}
	}
	return false
}

func (s *Scanner) keywordsApply(path string) bool {
	if len(s.KeywordPaths) == 0 {
		return true
	}
	for _, g := range s.KeywordPaths {
		if MatchGlob(g, filepath.ToSlash(path)) {
			return true
		}
	}
	return false
}

var binaryExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".ico": true, ".heic": true,
	".pdf": true, ".zip": true, ".gz": true, ".tgz": true, ".mov": true, ".mp4": true, ".mp3": true,
	".woff": true, ".woff2": true, ".ttf": true, ".otf": true, ".exe": true, ".dylib": true, ".so": true,
}

func IsBinary(path string, data []byte) bool {
	if binaryExt[strings.ToLower(filepath.Ext(path))] {
		return true
	}
	head := data
	if len(head) > 8000 {
		head = head[:8000]
	}
	return bytes.IndexByte(head, 0) >= 0
}

// ScanContent scans one file's content. path should be repo-relative.
func (s *Scanner) ScanContent(path string, data []byte) []Finding {
	path = filepath.ToSlash(path)
	if s.Ignored(path) || IsBinary(path, data) || s.Allow.PathAllowed(path) {
		return nil
	}
	keywords := s.keywordsApply(path)
	var out []Finding
	for i, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, InlineAllow) {
			continue
		}
		var lineFindings []Finding
		var spans [][2]int
		handled := false
		for _, r := range s.Rules {
			if r.LineScoped || s.Allow.RuleDisabled(r.ID) || (r.ContentOnly && !keywords) {
				continue
			}
			for _, m := range r.Pattern.FindAllStringSubmatchIndex(line, -1) {
				st, en := m[0], m[1]
				if r.Group > 0 && 2*r.Group+1 < len(m) && m[2*r.Group] >= 0 {
					st, en = m[2*r.Group], m[2*r.Group+1]
				}
				if overlaps(spans, st, en) {
					continue
				}
				secret := line[st:en]
				sev := r.Severity
				if r.Refine != nil {
					if sev = r.Refine(secret); sev == 0 {
						continue
					}
				}
				spans = append(spans, [2]int{st, en})
				handled = true
				f := Finding{RuleID: r.ID, Description: r.Description, Severity: sev, Path: path,
					Line: i + 1, Col: st + 1, Secret: secret, Fingerprint: Fingerprint(r.ID, secret)}
				if !s.Allow.Allowed(f) {
					lineFindings = append(lineFindings, f)
				}
			}
		}
		if !handled && keywords {
			for _, r := range s.Rules {
				if !r.LineScoped || s.Allow.RuleDisabled(r.ID) {
					continue
				}
				if m := r.Pattern.FindStringIndex(line); m != nil {
					f := Finding{RuleID: r.ID, Description: r.Description, Severity: r.Severity, Path: path,
						Line: i + 1, Col: m[0] + 1, Secret: line[m[0]:m[1]],
						Fingerprint: Fingerprint(r.ID, path+"\x00"+strings.TrimSpace(line))}
					if !s.Allow.Allowed(f) {
						lineFindings = append(lineFindings, f)
					}
					break
				}
			}
		}
		out = append(out, lineFindings...)
	}
	return out
}

func overlaps(spans [][2]int, st, en int) bool {
	for _, sp := range spans {
		if st < sp[1] && sp[0] < en {
			return true
		}
	}
	return false
}

// Fingerprint identifies a finding stably across runs.
func Fingerprint(rule, value string) string {
	sum := sha256.Sum256([]byte(rule + "\x00" + value))
	return hex.EncodeToString(sum[:])[:12]
}

// Redact keeps just enough of a secret to recognise it.
func Redact(s string) string {
	r := []rune(s)
	if len(r) <= 8 {
		if len(r) <= 2 {
			return strings.Repeat("*", len(r))
		}
		return string(r[:2]) + strings.Repeat("*", len(r)-2)
	}
	stars := len(r) - 6
	if stars > 12 {
		stars = 12
	}
	return string(r[:4]) + strings.Repeat("*", stars) + string(r[len(r)-2:])
}

// Count returns the number of HIGH and WARN findings.
func Count(fs []Finding) (high, warn int) {
	for _, f := range fs {
		if f.Severity == High {
			high++
		} else {
			warn++
		}
	}
	return
}

// Sort orders findings by severity, then location.
func Sort(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.Severity != b.Severity {
			return a.Severity > b.Severity
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Line < b.Line
	})
}

// WriteText prints a human-readable report.
func WriteText(w io.Writer, fs []Finding, color bool) {
	c := func(code, s string) string {
		if !color {
			return s
		}
		return "\x1b[" + code + "m" + s + "\x1b[0m"
	}
	for _, f := range fs {
		label := c("33;1", "WARN")
		if f.Severity == High {
			label = c("31;1", "HIGH")
		}
		fmt.Fprintf(w, "%s  %s  %s  %s\n", label, c("1", fmt.Sprintf("%s:%d", f.Path, f.Line)), f.RuleID, c("2", "fp:"+f.Fingerprint))
		fmt.Fprintf(w, "      %s: %s\n", f.Description, f.Shown())
	}
}

// WriteGitHub prints GitHub Actions workflow annotations.
func WriteGitHub(w io.Writer, fs []Finding) {
	for _, f := range fs {
		level := "warning"
		if f.Severity == High {
			level = "error"
		}
		fmt.Fprintf(w, "::%s file=%s,line=%d,col=%d,title=%s::%s (%s, fp:%s)\n",
			level, f.Path, f.Line, f.Col, f.RuleID, f.Description, f.Shown(), f.Fingerprint)
	}
}

// ---------------------------------------------------------------- allowlist

type Allowlist struct {
	fps     map[string]bool
	matches map[string]bool
	rules   map[string]bool
	paths   []string
}

func LoadAllowlist(path string) (*Allowlist, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Allowlist{}, nil
		}
		return nil, err
	}
	defer f.Close()
	a, err := ParseAllowlist(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return a, nil
}

func ParseAllowlist(r io.Reader) (*Allowlist, error) {
	a := &Allowlist{fps: map[string]bool{}, matches: map[string]bool{}, rules: map[string]bool{}}
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		if i := strings.Index(line, " #"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kind, val, ok := strings.Cut(line, ":")
		val = strings.TrimSpace(val)
		if !ok || val == "" {
			return nil, fmt.Errorf("line %d: expected fp:, match:, path: or rule:, got %q", n, line)
		}
		switch kind {
		case "fp":
			a.fps[val] = true
		case "match":
			a.matches[val] = true
		case "rule":
			a.rules[val] = true
		case "path":
			a.paths = append(a.paths, val)
		default:
			return nil, fmt.Errorf("line %d: unknown entry type %q", n, kind)
		}
	}
	return a, sc.Err()
}

func (a *Allowlist) Allowed(f Finding) bool {
	if a == nil {
		return false
	}
	return a.fps[f.Fingerprint] || a.matches[f.Secret] || a.rules[f.RuleID]
}

func (a *Allowlist) RuleDisabled(id string) bool { return a != nil && a.rules[id] }

func (a *Allowlist) PathAllowed(p string) bool {
	if a == nil {
		return false
	}
	for _, g := range a.paths {
		if MatchGlob(g, p) {
			return true
		}
	}
	return false
}

// AcceptWarnings appends fp: entries for every WARN finding to the file.
func AcceptWarnings(path string, fs []Finding) (int, error) {
	existing, err := LoadAllowlist(path)
	if err != nil {
		return 0, err
	}
	var b strings.Builder
	n := 0
	seen := map[string]bool{}
	for _, f := range fs {
		if f.Severity != Warn || existing.fps[f.Fingerprint] || seen[f.Fingerprint] {
			continue
		}
		seen[f.Fingerprint] = true
		if n == 0 {
			fmt.Fprintf(&b, "\n# accepted %s\n", time.Now().Format("2006-01-02 15:04"))
		}
		fmt.Fprintf(&b, "fp:%s # %s %s:%d\n", f.Fingerprint, f.RuleID, f.Path, f.Line)
		n++
	}
	if n == 0 {
		return 0, nil
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	_, err = file.WriteString(b.String())
	return n, err
}

// ---------------------------------------------------------------- globs

var globCache = map[string]*regexp.Regexp{}

// MatchGlob matches slash paths against globs supporting *, ** and ?.
func MatchGlob(glob, path string) bool {
	re, ok := globCache[glob]
	if !ok {
		re = globToRegexp(glob)
		globCache[glob] = re
	}
	return re.MatchString(path)
}

func globToRegexp(g string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(g); i++ {
		switch ch := g[i]; ch {
		case '*':
			if i+1 < len(g) && g[i+1] == '*' {
				i++
				if i+1 < len(g) && g[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
				} else {
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}
