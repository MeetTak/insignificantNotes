package convert

import (
	"bytes"
	"testing"
)

type fakeSink struct {
	mime string
	data []byte
}

func (f *fakeSink) SaveImage(mime string, data []byte) (string, error) {
	f.mime, f.data = mime, data
	return "assets/abc.png", nil
}

func TestToMarkdown(t *testing.T) {
	cases := []struct {
		name, title, in, want string
	}{
		{"title stripped, lines kept, blank lines split paragraphs", "T",
			`<div><h1>T</h1></div><div><br></div><div>a</div><div>b</div><div><br></div><div>c</div>`,
			"a\nb\n\nc\n"},
		{"monospaced lines become one code block", "",
			`<div>run</div><div><tt>tar -zxvf x.tgz</tt></div><div><tt>chown -R a:b c</tt></div>`,
			"run\n\n```\ntar -zxvf x.tgz\nchown -R a:b c\n```\n"},
		{"plain lines that look like markdown syntax are escaped", "",
			`<div># root</div><div>1. step</div><div>- dash</div>`,
			"\\# root\n1\\. step\n\\- dash\n"},
		{"inline formatting and links", "",
			`<div><b>bold</b> and <i>it</i> and <tt>code</tt> <a href="https://x.com/a b">link</a></div>`,
			"**bold** and *it* and `code` [link](https://x.com/a%20b)\n"},
		{"nested unordered list", "",
			`<ul><li>a</li><ul><li>b</li></ul><li>c</li></ul>`,
			"- a\n  - b\n- c\n"},
		{"ordered list", "",
			`<ol><li>x</li><li>y</li></ol>`,
			"1. x\n2. y\n"},
		{"special characters escaped, snake_case untouched", "",
			`<div>a*b_c snake_case [x] &lt;tag&gt; ~/.m2</div>`,
			"a\\*b_c snake_case \\[x\\] \\<tag> \\~/.m2\n"},
		{"table inside object", "",
			`<object><table><tbody><tr><td><div>h1</div></td><td><div>h2</div></td></tr><tr><td><div>a|b</div></td><td><div>c</div></td></tr></tbody></table></object>`,
			"| h1 | h2 |\n| --- | --- |\n| a\\|b | c |\n"},
		{"legacy entity without semicolon", "",
			`<div>run_as_user=&quotnexus”</div>`,
			"run_as_user=\"nexus”\n"},
		{"bold + monospace prompt is still code", "",
			`<div><b><tt>mtrix@ubuntu1</tt></b><tt>:</tt><b><tt>~</tt></b><tt>$</tt></div>`,
			"```\nmtrix@ubuntu1:~$\n```\n"},
		{"note headings shift down one level", "",
			`<div><h2>Sub</h2></div><div>x</div>`,
			"### Sub\n\nx\n"},
		{"setext underline guarded", "",
			`<div>text</div><div>---</div>`,
			"text\n\\---\n"},
		{"apple dash list", "",
			`<ul class="Apple-dash-list"><li>nexus folder</li></ul><div>it contains runtime</div>`,
			"- nexus folder\n\nit contains runtime\n"},
		{"adjacent bold runs are merged (Apple Notes splits them)", "",
			`<div>in simple words <b>if class </b><b><i>A</i></b><b> is a subtype of </b><b><i>B</i></b><b>, replace </b><b><i>B </i></b><b>with </b><b><i>A</i></b></div>`,
			"in simple words **if class *A* is a subtype of *B*, replace *B* with *A***\n"},
		{"empty note", "Only title",
			`<div><h1>Only title</h1></div><div><br></div>`,
			""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := ToMarkdown(c.in, Options{Title: c.title})
			if err != nil {
				t.Fatal(err)
			}
			if res.Markdown != c.want {
				t.Errorf("got:\n%q\nwant:\n%q", res.Markdown, c.want)
			}
		})
	}
}

func TestImagesGoToSink(t *testing.T) {
	sink := &fakeSink{}
	res, err := ToMarkdown(`<div><img src="data:image/png;base64,iVBORw0KGgo="></div>`, Options{Images: sink})
	if err != nil {
		t.Fatal(err)
	}
	if res.Markdown != "![](assets/abc.png)\n" {
		t.Errorf("markdown = %q", res.Markdown)
	}
	if sink.mime != "image/png" || !bytes.Equal(sink.data, []byte("\x89PNG\r\n\x1a\n")) {
		t.Errorf("sink got mime=%q data=%x", sink.mime, sink.data)
	}
}

func TestImageWithoutSinkWarns(t *testing.T) {
	res, _ := ToMarkdown(`<div><img src="data:image/png;base64,iVBORw0KGgo="></div>`, Options{})
	if res.Markdown != "" || len(res.Warnings) != 1 {
		t.Errorf("got %q, warnings %v", res.Markdown, res.Warnings)
	}
}
