# insignificantNotes

My notes on AWS, cloud/DevOps and Java, written in **Apple Notes** and published as a
documentation site on **GitHub Pages**.

The tool that does this is **`notesync`**, a single Go binary in this repo:

```
Apple Notes ──export──▶ content/*.md ──scan──▶ build ──▶ site/ ──git push──▶ GitHub Actions ──▶ GitHub Pages
 (aws, cloud, Java)      (Markdown +          (secret      (static HTML,        (tests + scan       (live site)
                          images)              gate)        search, TOC)         again, deploy)
```

One command takes a note from Apple Notes to the live site:

```sh
notesync publish
```

## Quick start

```sh
make build            # compile bin/notesync
./bin/notesync init   # git init + install pre-commit / pre-push secret-scan hooks
./bin/notesync doctor # check Notes access, folders, git, hooks
make serve            # preview at http://localhost:8080 (live reload)
make publish          # export → scan → build → commit → push
```

`make install` copies the binary to `~/.local/bin` so you can run `notesync` from anywhere
inside the repo. First-time GitHub setup (creating the repo and turning on Pages) is in
[GUIDE.md § Setup](GUIDE.md#2-one-time-setup).

## What gets published

| Apple Notes folder | Published as |
| --- | --- |
| `aws` (and its sub-folder `course`) | `content/aws/` → `/aws/` |
| `cloud` | `content/cloud/` → `/cloud/` |
| `Java` | `content/java/` → `/java/` |
| `Other` / `Others` (at any depth, including `aws/others`), `Notes`, `acad`, `Recently Deleted` | **never read** |

You choose folders in [`notesync.json`](notesync.json). A note is never published if it is
locked or contains `#private` or `#nopublish`.

## Commands

| Command | What it does |
| --- | --- |
| `notesync publish` | export → secret scan → build → commit → push. Stops at the first problem. `--dry-run` writes nothing. |
| `notesync export` | Pulls the configured folders into `content/` as Markdown. Incremental: only changed notes are re-read. |
| `notesync scan` | Looks for API keys, tokens, passwords, private keys, public IPs, emails… HIGH findings block. |
| `notesync build` | Renders `content/` into `site/`. |
| `notesync serve` | Builds, serves on localhost and live-reloads when notes change. |
| `notesync init` | Runs `git init` if needed and installs the scan hooks. |
| `notesync doctor` | Health check. |
| `notesync folders` | Lists Apple Notes folder *names* (never contents) with their publish status. |

## Safety: secrets never go live

Five independent checks have to pass before anything reaches the public site:

1. `notesync publish` scans the notes before committing. HIGH findings stop it, and warnings need you to confirm.
2. The **pre-commit hook** scans exactly what is being committed.
3. The **pre-push hook** scans every tracked file.
4. **`go test ./...`** includes a content gate (`tests/`) that fails on secrets, on files from excluded folders, and on broken links.
5. **GitHub Actions** re-runs vet, the tests and the scan, and only deploys when all of them pass.

## Docs

- **[GUIDE.md](GUIDE.md)**: the full guide. It covers architecture, setup, configuration, scanner rules, the allowlist, testing, CI, customising the site and troubleshooting.
- `CLAUDE.md`: local instructions for Claude Code (git-ignored).

## Layout

```
cmd/notesync/          CLI entry point (publish, export, scan, build, serve, init, doctor, folders)
internal/notes/        Apple Notes bridge (JXA via osascript) + incremental exporter
internal/convert/      Apple Notes HTML → Markdown
internal/scan/         secret scanner, rules, allowlist
internal/site/         static site generator, templates, CSS/JS
internal/config/       notesync.json loader + folder exclusion logic
internal/frontmatter/  note header read/write
internal/gitops/       git helpers
tests/                 publishing gate run against the real content/
content/               exported notes (Markdown + images), committed
.github/workflows/     CI: check & deploy GitHub Pages
```
