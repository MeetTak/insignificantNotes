# notesync guide

The complete guide to this project: how it works, how to set it up, how to use it every
day, how the secret scanner protects you, and how to change it.

**Contents**

1. [How it works](#1-how-it-works)
2. [One-time setup](#2-one-time-setup)
3. [Everyday workflow](#3-everyday-workflow)
4. [Command reference](#4-command-reference)
5. [Configuration (`notesync.json`)](#5-configuration-notesyncjson)
6. [How export works](#6-how-export-works)
7. [How Apple Notes formatting becomes Markdown](#7-how-apple-notes-formatting-becomes-markdown)
8. [Secret scanning](#8-secret-scanning)
9. [The website](#9-the-website)
10. [Tests](#10-tests)
11. [CI/CD: GitHub Actions and Pages](#11-cicd-github-actions-and-pages)
12. [Project layout](#12-project-layout)
13. [Extending and changing things](#13-extending-and-changing-things)
14. [Troubleshooting](#14-troubleshooting)
15. [Design decisions](#15-design-decisions)

---

## 1. How it works

```mermaid
flowchart LR
    A[Apple Notes<br/>aws · cloud · Java] -- "osascript + JXA<br/>(read-only)" --> B[notesync export]
    B --> C[content/*.md<br/>+ assets/*.png]
    C --> D{notesync scan}
    D -- HIGH --> X[stop]
    D -- warnings --> E[you confirm<br/>or accept]
    D -- clean --> F[notesync build]
    E --> F
    F --> G[site/]
    C --> H[git commit<br/>pre-commit hook scans]
    H --> I[git push<br/>pre-push hook scans]
    I --> J[GitHub Actions<br/>vet · tests · scan · build]
    J -- all green --> K[GitHub Pages]
```

There are three ideas behind it:

- **Apple Notes stays the place you write.** Nothing about your note-taking changes.
  `notesync` only *reads* from Notes, and it never creates, edits, moves or deletes a note.
- **Markdown in git is the source of truth for the site.** `content/` holds plain Markdown
  plus image files, so every change is reviewable in a diff and the site can be rebuilt
  anywhere (CI runs on Linux, which has no Apple Notes).
- **Nothing is published without passing the secret scan, several times over.** See
  [§ 8](#8-secret-scanning).

### The pieces

| Piece | File(s) | Job |
| --- | --- | --- |
| JXA bridge | `internal/notes/notes.js` | Runs inside `osascript`. Lists folders, lists notes in the allowed folders, and returns one note's HTML body. Never opens excluded folders. |
| Exporter | `internal/notes/export.go` | Works out which notes changed, converts them, writes `content/`, extracts images, and removes notes that disappeared. |
| Converter | `internal/convert/html2md.go` | Turns Apple Notes HTML into clean Markdown. |
| Scanner | `internal/scan/` | Regex + entropy rules for secrets and personal details. Also handles the allowlist. |
| Site builder | `internal/site/` | Markdown → HTML with sidebar, TOC, search index and themes. |
| CLI | `cmd/notesync/` | Wires the pieces into commands and runs git. |
| Gate tests | `tests/content_test.go` | Checks run against the real `content/`. |
| CI | `.github/workflows/pages.yml` | Re-checks everything, then deploys. |

---

## 2. One-time setup

### Prerequisites

- macOS (export needs Apple Notes and `osascript`; build and scan run anywhere)
- Go 1.24+ (`go version`)
- git, and optionally the GitHub CLI `gh`

### Build and initialise

```sh
cd insignificantNotes
make build                 # → bin/notesync
./bin/notesync init        # git init (branch main) + pre-commit / pre-push hooks
./bin/notesync doctor      # checks Notes access, folders, git, hooks
make install               # optional: puts `notesync` on your PATH (~/.local/bin)
```

The first time `notesync` talks to Notes, macOS asks whether your terminal may control
Notes. Click **OK**. If you clicked "Don't Allow", turn it back on in
**System Settings → Privacy & Security → Automation → (your terminal) → Notes**.

### Create the GitHub repository and turn on Pages

```sh
gh repo create insignificantNotes --public --source . --remote origin
# or create it on github.com and run: git remote add origin git@github.com:<you>/insignificantNotes.git
```

Then on GitHub: **Settings → Pages → Build and deployment → Source: GitHub Actions**.

Finally, set `site.repo_url` in `notesync.json` to the repo URL (this shows a GitHub icon
in the header). Keep `site.base_url` as `/<repo-name>/`, since only the 404 page needs it.
Your site will be at `https://<you>.github.io/insignificantNotes/`.

### First publish

```sh
notesync publish --dry-run   # export (no writes), scan, build; touches nothing
notesync serve               # look at it on http://localhost:8080
notesync publish             # for real
```

---

## 3. Everyday workflow

1. Write or edit notes in Apple Notes, in `aws`, `cloud` or `Java` (or their sub-folders).
2. Run `notesync publish`.
3. About a minute later, GitHub Actions has deployed the site.

That's it. To look before publishing, run `notesync export && notesync serve`. The preview
reloads by itself when you export again.

### Keeping a note private

| You want to… | Do this in Apple Notes |
| --- | --- |
| keep one note out of the site | add the tag `#private` (or `#nopublish`) anywhere in it |
| keep a note out *and* protected | lock it (locked notes are always skipped) |
| keep a whole folder out | it is already out unless it is listed in `source.folders` |
| unpublish a note | delete it, move it to an unpublished folder, lock it or tag it; the next export removes its page |

### Controlling order and grouping

- Notes are sorted by title in natural order (`module2` comes before `module10`).
- Sub-folders become groups on the section page (for example `aws/course`).
- To pin an order, edit the note's Markdown header in `content/` and add `order: "1"`.
  The next export overwrites the header **only if the note changed**, so for a
  permanent order it is better to name notes in the order you want them.

### Hand-written pages

Any `.md` file you add to `content/<section>/` that has no `source: "apple-notes"` header
is yours. Export never touches it. Use it for extra pages, or add
`content/<section>/_index.md` to put an introduction on a section page (its `title:` and
`description:` header keys override the section's).

---

## 4. Command reference

Every command looks for `notesync.json` by walking up from the current directory.

### `notesync publish`

export → scan → build → `git add` → `git commit` → `git push`.

| Flag | Meaning |
| --- | --- |
| `-m "msg"` | commit message (default: generated, listing added, updated and removed notes) |
| `--dry-run` | export in dry-run mode, then scan and build; no notes written, nothing committed or pushed |
| `--yes` | publish despite **warnings** (HIGH findings always block) |
| `--no-export` | publish what is already in `content/` |
| `--no-push` | commit only |
| `--force` | re-export every note, not just changed ones |
| `--all` | stage every change in the repo, not just `content/`, `.secretsallow` and `notesync.json` |

If the scan has warnings and you are in a terminal, you choose:
**[y]** publish once, **[a]** accept the warnings permanently (adds them to the
allowlist) and publish, or **[N]** stop. Outside a terminal it stops unless you pass `--yes`.

### `notesync export`

`--dry-run` lists what would change. `--force` re-converts every note (use it after
changing the converter).

### `notesync scan [paths…]`

| Flag | Meaning |
| --- | --- |
| *(no flags)* | scan the working tree (skips `.git`, `bin/`, `site/`, ignored paths) |
| `--staged` | scan the staged version of changed files (used by the pre-commit hook) |
| `--tracked` | scan every git-tracked file (used by the pre-push hook and CI) |
| `--strict` | warnings fail too (exit 1) |
| `--format github` | GitHub Actions annotations |
| `--rule <id>` | only report findings of one rule |
| `--accept-warnings` | append every reported WARN fingerprint to the allowlist |
| `--quiet` | no advice text |

Exit status: `0` clean or warnings only, `1` HIGH finding (or a warning under `--strict`).

### `notesync build`, `notesync serve`

`build` writes `site/`. `serve --port 8080` builds, serves on 127.0.0.1, rebuilds when
`content/` changes (`--no-watch` turns that off), and reloads open pages.

### `notesync init`, `notesync doctor`, `notesync folders`

- `init` runs `git init -b main` if needed and installs both hooks. It never overwrites a
  hook it didn't write unless you pass `--force`.
- `doctor` checks the config, Notes access, that each configured folder exists, `sips`,
  git, the `origin` remote, the hooks, and that `CLAUDE.md` is git-ignored.
- `folders` shows every folder *name* with its status: `published`, `excluded (never read)`,
  `inside an excluded folder (never read)` or `not published`.

### Make targets

`make help` lists them: `build`, `test`, `vet`, `check` (everything CI runs), `scan`,
`export`, `site`, `serve`, `dry-run`, `publish`, `hooks`, `install`, and `clean`
(`clean` deletes only `bin/` and `site/`; it never touches `content/` or Apple Notes).

---

## 5. Configuration (`notesync.json`)

Unknown keys are rejected, so a typo fails loudly instead of being ignored.

```jsonc
{
  "site": {
    "title": "insignificantNotes",          // header, home page, <title>
    "tagline": "…",                         // home page subtitle + meta description
    "author": "MeetTak",                    // footer
    "repo_url": "",                         // GitHub link in header (empty = hidden)
    "base_url": "/insignificantNotes/",     // only used by 404.html (needs absolute links)
    "timezone": "Asia/Kolkata"              // how dates are printed
  },
  "source": {
    "account": "iCloud",                    // Notes account to read ("" = all accounts)
    "folders": ["aws", "cloud", "Java"],    // top-level folders to publish (+ their sub-folders)
    "exclude_folders": ["other", "others", "notes", "acad"], // never read, at any depth
    "skip_tags": ["#private", "#nopublish"],// notes containing these are skipped
    "image_max_width": 1600                 // wider images are downscaled with sips (0 = off)
  },
  "sections": {                             // optional per-section display info, keyed by slug
    "aws": { "title": "AWS", "description": "…", "order": 1 }
  },
  "paths": { "content": "content", "output": "site" },
  "scan": {
    "allowlist": ".secretsallow",
    "ignore_paths": ["**/*_test.go", "**/testdata/**", "site/**", "bin/**", "go.sum"],
    "keyword_paths": ["content/**"],        // where keyword/IP/email/account rules run
    "fail_on_warnings": false               // true = warnings block like HIGH findings
  }
}
```

Folder matching ignores case and surrounding spaces, so `"Others "`, `"others"` and
`"OTHERS"` are the same folder. `Recently Deleted` is always excluded. A folder that is in
both `folders` and `exclude_folders` is a config error.

A section's **slug** is the folder name lower-cased with punctuation replaced by `-`
(`Java` → `java`). Sections appear in `order`, then in `source.folders` order.

---

## 6. How export works

1. **List.** One `osascript` call lists every note in the allowed folders: id, title,
   folder path, created and modified dates, and whether it is locked. Bodies are not read
   yet. Excluded sub-folders are skipped without being opened. The Go side drops anything
   under an excluded name again, as a second guard.
2. **Plan paths.** Each note gets `content/<section>/<sub-folders…>/<slug>.md`. Two notes
   with the same title in one folder become `slug.md` and `slug-2.md`, in creation order.
3. **Diff.** Every exported file records the note's `note_id` and `modified` time in its
   header. A note is re-read only if its modified time or path changed (or with `--force`).
   An unchanged run of 31 notes takes about a second.
4. **Fetch and convert** changed notes one at a time. If a note has a skip tag, it is
   recorded in `.notesync-cache.json` (git-ignored) so it isn't re-read until it changes.
5. **Images** arrive as base64 inside the HTML. Each one is saved as
   `content/<section>/assets/<sha256-16>.<ext>`, so identical images are stored once.
   Images wider than `image_max_width` are downscaled with macOS `sips`, and HEIC/TIFF are
   converted to JPEG/PNG.
6. **Remove** files whose note was deleted, moved out, locked or tagged, then delete images
   that nothing references any more and any empty folders.

**Safety guard:** if Apple Notes returns zero notes while `content/` has exported notes
(for example, permission was revoked), export refuses to delete anything unless you pass
`--force`. Files without `source: "apple-notes"` in their header are never touched.

### The Markdown header

```yaml
---
title: "sonatype nexus installation"
folder: "cloud"
created: "2025-12-24T18:06:51.000Z"
modified: "2025-12-26T10:14:44.000Z"
note_id: "x-coredata://…/ICNote/p3306"
source: "apple-notes"
---
```

---

## 7. How Apple Notes formatting becomes Markdown

| In Apple Notes | In Markdown / on the site |
| --- | --- |
| Title (first line) | `title:` header. Not repeated in the body. |
| Each line | a line. Consecutive lines stay together with line breaks. |
| Empty line | paragraph break |
| **Title / Heading / Subheading** inside a note | `##` / `###` / `####` (they also feed the "On this page" TOC) |
| **Monospaced** lines | a fenced code block (consecutive lines merge into one block, with syntax highlighting and a copy button on the site) |
| Monospaced words inside a sentence | `` `inline code` `` |
| Bold, italic, underline, strikethrough | `**b**`, `*i*`, `<u>u</u>`, `~~s~~` |
| Bulleted, dashed, numbered lists (nested) | `-` / `1.` lists |
| Checklists | `- [ ]` / `- [x]` |
| Tables | GitHub-style tables |
| Links | `[text](url)` |
| Images and screenshots | files in `assets/` |
| Attachments other than images (PDFs, audio, scans) | not exported (Notes doesn't include them in the HTML) |

Lines that would accidentally become Markdown syntax (`# root prompt`, `1. not a list`,
`---`) are escaped so they show exactly as you typed them.

**Tips for nicer pages:** use *Format → Monospaced* for commands and code, use
*Heading/Subheading* to split long notes (you get a table of contents), and keep one topic
per note.

---

## 8. Secret scanning

### Where it runs

| # | When | Command | What it checks | On HIGH |
| --- | --- | --- | --- | --- |
| 1 | `notesync publish`, after export | built in | `content/` (`--all`: whole repo) | stops; warnings need your OK |
| 2 | every `git commit` | pre-commit hook → `scan --staged` | exactly what is being committed | commit refused |
| 3 | every `git push` | pre-push hook → `scan --tracked` | every tracked file | push refused |
| 4 | `go test ./...` / `make check` | `tests/content_test.go` | `content/` | test fails |
| 5 | GitHub Actions | `scan --tracked --format github` + all tests | whole repo | **no deploy** |

Layers 2 and 3 can be bypassed with `git commit --no-verify`, so layer 5 exists as the
backstop: GitHub Pages only deploys from a green workflow.

### Severities

- **HIGH**: a real-looking credential. Always blocks. It is shown redacted (`ghp_************xy`).
- **WARN**: something to look at: a keyword such as "password", a placeholder credential,
  a public IP, an email, an AWS account ID, a random-looking string. It is printed, and
  publish asks you before continuing.

### Rules

| Rule id | Severity | Finds |
| --- | --- | --- |
| `private-key` | HIGH | `BEGIN … PRIVATE KEY` blocks |
| `aws-access-key-id` | HIGH (WARN if it contains `EXAMPLE`) | `AKIA…`, `ASIA…` and other AWS key ids |
| `aws-secret-access-key` | HIGH | 40-char secret next to "aws … secret" |
| `github-token` | HIGH | `ghp_`, `gho_`, `ghs_`, `github_pat_` … |
| `gitlab-token`, `anthropic-api-key`, `openai-api-key`, `google-api-key`, `slack-token`, `slack-webhook`, `stripe-key`, `sendgrid-key`, `npm-token`, `jwt` | HIGH | provider-specific token formats |
| `url-credentials` | HIGH / WARN | `scheme://user:pass@host` (WARN when the password looks like a placeholder) |
| `cli-basic-auth` | HIGH / WARN | `curl -u user:pass`, `wget --user` … |
| `bearer-token` | HIGH / WARN | `Bearer <token>` |
| `secret-assignment` | HIGH / WARN | `api_key = …`, `password: …`, `token=…` (HIGH only when the value is random-looking) |
| `aws-account-id` | WARN | 12-digit id near "account" |
| `public-ip` | WARN | non-private, non-documentation IPv4 addresses |
| `email` | WARN | email addresses (except example/noreply domains) |
| `high-entropy` | WARN | 32+ char random-looking strings |
| `sensitive-keyword` | WARN | lines that mention API keys, passwords, credentials, secrets, tokens… (only when no other rule matched the line) |

The last four, plus `sensitive-keyword`, run only on files matching `scan.keyword_paths`
(your notes). That keeps code and docs that legitimately mention "password" quiet.

### When the scanner finds something

**A real secret (HIGH):**
1. Remove it from the note in Apple Notes, then run `notesync publish` again.
2. **Rotate it** (create a new key and revoke the old one). If it was ever pushed, assume
   it is compromised, because git history and forks keep it.

**A false positive or an accepted warning.** Add it to `.secretsallow`:

```text
fp:2b6a6b5ff848            # one finding (fingerprints are printed by `notesync scan`)
match:AKIAIOSFODNN7EXAMPLE # a literal value, anywhere
path:content/java/**       # every finding in matching files
rule:email                 # switch a rule off entirely
```

Shortcuts: `notesync scan --accept-warnings` accepts every current warning, and
`notesync scan --rule sensitive-keyword --accept-warnings` accepts one rule's warnings. HIGH
findings are never auto-accepted. A keyword fingerprint includes the line's text, so
editing that line in the note makes the scanner ask again. You can also put
`notesync:allow` on a line to silence that line.

---

## 9. The website

### Pages

| URL | Page |
| --- | --- |
| `/` | home: hero, search, every section as a card with its notes (grouped by sub-folder), and recently updated notes |
| `/<section>/` | section overview: description, optional `_index.md` intro, each note with an excerpt, date and reading time |
| `/<section>/[<sub>/]<note>.html` | a note: sidebar, breadcrumb, content, "On this page" TOC, previous/next |
| `/search-index.json` | client-side search data |
| `/404.html` | not-found page |

### Features

- Light and dark themes: follows the OS, with a toggle that is remembered.
- Search: press `/` or `⌘K`. It searches titles, headings and text, with highlighted snippets.
- Code: highlighting with highlight.js (loaded from cdnjs) and a copy button.
- Images: lazy-loaded, click to zoom.
- Mobile: the sidebar becomes a drawer.
- Every link is relative, so the site works at `/<repo>/`, on a custom domain, and locally.

### Customising

- Colours, fonts and spacing: `internal/site/static/style.css`. Every colour is a CSS
  variable at the top, with a dark-mode block below it.
- Layout and markup: `internal/site/templates/*.html` (Go `html/template`).
  `layout.html` holds the header, footer, search dialog and sidebar.
- Behaviour: `internal/site/static/app.js`.

Templates and static files are embedded in the binary, so run `make build` after
editing them.

---

## 10. Tests

```sh
make test                 # = go test ./...
go test ./tests -v        # the content gate, showing warnings too
make check                # vet + gofmt + tests + build + scan: everything CI runs
```

| Package | Covers |
| --- | --- |
| `internal/convert` | every formatting rule in § 7, escaping, merging of split bold runs, image extraction |
| `internal/scan` | each rule fires (HIGH vs WARN), clean text stays clean, allowlist (fp, match, path, rule), inline allow, ignore globs, binaries, redaction, rule scoping |
| `internal/notes` | full export lifecycle with a fake Notes source: add, incremental no-op, rename, delete, image dedupe and cleanup, `#private`, excluded folders are never read, the mass-delete guard, hand-written files kept |
| `internal/config` | exclusion is case- and space-insensitive, validation, unknown keys, slugs |
| `internal/site` | pages and assets generated, natural sort, nested relative paths, TOC, `.md` links rewritten, `_index.md`, 404 base URL, refuses to wipe a foreign output dir |
| `internal/frontmatter` | header round-trip, quoting, BOM/CRLF |
| `tests` (gate) | on the **real** `content/`: no HIGH secrets, nothing from excluded folders, every note has a title, every image exists, no broken links in the built site |

Test secrets are assembled at runtime (`"AKIA" + "…"`) so the test files never contain
something that GitHub push protection or other scanners would flag. `*_test.go` and
`testdata/` are also in `scan.ignore_paths`.

---

## 11. CI/CD: GitHub Actions and Pages

`.github/workflows/pages.yml` runs on every push to `main`, on pull requests, and on
manual dispatch:

1. `go vet` + `gofmt` check
2. `go test ./... -v` (unit tests and the content gate)
3. build `notesync`
4. `notesync scan --tracked --format github`. Findings show up as annotations on the files.
5. `notesync build`
6. upload `site/` and deploy to GitHub Pages (only on `main`, and only if steps 1–5 passed)

Pull requests run steps 1–5 without deploying.

---

## 12. Project layout

```
.
├── cmd/notesync/           CLI (main.go: every command; flags.go)
├── internal/
│   ├── config/             notesync.json, exclusion rules, slugs
│   ├── convert/            Apple Notes HTML → Markdown
│   ├── frontmatter/        note header parse/render
│   ├── gitops/             git helpers
│   ├── notes/              notes.js (JXA bridge), client.go (osascript), export.go
│   ├── scan/               rules.go (rule set), scan.go (engine, allowlist, output)
│   └── site/               build.go, templates/, static/ (style.css, app.js, favicon)
├── tests/                  content gate
├── content/                exported notes (committed)
├── .github/workflows/      pages.yml
├── notesync.json           configuration
├── .secretsallow           scanner allowlist
├── Makefile
├── README.md / GUIDE.md    docs (keep them current)
└── CLAUDE.md               local Claude Code instructions (git-ignored)
```

Git-ignored: `bin/`, `site/`, `CLAUDE.md`, `.claude/`, `.notesync-cache.json`.

---

## 13. Extending and changing things

### Publish another folder

Add its name to `source.folders`, and optionally a `sections` entry with a title,
description and order. Then run `notesync folders`, `notesync export` and `notesync publish`.
Never add a folder that is in `exclude_folders`.

### Add a scanner rule

Add a `Rule` to `DefaultRules()` in `internal/scan/rules.go`. Put more specific rules
earlier, because the first rule to claim some characters wins. Use `Group` to point at the
secret part, `Refine` to grade it, and `ContentOnly` for personal or infrastructure details.
Add positive and negative cases to `TestRulesDetect` / `TestCleanTextHasNoFindings`, and add
a row to the rules table in § 8.

### Change how notes convert

Edit `internal/convert/html2md.go` and add a case to `TestToMarkdown`, using the real Notes
HTML. To get that HTML, run a JXA snippet that returns `note.body()` for a note in a
*published* folder. Then run `notesync export --force` to re-convert everything.

### Feature checklist

Every change or new feature must:

- [ ] include tests
- [ ] update **README.md** (if user-facing) and **GUIDE.md** (always)
- [ ] pass `make check`

---

## 14. Troubleshooting

| Symptom | Fix |
| --- | --- |
| `macOS denied access to Notes` | System Settings → Privacy & Security → Automation → your terminal → enable **Notes** |
| `folder "x" was not found` | check the spelling in `notesync.json`; `notesync folders` shows the exact names |
| export takes long the first time | images are large; later runs are incremental |
| a page looks wrong after changing the converter | `notesync export --force` |
| commit refused by the hook | read the findings; fix the note or allowlist a false positive; `make build` if the binary is missing |
| `refusing to clear site` | `paths.output` points at a folder notesync didn't create; pick another folder or empty it |
| Pages shows the old site | check the Actions tab; the deploy only runs when every check passes |
| GitHub icon missing | set `site.repo_url` |

---

## 15. Design decisions

- **Go, single binary, two dependencies** (goldmark for Markdown, x/net/html for parsing).
  It builds in seconds and runs the same locally and in CI.
- **JXA through `osascript`**, not reading the Notes SQLite database. That is the supported
  automation interface, needs no Full Disk Access, and is read-only by construction.
- **An allowlist of folders plus a deny list**: a folder is published only if you name it,
  and the deny list wins even if you name it by mistake. Exclusion is enforced in the JXA
  script, again in Go, and again in the gate test.
- **Markdown committed to git**, not HTML. Diffs are readable, history is useful, and CI can
  rebuild the site.
- **Content-addressed images** (`sha256` names). Identical screenshots are stored once, and
  unchanged images never create diffs.
- **Defence in depth for secrets.** Each layer can be skipped locally, but CI cannot be,
  and Pages only deploys from CI.
