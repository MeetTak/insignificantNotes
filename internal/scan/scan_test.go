package scan

import (
	"strings"
	"testing"
)

// fake returns a deterministic random-looking string. Test secrets are built
// at runtime (prefix + fake) so this file never contains a real-looking key
// that other scanners, or GitHub push protection, would flag.
func fake(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	seed := uint32(2166136261)
	b := make([]byte, n)
	for i := range b {
		seed = seed*1664525 + 1013904223
		b[i] = alphabet[(seed>>16)%uint32(len(alphabet))]
	}
	return string(b)
}

func scanLine(t *testing.T, line string) []Finding {
	t.Helper()
	s := New(nil, nil, nil)
	return s.ScanContent("content/aws/note.md", []byte(line))
}

func TestRulesDetect(t *testing.T) {
	cases := []struct {
		name, line, rule string
		sev              Severity
	}{
		{"aws access key id", "aws_access_key_id = " + "AKIA" + "Q3ZT7XK2LM9PW4RB", "aws-access-key-id", High},
		{"aws docs example key is only a warning", "key " + "AKIA" + "IOSFODNN7EXAMPLE", "aws-access-key-id", Warn},
		{"aws secret access key", "aws_secret_access_key = " + fake(20) + "/" + fake(19), "aws-secret-access-key", High},
		{"private key", "-----BEGIN " + "RSA PRIVATE KEY-----", "private-key", High},
		{"github token", "token " + "ghp_" + fake(36), "github-token", High},
		{"github fine-grained token", "github" + "_pat_" + fake(40), "github-token", High},
		{"anthropic key", "key: " + "sk-ant-" + "api03-" + fake(40), "anthropic-api-key", High},
		{"openai key", "export OPENAI=" + "sk-proj-" + fake(40), "openai-api-key", High},
		{"google api key", "maps " + "AIza" + fake(35), "google-api-key", High},
		{"slack token", "slack " + "xoxb-" + "1234567890-" + fake(24), "slack-token", High},
		{"stripe key", "stripe " + "sk_live_" + fake(24), "stripe-key", High},
		{"jwt", "Authorization: " + "eyJ" + fake(20) + ".eyJ" + fake(20) + "." + fake(20), "jwt", High},
		{"url credentials", "jdbc " + "https://admin:" + fake(14) + "9@db.internal.io/app", "url-credentials", High},
		{"url credential placeholder is a warning", "https://user:changeme@host.com/x", "url-credentials", Warn},
		{"generic api key assignment", "api_key = " + fake(26) + "7", "secret-assignment", High},
		{"generic placeholder is a warning", "password: your-password-here", "secret-assignment", Warn},
		{"low-entropy password is a warning", "password=nexusadmin", "secret-assignment", Warn},
		{"curl basic auth from a real note", "curl -u user:passwords -X GET 'http://10.0.0.5:8081/service/rest/v1/repositories'", "cli-basic-auth", Warn},
		{"bearer token", "Authorization: Bearer " + fake(32) + "9", "bearer-token", High},
		{"public ip", "ssh root@139.59.86.12", "public-ip", Warn},
		{"aws account id", "account id: 1234-5678-9012", "aws-account-id", Warn},
		{"email", "mail me at someone@gmail.com", "email", Warn},
		{"high entropy blob", "blob " + fake(44) + "Q7", "high-entropy", Warn},
		{"keyword only", "store the secret access key somewhere safe", "sensitive-keyword", Warn},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := scanLine(t, c.line)
			var got *Finding
			for i := range fs {
				if fs[i].RuleID == c.rule {
					got = &fs[i]
				}
			}
			if got == nil {
				t.Fatalf("rule %s did not fire on %q; findings: %+v", c.rule, c.line, fs)
			}
			if got.Severity != c.sev {
				t.Errorf("severity = %s, want %s", got.Severity, c.sev)
			}
		})
	}
}

func TestCleanTextHasNoFindings(t *testing.T) {
	for _, line := range []string{
		"use the chown command to set the user and group as nexus instead of root",
		"tar -zxvf nexus-3.87.1-01-linux-x86_64.tar.gz",
		"open http://192.168.1.10:8081 in the browser",
		"![](assets/0123456789abcdef.png)",
		"commit 3f786850e387550fdab836ed7e6dc881de23001b",
		"see https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html",
		"git@github.com:user/repo.git",
		"Thread class in the java.lang package extends the runnable interface",
	} {
		if fs := scanLine(t, line); len(fs) != 0 {
			t.Errorf("%q: unexpected findings %+v", line, fs)
		}
	}
}

func TestOneFindingPerSpan(t *testing.T) {
	fs := scanLine(t, "sk-ant-"+"api03-"+fake(40))
	if len(fs) != 1 || fs[0].RuleID != "anthropic-api-key" {
		t.Fatalf("want exactly one anthropic finding, got %+v", fs)
	}
}

func TestKeywordSuppressedWhenValueRuleMatches(t *testing.T) {
	fs := scanLine(t, "password = "+fake(16)+"8")
	for _, f := range fs {
		if f.RuleID == "sensitive-keyword" {
			t.Fatalf("keyword rule should not duplicate a value finding: %+v", fs)
		}
	}
}

func TestKeywordScope(t *testing.T) {
	s := New(nil, nil, []string{"content/**"})
	if fs := s.ScanContent("GUIDE.md", []byte("rotate the API key")); len(fs) != 0 {
		t.Errorf("keyword rule should not apply outside content/: %+v", fs)
	}
	if fs := s.ScanContent("content/x/y.md", []byte("rotate the API key")); len(fs) != 1 {
		t.Errorf("keyword rule should apply in content/: %+v", fs)
	}
}

func TestContentOnlyRules(t *testing.T) {
	s := New(nil, nil, []string{"content/**"})
	svg := `<path d="M12 .5a11.5 11.5 0 0 0-3.64 22.41c.58.1-3.36.96.1 5.68.41.36"/>`
	if fs := s.ScanContent("internal/site/templates/layout.html", []byte(svg+" someone@gmail.com")); len(fs) != 0 {
		t.Errorf("IP/email rules should only run on notes: %+v", fs)
	}
	if fs := s.ScanContent("content/cloud/vm.md", []byte("ssh root@139.59.86.12")); len(fs) != 1 {
		t.Errorf("IP rule should run on notes: %+v", fs)
	}
}

func TestCodeExpressionIsNotASecret(t *testing.T) {
	for _, line := range []string{"Secret: line[m[0]:m[1]],", `token := os.Getenv("GITHUB_TOKEN")`} {
		if fs := scanLine(t, line); len(fs) != 0 && fs[0].RuleID == "secret-assignment" {
			t.Errorf("%q flagged as a secret value: %+v", line, fs)
		}
	}
}

func TestInlineAllow(t *testing.T) {
	if fs := scanLine(t, "AKIA"+"Q3ZT7XK2LM9PW4RB "+InlineAllow); len(fs) != 0 {
		t.Errorf("inline allow ignored: %+v", fs)
	}
}

func TestAllowlist(t *testing.T) {
	key := "AKIA" + "Q3ZT7XK2LM9PW4RB"
	fp := Fingerprint("aws-access-key-id", key)
	for _, entry := range []string{"fp:" + fp, "match:" + key, "rule:aws-access-key-id", "path:content/**"} {
		a, err := ParseAllowlist(strings.NewReader("# comment\n" + entry + " # why\n"))
		if err != nil {
			t.Fatal(err)
		}
		s := New(a, nil, nil)
		if fs := s.ScanContent("content/aws/note.md", []byte("key "+key)); len(fs) != 0 {
			t.Errorf("%s did not suppress: %+v", entry, fs)
		}
	}
	if _, err := ParseAllowlist(strings.NewReader("bogus:thing")); err == nil {
		t.Error("unknown entry type should be an error")
	}
}

func TestIgnoreAndBinary(t *testing.T) {
	s := New(nil, []string{"**/*_test.go", "site/**"}, nil)
	secret := []byte("AKIA" + "Q3ZT7XK2LM9PW4RB")
	for _, p := range []string{"internal/scan/scan_test.go", "x_test.go", "site/aws/a.html"} {
		if fs := s.ScanContent(p, secret); len(fs) != 0 {
			t.Errorf("%s should be ignored", p)
		}
	}
	if fs := s.ScanContent("content/a/b.png", secret); len(fs) != 0 {
		t.Error("images should be skipped")
	}
	if fs := s.ScanContent("content/a/b.bin", append([]byte{0}, secret...)); len(fs) != 0 {
		t.Error("binary content should be skipped")
	}
}

func TestGlob(t *testing.T) {
	cases := []struct {
		glob, path string
		want       bool
	}{
		{"**/*_test.go", "a/b/c_test.go", true},
		{"**/*_test.go", "c_test.go", true},
		{"**/*_test.go", "a/b/c.go", false},
		{"site/**", "site/x/y.html", true},
		{"site/**", "content/site/x", false},
		{"content/*.md", "content/a.md", true},
		{"content/*.md", "content/a/b.md", false},
		{"content/??.md", "content/ab.md", true},
	}
	for _, c := range cases {
		if got := MatchGlob(c.glob, c.path); got != c.want {
			t.Errorf("MatchGlob(%q, %q) = %v", c.glob, c.path, got)
		}
	}
}

func TestRedactHidesSecret(t *testing.T) {
	s := "ghp_" + fake(36)
	r := Redact(s)
	if strings.Contains(r, s[6:30]) || len(r) >= len(s) {
		t.Errorf("Redact leaked too much: %q", r)
	}
	f := Finding{Severity: High, Secret: s}
	if f.Shown() == s {
		t.Error("HIGH findings must be redacted when shown")
	}
}

func TestEntropy(t *testing.T) {
	if Entropy("aaaa") != 0 {
		t.Error("constant string has zero entropy")
	}
	if e := Entropy(fake(40)); e < 4.3 {
		t.Errorf("fake() entropy %.2f too low for tests", e)
	}
}
