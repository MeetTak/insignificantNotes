// Package convert turns the HTML that Apple Notes returns for a note body into
// clean Markdown.
//
// Apple Notes HTML is line oriented: every line is a <div>, empty lines are
// <div><br></div>, "Monospaced" text is <tt>, and images arrive inline as
// base64 data URIs. The converter keeps that line structure (the site renders
// with hard line breaks), turns runs of monospaced lines into fenced code
// blocks, and hands images to an ImageSink so they become real files.
package convert

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ImageSink stores a decoded image and returns the Markdown path to it.
type ImageSink interface {
	SaveImage(mime string, data []byte) (string, error)
}

type Options struct {
	// Title is the note title; a leading heading or line equal to it is
	// dropped because the site prints the title itself.
	Title string
	// Images receives inline images. When nil, images are dropped.
	Images ImageSink
}

type Result struct {
	Markdown string
	Warnings []string
}

type kind int

const (
	kLine  kind = iota // one line of a paragraph
	kBlank             // paragraph separator
	kCode              // one line of a code block
	kChunk             // a pre-rendered block (heading, list, table…)
)

type block struct {
	kind  kind
	text  string
	plain string
}

type converter struct {
	opt      Options
	blocks   []block
	warnings []string
}

// ToMarkdown converts one Apple Notes body.
func ToMarkdown(src string, opt Options) (Result, error) {
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return Result{}, err
	}
	root := find(doc, atom.Body)
	if root == nil {
		root = doc
	}
	mergeAdjacent(root)
	c := &converter{opt: opt}
	c.blockChildren(root)
	c.stripTitle()
	return Result{Markdown: c.assemble(), Warnings: c.warnings}, nil
}

func (c *converter) add(b block) { c.blocks = append(c.blocks, b) }
func (c *converter) warn(format string, a ...any) {
	c.warnings = append(c.warnings, fmt.Sprintf(format, a...))
}
func children(n *html.Node) (out []*html.Node) {
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		out = append(out, ch)
	}
	return
}
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func nbsp(s string) string     { return strings.ReplaceAll(s, " ", " ") }
func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }
func isAlnum(r rune) bool      { return unicode.IsLetter(r) || unicode.IsDigit(r) }

func find(n *html.Node, a atom.Atom) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == a {
		return n
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if f := find(ch, a); f != nil {
			return f
		}
	}
	return nil
}

// mergeable formatting tags. Apple Notes splits one bold sentence into many
// adjacent <b> elements (<b>if class </b><b><i>A</i></b><b> is…</b>); emitted
// one by one they produce unreadable "*****" runs, so they are merged first.
var mergeable = map[atom.Atom]bool{
	atom.B: true, atom.Strong: true, atom.I: true, atom.Em: true, atom.U: true,
	atom.S: true, atom.Strike: true, atom.Del: true, atom.Tt: true, atom.Code: true,
}

func mergeAdjacent(n *html.Node) {
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		for ch.Type == html.ElementNode && mergeable[ch.DataAtom] {
			next := ch.NextSibling
			if next == nil || next.Type != html.ElementNode || next.DataAtom != ch.DataAtom {
				break
			}
			for k := next.FirstChild; k != nil; {
				after := k.NextSibling
				next.RemoveChild(k)
				ch.AppendChild(k)
				k = after
			}
			n.RemoveChild(next)
		}
		mergeAdjacent(ch)
	}
}

var blockAtoms = map[atom.Atom]bool{
	atom.Div: true, atom.P: true, atom.H1: true, atom.H2: true, atom.H3: true,
	atom.H4: true, atom.H5: true, atom.H6: true, atom.Ul: true, atom.Ol: true,
	atom.Table: true, atom.Pre: true, atom.Blockquote: true, atom.Hr: true,
	atom.Object: true, atom.Section: true, atom.Article: true, atom.Header: true,
	atom.Footer: true, atom.Main: true, atom.Figure: true, atom.Li: true,
}

func isBlock(n *html.Node) bool { return n.Type == html.ElementNode && blockAtoms[n.DataAtom] }

func hasBlockChild(n *html.Node) bool {
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if isBlock(ch) {
			return true
		}
	}
	return false
}

// blockChildren handles n's children, grouping consecutive inline nodes into
// one line run.
func (c *converter) blockChildren(n *html.Node) {
	var run []*html.Node
	flush := func() {
		if len(run) > 0 {
			c.run(run)
			run = nil
		}
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if isBlock(ch) {
			flush()
			c.block(ch)
		} else {
			run = append(run, ch)
		}
	}
	flush()
}

func (c *converter) block(n *html.Node) {
	switch n.DataAtom {
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		plain := collapse(rawText(n))
		if plain == "" {
			return
		}
		lvl := int(n.Data[1]-'0') + 1 // h1 is the page title, so shift down
		if lvl > 6 {
			lvl = 6
		}
		txt := strings.TrimSpace(strings.ReplaceAll(c.inline(children(n)), "\n", " "))
		c.add(block{kind: kChunk, text: strings.Repeat("#", lvl) + " " + txt, plain: plain})
	case atom.Ul, atom.Ol:
		var lines []string
		c.list(n, "", &lines)
		if len(lines) > 0 {
			c.add(block{kind: kChunk, text: strings.Join(lines, "\n")})
		}
	case atom.Table:
		c.table(n)
	case atom.Pre:
		code := strings.TrimRight(rawText(n), "\n")
		if strings.TrimSpace(code) != "" {
			c.add(block{kind: kChunk, text: fence(code)})
		}
	case atom.Blockquote:
		sub := &converter{opt: c.opt}
		sub.blockChildren(n)
		c.warnings = append(c.warnings, sub.warnings...)
		md := strings.TrimRight(sub.assemble(), "\n")
		if md == "" {
			return
		}
		lines := strings.Split(md, "\n")
		for i, l := range lines {
			if l == "" {
				lines[i] = ">"
			} else {
				lines[i] = "> " + l
			}
		}
		c.add(block{kind: kChunk, text: strings.Join(lines, "\n")})
	case atom.Hr:
		c.add(block{kind: kChunk, text: "---"})
	default:
		if hasBlockChild(n) {
			c.blockChildren(n)
		} else {
			c.run(children(n))
		}
	}
}

// run converts a sequence of inline nodes (usually one <div>'s content).
func (c *converter) run(nodes []*html.Node) {
	if !hasContent(nodes) {
		if containsAtom(nodes, atom.Br) {
			c.add(block{kind: kBlank})
		}
		return
	}
	if isMono(nodes) {
		raw := strings.TrimRight(rawText(nodes...), "\n")
		for _, l := range strings.Split(raw, "\n") {
			c.add(block{kind: kCode, text: strings.TrimRight(l, " \t")})
		}
		return
	}
	plain := collapse(rawText(nodes...))
	lines := strings.Split(c.inline(nodes), "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			c.add(block{kind: kBlank})
			continue
		}
		c.add(block{kind: kLine, text: escapeLineStart(l), plain: plain})
	}
}

func hasContent(nodes []*html.Node) bool {
	for _, n := range nodes {
		switch n.Type {
		case html.TextNode:
			if strings.TrimSpace(nbsp(n.Data)) != "" {
				return true
			}
		case html.ElementNode:
			if n.DataAtom == atom.Img || hasContent(children(n)) {
				return true
			}
		}
	}
	return false
}

func containsAtom(nodes []*html.Node, a atom.Atom) bool {
	for _, n := range nodes {
		if n.Type == html.ElementNode && (n.DataAtom == a || containsAtom(children(n), a)) {
			return true
		}
	}
	return false
}

// isMono reports whether every visible character sits inside <tt>/<code>.
func isMono(nodes []*html.Node) bool {
	saw, ok := false, true
	var walk func(n *html.Node, mono bool)
	walk = func(n *html.Node, mono bool) {
		if !ok {
			return
		}
		switch n.Type {
		case html.TextNode:
			if strings.TrimSpace(nbsp(n.Data)) != "" {
				saw = true
				if !mono {
					ok = false
				}
			}
		case html.ElementNode:
			if n.DataAtom == atom.Img {
				ok = false
				return
			}
			m := mono || n.DataAtom == atom.Tt || n.DataAtom == atom.Code ||
				n.DataAtom == atom.Kbd || n.DataAtom == atom.Samp || n.DataAtom == atom.Pre
			for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
				walk(ch, m)
			}
		}
	}
	for _, n := range nodes {
		walk(n, false)
	}
	return ok && saw
}

// rawText is the visible text, with <br> and block ends as newlines.
func rawText(nodes ...*html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			b.WriteString(nbsp(n.Data))
		case html.ElementNode:
			if n.DataAtom == atom.Br {
				b.WriteByte('\n')
				return
			}
			for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
				walk(ch)
			}
			if isBlock(n) && b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
				b.WriteByte('\n')
			}
		default:
			for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
				walk(ch)
			}
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	return b.String()
}

func (c *converter) inline(nodes []*html.Node) string {
	var b strings.Builder
	for _, n := range nodes {
		b.WriteString(c.inlineNode(n))
	}
	return b.String()
}

var linkEscaper = strings.NewReplacer(" ", "%20", "<", "%3C", ">", "%3E")

func (c *converter) inlineNode(n *html.Node) string {
	switch n.Type {
	case html.TextNode:
		return escapeInline(strings.ReplaceAll(nbsp(n.Data), "\n", " "))
	case html.ElementNode:
	default:
		return ""
	}
	kids := children(n)
	switch n.DataAtom {
	case atom.Br:
		return "\n"
	case atom.B, atom.Strong:
		return wrap("**", "**", c.inline(kids))
	case atom.I, atom.Em:
		return wrap("*", "*", c.inline(kids))
	case atom.U, atom.Ins:
		return wrap("<u>", "</u>", c.inline(kids))
	case atom.S, atom.Strike, atom.Del:
		return wrap("~~", "~~", c.inline(kids))
	case atom.Sub:
		return wrap("<sub>", "</sub>", c.inline(kids))
	case atom.Sup:
		return wrap("<sup>", "</sup>", c.inline(kids))
	case atom.Tt, atom.Code, atom.Kbd, atom.Samp:
		return codeSpan(rawText(kids...))
	case atom.A:
		href := strings.TrimSpace(attr(n, "href"))
		label := strings.TrimSpace(strings.ReplaceAll(c.inline(kids), "\n", " "))
		if href == "" || strings.HasPrefix(strings.ToLower(href), "javascript:") {
			return label
		}
		if label == "" {
			label = escapeInline(href)
		}
		dest := linkEscaper.Replace(href)
		if strings.ContainsAny(dest, "()") {
			dest = "<" + dest + ">"
		}
		return "[" + label + "](" + dest + ")"
	case atom.Img:
		return c.image(n)
	case atom.Script, atom.Style, atom.Head, atom.Title:
		return ""
	default:
		s := c.inline(kids)
		if isBlock(n) {
			s += "\n"
		}
		return s
	}
}

// wrap surrounds each non-empty line of s with open/close, keeping the
// surrounding whitespace outside the markers (CommonMark requires that).
func wrap(open, close, s string) string {
	if strings.TrimSpace(s) == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		core := strings.TrimSpace(l)
		if core == "" {
			continue
		}
		lead := l[:strings.Index(l, core)]
		trail := l[len(lead)+len(core):]
		lines[i] = lead + open + core + close + trail
	}
	return strings.Join(lines, "\n")
}

func codeSpan(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	core := strings.TrimSpace(s)
	if core == "" {
		return s
	}
	lead := s[:strings.Index(s, core)]
	trail := s[len(lead)+len(core):]
	ticks := strings.Repeat("`", longestRun(core, '`')+1)
	pad := ""
	if strings.HasPrefix(core, "`") || strings.HasSuffix(core, "`") {
		pad = " "
	}
	return lead + ticks + pad + core + pad + ticks + trail
}

func longestRun(s string, ch rune) int {
	best, cur := 0, 0
	for _, r := range s {
		if r == ch {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	return best
}

func fence(code string) string {
	n := 3
	if r := longestRun(code, '`'); r >= n {
		n = r + 1
	}
	f := strings.Repeat("`", n)
	return f + "\n" + code + "\n" + f
}

var entityRe = regexp.MustCompile(`^&(#[0-9]+|#[xX][0-9a-fA-F]+|[A-Za-z][A-Za-z0-9]*);`)

// escapeInline escapes characters Markdown would otherwise interpret. "_" is
// left alone inside words (snake_case) because CommonMark ignores it there.
func escapeInline(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		switch r {
		case '\\', '`', '*', '[', ']', '<', '~':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '_':
			if i > 0 && i < len(rs)-1 && isAlnum(rs[i-1]) && isAlnum(rs[i+1]) {
				b.WriteRune(r)
			} else {
				b.WriteString(`\_`)
			}
		case '&':
			end := i + 12
			if end > len(rs) {
				end = len(rs)
			}
			if entityRe.MatchString(string(rs[i:end])) {
				b.WriteString(`\&`)
			} else {
				b.WriteRune(r)
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

var (
	reOrdered    = regexp.MustCompile(`^(\d{1,9})([.)])(\s|$)`)
	reBlockStart = regexp.MustCompile(`^(#{1,6}(\s|$)|[-+*](\s|$)|>|=+\s*$|-+\s*$|\++\s*$)`)
)

// escapeLineStart stops a plain line from being read as a heading, list item,
// quote or setext underline.
func escapeLineStart(l string) string {
	if m := reOrdered.FindStringSubmatchIndex(l); m != nil {
		return l[:m[4]] + `\` + l[m[4]:]
	}
	if reBlockStart.MatchString(l) {
		return `\` + l
	}
	return l
}

func (c *converter) list(n *html.Node, indent string, out *[]string) {
	ordered := n.DataAtom == atom.Ol
	idx := 1
	if v := attr(n, "start"); v != "" {
		fmt.Sscanf(v, "%d", &idx)
	}
	checklist := strings.Contains(strings.ToLower(attr(n, "class")), "checklist")
	childIndent := indent + "  "
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if ch.Type != html.ElementNode {
			continue
		}
		switch ch.DataAtom {
		case atom.Li:
			marker := "-"
			if ordered {
				marker = fmt.Sprintf("%d.", idx)
				idx++
			}
			ci := indent + strings.Repeat(" ", len(marker)+1)
			var inl, nested []*html.Node
			for k := ch.FirstChild; k != nil; k = k.NextSibling {
				if k.Type == html.ElementNode && (k.DataAtom == atom.Ul || k.DataAtom == atom.Ol) {
					nested = append(nested, k)
				} else {
					inl = append(inl, k)
				}
			}
			var lines []string
			for _, l := range strings.Split(c.inline(inl), "\n") {
				if l = strings.TrimSpace(l); l != "" {
					lines = append(lines, escapeLineStart(l))
				}
			}
			if len(lines) == 0 && len(nested) == 0 {
				continue
			}
			if len(lines) == 0 {
				lines = []string{""}
			}
			if checklist {
				box := "[ ] "
				if strings.Contains(strings.ToLower(attr(ch, "class")), "checked") {
					box = "[x] "
				}
				lines[0] = box + lines[0]
			}
			*out = append(*out, strings.TrimRight(indent+marker+" "+lines[0], " "))
			for _, l := range lines[1:] {
				*out = append(*out, ci+l)
			}
			for _, nl := range nested {
				c.list(nl, ci, out)
			}
			childIndent = ci
		case atom.Ul, atom.Ol:
			c.list(ch, childIndent, out)
		}
	}
}

func (c *converter) table(n *html.Node) {
	var rows [][]string
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		for ch := x.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Type != html.ElementNode {
				continue
			}
			switch ch.DataAtom {
			case atom.Tr:
				var cells []string
				for td := ch.FirstChild; td != nil; td = td.NextSibling {
					if td.Type != html.ElementNode || (td.DataAtom != atom.Td && td.DataAtom != atom.Th) {
						continue
					}
					var parts []string
					for _, l := range strings.Split(c.inline(children(td)), "\n") {
						if l = strings.TrimSpace(l); l != "" {
							parts = append(parts, l)
						}
					}
					cells = append(cells, strings.ReplaceAll(strings.Join(parts, "<br>"), "|", `\|`))
				}
				rows = append(rows, cells)
			case atom.Thead, atom.Tbody, atom.Tfoot:
				walk(ch)
			}
		}
	}
	walk(n)
	cols := 0
	for _, r := range rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	if cols == 0 {
		return
	}
	var lines []string
	for i, r := range rows {
		for len(r) < cols {
			r = append(r, "")
		}
		lines = append(lines, "| "+strings.Join(r, " | ")+" |")
		if i == 0 {
			lines = append(lines, "|"+strings.Repeat(" --- |", cols))
		}
	}
	c.add(block{kind: kChunk, text: strings.Join(lines, "\n")})
}

func (c *converter) image(n *html.Node) string {
	src := strings.TrimSpace(attr(n, "src"))
	alt := escapeInline(attr(n, "alt"))
	switch {
	case strings.HasPrefix(src, "data:"):
		mime, data, err := decodeDataURI(src)
		if err != nil {
			c.warn("image skipped: %v", err)
			return ""
		}
		if c.opt.Images == nil {
			c.warn("image dropped: no image sink configured")
			return ""
		}
		p, err := c.opt.Images.SaveImage(mime, data)
		if err != nil {
			c.warn("image skipped: %v", err)
			return ""
		}
		return "![" + alt + "](" + p + ")"
	case strings.HasPrefix(src, "http://"), strings.HasPrefix(src, "https://"):
		return "![" + alt + "](" + linkEscaper.Replace(src) + ")"
	}
	return ""
}

func decodeDataURI(s string) (string, []byte, error) {
	meta, payload, ok := strings.Cut(strings.TrimPrefix(s, "data:"), ",")
	if !ok {
		return "", nil, fmt.Errorf("malformed data URI")
	}
	mime := strings.TrimSpace(strings.Split(meta, ";")[0])
	if !strings.Contains(meta, ";base64") {
		d, err := url.PathUnescape(payload)
		return mime, []byte(d), err
	}
	payload = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, payload)
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(payload, "="))
	}
	return mime, data, err
}

func normTitle(s string) string {
	s = strings.TrimLeft(s, "# ")
	s = strings.NewReplacer("**", "", "\\", "").Replace(s)
	return strings.ToLower(collapse(s))
}

func (c *converter) stripTitle() {
	t := normTitle(c.opt.Title)
	if t == "" {
		return
	}
	for i, b := range c.blocks {
		if b.kind == kBlank {
			continue
		}
		if (b.plain != "" && normTitle(b.plain) == t) || normTitle(b.text) == t {
			c.blocks = c.blocks[i+1:]
		}
		return
	}
}

func (c *converter) assemble() string {
	var chunks []string
	bs := c.blocks
	for i := 0; i < len(bs); {
		switch bs[i].kind {
		case kBlank:
			i++
		case kLine, kCode:
			k := bs[i].kind
			var ls []string
			for i < len(bs) && bs[i].kind == k {
				ls = append(ls, bs[i].text)
				i++
			}
			if k == kCode {
				chunks = append(chunks, fence(strings.Join(ls, "\n")))
			} else {
				chunks = append(chunks, strings.Join(ls, "\n"))
			}
		default:
			chunks = append(chunks, bs[i].text)
			i++
		}
	}
	if len(chunks) == 0 {
		return ""
	}
	return strings.Join(chunks, "\n\n") + "\n"
}
