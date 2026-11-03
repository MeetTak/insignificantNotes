package scan

import (
	"math"
	"net"
	"regexp"
	"strings"
	"unicode"
)

// DefaultRules is the built-in rule set, most specific first. When two rules
// match the same characters only the first one is reported.
func DefaultRules() []Rule {
	return []Rule{
		// ---- provider tokens with unmistakable formats (HIGH) ----
		{ID: "private-key", Severity: High, Description: "private key block",
			Pattern: regexp.MustCompile(`-----BEGIN (?:[A-Z0-9]+ )*PRIVATE KEY(?: BLOCK)?-----`)},
		{ID: "aws-access-key-id", Severity: High, Description: "AWS access key ID",
			Pattern: regexp.MustCompile(`\b((?:AKIA|ASIA|ABIA|ACCA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|APKA)[0-9A-Z]{16})\b`), Group: 1,
			Refine: notExample(High)},
		{ID: "aws-secret-access-key", Severity: High, Description: "AWS secret access key",
			Pattern: regexp.MustCompile(`(?i)aws.{0,25}?(?:secret|private).{0,25}?[=:"'\s]\s*["']?([A-Za-z0-9/+]{40})(?:[^A-Za-z0-9/+=]|$)`), Group: 1,
			Refine: func(s string) Severity {
				if strings.Contains(strings.ToUpper(s), "EXAMPLE") || Entropy(s) < 4.0 {
					return Warn
				}
				return High
			}},
		{ID: "github-token", Severity: High, Description: "GitHub token",
			Pattern: regexp.MustCompile(`\b((?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{36,255}|github_pat_[A-Za-z0-9_]{22,255})\b`)},
		{ID: "gitlab-token", Severity: High, Description: "GitLab personal access token",
			Pattern: regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}`)},
		{ID: "anthropic-api-key", Severity: High, Description: "Anthropic API key",
			Pattern: regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`)},
		{ID: "openai-api-key", Severity: High, Description: "OpenAI API key",
			Pattern: regexp.MustCompile(`\bsk-(?:proj-|svcacct-|admin-)?[A-Za-z0-9_-]{20,}`),
			Refine: func(s string) Severity {
				if hasLetter(s) && hasDigit(s) {
					return High
				}
				return 0
			}},
		{ID: "google-api-key", Severity: High, Description: "Google API key",
			Pattern: regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`)},
		{ID: "slack-token", Severity: High, Description: "Slack token",
			Pattern: regexp.MustCompile(`\bxox[abposr]-[A-Za-z0-9-]{10,}`)},
		{ID: "slack-webhook", Severity: High, Description: "Slack incoming webhook URL",
			Pattern: regexp.MustCompile(`https://hooks\.slack\.com/services/[A-Za-z0-9/_-]{20,}`)},
		{ID: "stripe-key", Severity: High, Description: "Stripe secret key",
			Pattern: regexp.MustCompile(`\b(?:sk|rk)_(?:live|test)_[0-9A-Za-z]{16,}\b`)},
		{ID: "sendgrid-key", Severity: High, Description: "SendGrid API key",
			Pattern: regexp.MustCompile(`\bSG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}\b`)},
		{ID: "npm-token", Severity: High, Description: "npm access token",
			Pattern: regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}\b`)},
		{ID: "jwt", Severity: High, Description: "JSON Web Token",
			Pattern: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)},

		// ---- credentials in context (HIGH when the value looks real) ----
		{ID: "url-credentials", Severity: High, Description: "username:password inside a URL",
			Pattern: regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9+.-]*://[^\s:/@'"]+:([^\s:/@'"]+)@[^\s/'"]+`), Group: 1,
			Refine: secretValue},
		{ID: "cli-basic-auth", Severity: High, Description: "credentials passed to curl/wget",
			Pattern: regexp.MustCompile(`(?i)\b(?:curl|wget|http)\b[^\n]*?\s(?:-u|--user|--http-user|--http-password)[\s=]+["']?([^\s"']+)`), Group: 1,
			Refine: func(s string) Severity {
				_, pw, ok := strings.Cut(s, ":")
				if !ok {
					pw = s
				}
				return secretValue(pw)
			}},
		{ID: "bearer-token", Severity: High, Description: "bearer token",
			Pattern: regexp.MustCompile(`(?i)\bbearer\s+([A-Za-z0-9._~+/-]{20,}=*)`), Group: 1,
			Refine: secretValue},
		{ID: "secret-assignment", Severity: High, Description: "value assigned to a key/token/password",
			Pattern: regexp.MustCompile(`(?i)\b(?:api[_-]?key|apikey|api[_-]?secret|secret[_-]?key|client[_-]?secret|access[_-]?token|auth[_-]?token|refresh[_-]?token|private[_-]?key|password|passwd|pwd|secret|token)\b["']?\s*[:=]\s*["']?([^\s"'` + "`" + `,;]{8,})`), Group: 1,
			Refine: secretValue},

		// ---- personal / infrastructure details (WARN) ----
		{ID: "aws-account-id", Severity: Warn, Description: "AWS account ID", ContentOnly: true,
			Pattern: regexp.MustCompile(`(?i)\baccount(?:[ _-]?(?:id|number))?\b[^\n0-9]{0,20}\b(\d{4}-?\d{4}-?\d{4})\b`), Group: 1},
		{ID: "public-ip", Severity: Warn, Description: "public IP address", ContentOnly: true,
			Pattern: regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
			Refine:  publicIP},
		{ID: "email", Severity: Warn, Description: "email address", ContentOnly: true,
			Pattern: regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`),
			Refine:  email},
		{ID: "high-entropy", Severity: Warn, Description: "random-looking string (possible key)",
			Pattern: regexp.MustCompile(`[A-Za-z0-9+/_-]{32,}={0,2}`),
			Refine:  highEntropy},

		// ---- keyword search (WARN, only when nothing else matched the line) ----
		{ID: "sensitive-keyword", Severity: Warn, Description: "mentions credentials; check nothing real is here", LineScoped: true,
			Pattern: regexp.MustCompile(`(?i)\b(?:api[ _-]?keys?|secret[ _-]?(?:access[ _-]?)?keys?|access[ _-]?keys?|access[ _-]?tokens?|auth[ _-]?tokens?|client[ _-]?secrets?|private[ _-]?keys?|passwords?|passwd|credentials?|bearer|secrets?)\b`)},
	}
}

var placeholderHints = []string{
	"example", "your", "xxx", "***", "<", ">", "${", "{{", "placeholder", "dummy", "sample",
	"redacted", "replace", "todo", "changeme",
}

// secretValue grades a captured credential: placeholders and low-entropy
// words are warnings, mixed random-looking values are HIGH.
func secretValue(s string) Severity {
	// Code expressions such as line[m[0]:m[1]] or os.Getenv("X") are not values.
	if strings.ContainsAny(s, "([") && strings.ContainsAny(s, ")]") {
		return 0
	}
	l := strings.ToLower(s)
	for _, h := range placeholderHints {
		if strings.Contains(l, h) {
			return Warn
		}
	}
	if Entropy(s) >= 3.5 && hasLetter(s) && (hasDigit(s) || hasSymbol(s)) {
		return High
	}
	return Warn
}

func notExample(sev Severity) func(string) Severity {
	return func(s string) Severity {
		if strings.Contains(strings.ToUpper(s), "EXAMPLE") {
			return Warn
		}
		return sev
	}
}

func publicIP(s string) Severity {
	ip := net.ParseIP(s)
	if ip == nil {
		return 0
	}
	ip4 := ip.To4()
	if ip4 == nil || ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsMulticast() || ip4[0] == 0 || ip4[0] >= 240 {
		return 0
	}
	for _, doc := range []string{"192.0.2.", "198.51.100.", "203.0.113."} {
		if strings.HasPrefix(s, doc) {
			return 0
		}
	}
	return Warn
}

func email(s string) Severity {
	l := strings.ToLower(s)
	at := strings.LastIndex(l, "@")
	domain := l[at+1:]
	for _, d := range []string{"example.com", "example.org", "example.net", "test.com", "users.noreply.github.com", "noreply.github.com"} {
		if domain == d {
			return 0
		}
	}
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp"} {
		if strings.HasSuffix(l, ext) {
			return 0
		}
	}
	if strings.HasPrefix(l, "git@") {
		return 0
	}
	return Warn
}

var hexOnly = regexp.MustCompile(`^[0-9a-fA-F]+$`)

func highEntropy(s string) Severity {
	t := strings.TrimRight(s, "=")
	if len(t) < 32 || hexOnly.MatchString(t) || strings.Count(t, "/") > 3 || strings.Count(t, "-") > 4 {
		return 0
	}
	if !hasUpper(t) || !hasLower(t) || !hasDigit(t) || Entropy(t) < 4.3 {
		return 0
	}
	return Warn
}

// Entropy is the Shannon entropy of s in bits per character.
func Entropy(s string) float64 {
	if s == "" {
		return 0
	}
	counts := map[rune]int{}
	n := 0
	for _, r := range s {
		counts[r]++
		n++
	}
	var h float64
	for _, c := range counts {
		p := float64(c) / float64(n)
		h -= p * math.Log2(p)
	}
	return h
}

func hasLetter(s string) bool { return strings.IndexFunc(s, unicode.IsLetter) >= 0 }
func hasDigit(s string) bool  { return strings.IndexFunc(s, unicode.IsDigit) >= 0 }
func hasUpper(s string) bool  { return strings.IndexFunc(s, unicode.IsUpper) >= 0 }
func hasLower(s string) bool  { return strings.IndexFunc(s, unicode.IsLower) >= 0 }
func hasSymbol(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return unicode.IsPunct(r) || unicode.IsSymbol(r) }) >= 0
}
