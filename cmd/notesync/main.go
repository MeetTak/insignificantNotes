// Command notesync publishes selected Apple Notes folders as a static docs
// site on GitHub Pages: export → secret scan → build → commit → push.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	_ "time/tzdata" // site.timezone works on any machine

	"insignificantnotes/internal/config"
	"insignificantnotes/internal/gitops"
	"insignificantnotes/internal/notes"
	"insignificantnotes/internal/scan"
	"insignificantnotes/internal/site"
)

var version = "dev"

const usage = `notesync: publish Apple Notes folders as a docs site on GitHub Pages

Usage:
  notesync <command> [flags]

Commands:
  publish   export → scan → build → commit → push (stops at the first problem)
  export    pull the configured Apple Notes folders into content/ (incremental)
  scan      check files for API keys, tokens, passwords and personal details
  build     render content/ into site/
  serve     build, serve on localhost, rebuild + live-reload on changes
  init      set up git and install the pre-commit / pre-push scan hooks
  doctor    check Notes access, git, hooks and config
  folders   list Apple Notes folder names and whether each is published
  version   print the version

Run "notesync <command> -h" for flags. Full guide: GUIDE.md
`

type exitCode int

func (e exitCode) Error() string { return "exit " + strconv.Itoa(int(e)) }

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	cmds := map[string]func([]string) error{
		"publish": cmdPublish, "export": cmdExport, "scan": cmdScan, "build": cmdBuild,
		"serve": cmdServe, "init": cmdInit, "doctor": cmdDoctor, "folders": cmdFolders,
	}
	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return
	case "version", "--version":
		fmt.Println("notesync", version)
		return
	}
	run, ok := cmds[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err := run(args); err != nil {
		var ec exitCode
		if errors.As(err, &ec) {
			os.Exit(int(ec))
		}
		fmt.Fprintln(os.Stderr, red("error: ")+err.Error())
		os.Exit(1)
	}
}

// ------------------------------------------------------------------ output

var useColor = func() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}()

func paint(code, s string) string {
	if !useColor {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}
func red(s string) string    { return paint("31;1", s) }
func yellow(s string) string { return paint("33;1", s) }
func green(s string) string  { return paint("32;1", s) }
func bold(s string) string   { return paint("1", s) }
func dim(s string) string    { return paint("2", s) }

func step(n, total int, msg string) {
	fmt.Printf("\n%s %s\n", bold(fmt.Sprintf("[%d/%d]", n, total)), msg)
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(".")
	if err != nil {
		return nil, fmt.Errorf("%w\nrun notesync from inside the repository", err)
	}
	return cfg, nil
}

func newClient(cfg *config.Config) *notes.Client {
	return &notes.Client{Account: cfg.Source.Account, Include: cfg.Source.Folders, Exclude: cfg.ExcludeList()}
}

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

// ------------------------------------------------------------------ export

func cmdExport(args []string) error {
	fl := newFlags("export", "Pull the configured Apple Notes folders into content/ as Markdown.")
	force := fl.Bool("force", false, "re-export every note, even unchanged ones")
	dry := fl.Bool("dry-run", false, "show what would change without writing anything")
	fl.Parse(args)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	_, err = runExport(cfg, *force, *dry)
	return err
}

func runExport(cfg *config.Config, force, dry bool) (*notes.Report, error) {
	ctx, cancel := signalContext()
	defer cancel()
	client := newClient(cfg)
	defer client.Close()
	fmt.Printf("  folders: %s   %s\n", strings.Join(cfg.Source.Folders, ", "), dim("(never read: "+strings.Join(cfg.ExcludeList(), ", ")+")"))
	ex := &notes.Exporter{Cfg: cfg, Source: client, Force: force, DryRun: dry,
		Progress: func(done, total int, title string) {
			if isTTY(os.Stdout) {
				fmt.Printf("\r\x1b[K  fetching %d/%d  %s", done, total, title)
			}
		}}
	start := time.Now()
	rep, err := ex.Run(ctx)
	if isTTY(os.Stdout) {
		fmt.Print("\r\x1b[K")
	}
	if err != nil {
		return nil, err
	}
	for _, c := range rep.Changes {
		var tag string
		switch c.Action {
		case "add":
			tag = green("+ add   ")
		case "update":
			tag = yellow("~ update")
		case "rename":
			tag = yellow("→ rename")
		case "remove":
			tag = red("- remove")
		default:
			tag = dim("  skip  ")
		}
		line := fmt.Sprintf("  %s %s", tag, c.Title)
		if c.Path != "" {
			line += dim("  " + filepath.ToSlash(filepath.Join(cfg.Paths.Content, c.Path)))
		}
		if c.Detail != "" {
			line += dim("  (" + c.Detail + ")")
		}
		fmt.Println(line)
	}
	for _, w := range rep.Warnings {
		fmt.Println("  " + yellow("warning: ") + w)
	}
	verb := "exported"
	if dry {
		verb = "would export"
	}
	fmt.Printf("  %s %d added, %d updated, %d renamed, %d removed, %d unchanged %s\n", verb,
		rep.Count("add"), rep.Count("update"), rep.Count("rename"), rep.Count("remove"), rep.Unchanged,
		dim(fmt.Sprintf("in %.1fs", time.Since(start).Seconds())))
	return rep, nil
}

// ------------------------------------------------------------------ scan

func newScanner(cfg *config.Config) (*scan.Scanner, error) {
	allow, err := scan.LoadAllowlist(cfg.AllowlistPath())
	if err != nil {
		return nil, err
	}
	ignore := append([]string{}, cfg.Scan.IgnorePaths...)
	ignore = append(ignore, filepath.ToSlash(cfg.Paths.Output)+"/**", "bin/**", ".git/**", cfg.Scan.Allowlist)
	return scan.New(allow, ignore, cfg.Scan.KeywordPaths), nil
}

var skipDirs = map[string]bool{".git": true, "node_modules": true, "bin": true, ".claude": true}

// scanWorkingTree scans files on disk under the given repo-relative roots.
func scanWorkingTree(cfg *config.Config, sc *scan.Scanner, roots []string) ([]scan.Finding, int, error) {
	var out []scan.Finding
	files := 0
	out2 := filepath.Clean(cfg.Paths.Output)
	for _, r := range roots {
		abs := filepath.Join(cfg.Root, r)
		err := filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			rel, _ := filepath.Rel(cfg.Root, p)
			if d.IsDir() {
				if p != abs && (skipDirs[d.Name()] || rel == out2) {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() || sc.Ignored(rel) {
				return nil
			}
			if fi, err := d.Info(); err != nil || fi.Size() > 8<<20 {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			files++
			out = append(out, sc.ScanContent(rel, data)...)
			return nil
		})
		if err != nil {
			return nil, files, err
		}
	}
	return out, files, nil
}

func cmdScan(args []string) error {
	fl := newFlags("scan", "Scan for secrets. Exit status 1 when a HIGH finding (or, with --strict, any finding) exists.\n\nWith no paths, scans the whole working tree.")
	staged := fl.Bool("staged", false, "scan the staged (about to be committed) version of changed files")
	tracked := fl.Bool("tracked", false, "scan every file tracked by git")
	strict := fl.Bool("strict", false, "fail on warnings too")
	format := fl.String("format", "text", "output format: text or github (workflow annotations)")
	accept := fl.Bool("accept-warnings", false, "append every current WARN fingerprint to the allowlist")
	quiet := fl.Bool("quiet", false, "only print findings and the summary line")
	rule := fl.String("rule", "", "only report (and accept) findings from this rule id, e.g. sensitive-keyword")
	fl.Parse(args)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	sc, err := newScanner(cfg)
	if err != nil {
		return err
	}
	var findings []scan.Finding
	files := 0
	switch {
	case *staged:
		paths, err := gitops.StagedFiles(cfg.Root)
		if err != nil {
			return err
		}
		for _, p := range paths {
			data, err := gitops.StagedContent(cfg.Root, p)
			if err != nil {
				continue
			}
			files++
			findings = append(findings, sc.ScanContent(p, data)...)
		}
	case *tracked:
		paths, err := gitops.TrackedFiles(cfg.Root)
		if err != nil {
			return err
		}
		for _, p := range paths {
			data, err := os.ReadFile(filepath.Join(cfg.Root, p))
			if err != nil {
				continue
			}
			files++
			findings = append(findings, sc.ScanContent(p, data)...)
		}
	default:
		roots := []string{"."}
		if fl.NArg() > 0 {
			roots = nil
			for _, a := range fl.Args() {
				abs, _ := filepath.Abs(a)
				rel, err := filepath.Rel(cfg.Root, abs)
				if err != nil || strings.HasPrefix(rel, "..") {
					return fmt.Errorf("%s is outside the repository", a)
				}
				roots = append(roots, rel)
			}
		}
		findings, files, err = scanWorkingTree(cfg, sc, roots)
		if err != nil {
			return err
		}
	}
	if *rule != "" {
		var kept []scan.Finding
		for _, f := range findings {
			if f.RuleID == *rule {
				kept = append(kept, f)
			}
		}
		findings = kept
	}
	return reportScan(cfg, findings, files, *format, *strict || cfg.Scan.FailOnWarnings, *accept, *quiet)
}

func reportScan(cfg *config.Config, findings []scan.Finding, files int, format string, strict, accept, quiet bool) error {
	scan.Sort(findings)
	high, warn := scan.Count(findings)
	if format == "github" {
		scan.WriteGitHub(os.Stdout, findings)
	} else {
		scan.WriteText(os.Stdout, findings, useColor)
	}
	if accept && warn > 0 {
		n, err := scan.AcceptWarnings(cfg.AllowlistPath(), findings)
		if err != nil {
			return err
		}
		fmt.Printf("accepted %d warning(s) into %s\n", n, cfg.Scan.Allowlist)
		warn = 0
	}
	summary := fmt.Sprintf("secret scan: %d file(s), %d high, %d warning(s)", files, high, warn)
	switch {
	case high > 0:
		fmt.Println(red("✗ ") + summary)
		if !quiet {
			fmt.Println(dim("  HIGH findings block commits, pushes and deploys. Remove the secret from the note (then rotate it),"))
			fmt.Println(dim("  or, if it is a false positive, add its fp: line to " + cfg.Scan.Allowlist + "."))
		}
		return exitCode(1)
	case warn > 0 && strict:
		fmt.Println(red("✗ ") + summary + " (strict)")
		return exitCode(1)
	case warn > 0:
		fmt.Println(yellow("! ") + summary)
		if !quiet {
			fmt.Println(dim("  Review the warnings above. `notesync scan --accept-warnings` stops re-reporting them."))
		}
	default:
		fmt.Println(green("✓ ") + summary)
	}
	return nil
}

// ------------------------------------------------------------------ build / serve

func cmdBuild(args []string) error {
	fl := newFlags("build", "Render content/ into the static site directory.")
	fl.Parse(args)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	return runBuild(cfg)
}

func runBuild(cfg *config.Config) error {
	start := time.Now()
	res, err := site.Build(cfg)
	if err != nil {
		return err
	}
	for _, w := range res.Warnings {
		fmt.Println("  " + yellow("warning: ") + w)
	}
	fmt.Printf("  built %d pages (%d notes in %d sections, %d assets) → %s %s\n", res.Pages, res.Notes, res.Sections,
		res.Assets, cfg.Paths.Output+"/", dim(fmt.Sprintf("in %dms", time.Since(start).Milliseconds())))
	return nil
}

func cmdServe(args []string) error {
	fl := newFlags("serve", "Build the site, serve it locally and rebuild when content/ changes.")
	port := fl.Int("port", 8080, "port to listen on")
	noWatch := fl.Bool("no-watch", false, "do not rebuild on changes")
	fl.Parse(args)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if err := runBuild(cfg); err != nil {
		return err
	}
	var buildID atomic.Int64
	buildID.Store(time.Now().UnixNano())
	if !*noWatch {
		go func() {
			last := contentSignature(cfg.ContentDir())
			for range time.Tick(time.Second) {
				if sig := contentSignature(cfg.ContentDir()); sig != last {
					last = sig
					fmt.Println(dim("  change detected, rebuilding…"))
					if err := runBuild(cfg); err != nil {
						fmt.Println("  " + red("build failed: ") + err.Error())
						continue
					}
					buildID.Store(time.Now().UnixNano())
				}
			}
		}()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/__notesync/build", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprint(w, buildID.Load())
	})
	files := http.FileServer(http.Dir(cfg.OutputDir()))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	}))
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(*port))
	if err != nil {
		return err
	}
	fmt.Printf("\n  serving %s  %s\n", bold("http://localhost:"+strconv.Itoa(*port)+"/"), dim("(ctrl+c to stop)"))
	return http.Serve(ln, mux)
}

func contentSignature(dir string) string {
	var n int
	var size, latest int64
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if fi, err := d.Info(); err == nil {
			n++
			size += fi.Size()
			if m := fi.ModTime().UnixNano(); m > latest {
				latest = m
			}
		}
		return nil
	})
	return fmt.Sprintf("%d/%d/%d", n, size, latest)
}

// ------------------------------------------------------------------ publish

func cmdPublish(args []string) error {
	fl := newFlags("publish", "Export from Apple Notes, scan for secrets, build, commit and push.\nGitHub Actions then re-runs tests + the scan and deploys GitHub Pages.")
	msg := fl.String("m", "", "commit message (default: generated from the changes)")
	yes := fl.Bool("yes", false, "publish even if the scan has warnings (HIGH findings still block)")
	noExport := fl.Bool("no-export", false, "skip the Apple Notes export (publish what is in content/)")
	noPush := fl.Bool("no-push", false, "commit but do not push")
	dry := fl.Bool("dry-run", false, "export (dry-run), scan and build, but do not write notes, commit or push")
	force := fl.Bool("force", false, "re-export every note")
	all := fl.Bool("all", false, "commit every change in the repo, not only content/")
	fl.Parse(args)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if !*dry && !gitops.IsRepo(cfg.Root) {
		return errors.New("not a git repository yet: run `notesync init` first")
	}
	const total = 5

	var rep *notes.Report
	step(1, total, "Export from Apple Notes")
	if *noExport {
		fmt.Println(dim("  skipped (--no-export)"))
	} else if rep, err = runExport(cfg, *force, *dry); err != nil {
		return err
	}

	step(2, total, "Secret scan")
	sc, err := newScanner(cfg)
	if err != nil {
		return err
	}
	roots := []string{cfg.Paths.Content}
	if *all {
		roots = []string{"."}
	}
	findings, files, err := scanWorkingTree(cfg, sc, roots)
	if err != nil {
		return err
	}
	scan.Sort(findings)
	high, warn := scan.Count(findings)
	scan.WriteText(os.Stdout, findings, useColor)
	if high > 0 {
		return reportScan(cfg, findings, files, "text", false, false, false)
	}
	if warn > 0 && !*yes && !*dry {
		fmt.Printf("%s %d warning(s) above. Nothing is published until you decide.\n", yellow("!"), warn)
		if !isTTY(os.Stdin) {
			return errors.New("warnings need review: re-run in a terminal, or pass --yes to publish anyway")
		}
		switch ask("  [y] publish anyway  [a] accept them permanently and publish  [N] stop: ") {
		case "y", "yes":
		case "a", "accept":
			n, err := scan.AcceptWarnings(cfg.AllowlistPath(), findings)
			if err != nil {
				return err
			}
			fmt.Printf("  accepted %d warning(s) into %s\n", n, cfg.Scan.Allowlist)
		default:
			return errors.New("stopped before publishing; fix the notes or accept the warnings, then run publish again")
		}
	} else {
		fmt.Printf("  %s %d file(s), %d high, %d warning(s)\n", green("✓"), files, high, warn)
	}

	step(3, total, "Build site")
	if err := runBuild(cfg); err != nil {
		return err
	}
	if *dry {
		fmt.Println("\n" + green("Dry run complete.") + " Nothing was written to content/, committed or pushed. Preview with `notesync serve`.")
		return nil
	}

	step(4, total, "Commit")
	addArgs := []string{"add", "-A", "--"}
	if *all {
		addArgs = append(addArgs, ".")
	} else {
		for _, p := range []string{cfg.Paths.Content, cfg.Scan.Allowlist, config.FileName} {
			if _, err := os.Stat(filepath.Join(cfg.Root, p)); err == nil {
				addArgs = append(addArgs, p)
			}
		}
	}
	if _, err := gitops.Run(cfg.Root, addArgs...); err != nil {
		return err
	}
	if !gitops.HasStagedChanges(cfg.Root) {
		fmt.Println(dim("  nothing new to commit"))
	} else {
		message := *msg
		if message == "" {
			message = commitMessage(rep)
		}
		if err := gitops.Stream(cfg.Root, "commit", "-q", "-m", message); err != nil {
			return fmt.Errorf("git commit failed (the pre-commit secret scan may have blocked it): %w", err)
		}
		fmt.Println("  " + green("✓") + " committed: " + strings.SplitN(message, "\n", 2)[0])
	}

	step(5, total, "Push")
	if *noPush {
		fmt.Println(dim("  skipped (--no-push)"))
		return nil
	}
	if gitops.Remote(cfg.Root, "origin") == "" {
		fmt.Println(yellow("  no `origin` remote yet.") + " Create the GitHub repo, then:")
		fmt.Println("    git remote add origin git@github.com:<you>/<repo>.git && notesync publish --no-export")
		return nil
	}
	pushArgs := []string{"push"}
	if !gitops.HasUpstream(cfg.Root) {
		pushArgs = []string{"push", "-u", "origin", "HEAD"}
	}
	if err := gitops.Stream(cfg.Root, pushArgs...); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}
	fmt.Println("  " + green("✓") + " pushed. GitHub Actions now runs the tests + secret scan and deploys Pages.")
	return nil
}

func commitMessage(rep *notes.Report) string {
	subject := "notes: sync from Apple Notes"
	if rep == nil || !rep.Written() {
		return subject
	}
	var b strings.Builder
	b.WriteString(subject)
	b.WriteString(fmt.Sprintf(" (+%d ~%d -%d)\n\n", rep.Count("add"), rep.Count("update")+rep.Count("rename"), rep.Count("remove")))
	n := 0
	for _, c := range rep.Changes {
		if c.Action == "skip" {
			continue
		}
		if n == 40 {
			b.WriteString("- …\n")
			break
		}
		b.WriteString(fmt.Sprintf("- %s: %s\n", c.Action, c.Title))
		n++
	}
	return strings.TrimRight(b.String(), "\n")
}

func ask(prompt string) string {
	fmt.Print(prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.ToLower(strings.TrimSpace(line))
}

// ------------------------------------------------------------------ init / doctor / folders

const hookMarker = "notesync-hook"

func hookScript(kind, scanArgs string) string {
	return `#!/bin/sh
# ` + hookMarker + `: ` + kind + ` (installed by "notesync init"; re-run it to update)
# Blocks the ` + kind + ` when the secret scan finds a HIGH-severity secret.
root="$(git rev-parse --show-toplevel)"
bin="$root/bin/notesync"
[ -x "$bin" ] || bin="$(command -v notesync 2>/dev/null)"
if [ -z "$bin" ]; then
  echo "notesync: binary not found. Run 'make build' first (or bypass once with --no-verify)." >&2
  exit 1
fi
cd "$root" && exec "$bin" scan ` + scanArgs + ` --quiet
`
}

func cmdInit(args []string) error {
	fl := newFlags("init", "Initialise git (if needed) and install the pre-commit and pre-push secret-scan hooks.")
	force := fl.Bool("force", false, "overwrite existing hooks that were not written by notesync")
	fl.Parse(args)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if !gitops.IsRepo(cfg.Root) {
		if _, err := gitops.Run(cfg.Root, "init", "-b", "main"); err != nil {
			return err
		}
		fmt.Println(green("✓") + " initialised git repository (branch main)")
	}
	hooks, err := gitops.HooksDir(cfg.Root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		return err
	}
	for kind, scanArgs := range map[string]string{"pre-commit": "--staged", "pre-push": "--tracked"} {
		p := filepath.Join(hooks, kind)
		if old, err := os.ReadFile(p); err == nil && !strings.Contains(string(old), hookMarker) && !*force {
			fmt.Printf("%s %s exists and was not written by notesync; left alone (use --force to replace)\n", yellow("!"), p)
			continue
		}
		if err := os.WriteFile(p, []byte(hookScript(kind, scanArgs)), 0o755); err != nil {
			return err
		}
		fmt.Printf("%s installed %s hook (notesync scan %s)\n", green("✓"), kind, scanArgs)
	}
	if _, err := os.Stat(cfg.AllowlistPath()); os.IsNotExist(err) {
		os.WriteFile(cfg.AllowlistPath(), []byte("# notesync secret-scan allowlist; see GUIDE.md\n"), 0o644)
		fmt.Printf("%s created %s\n", green("✓"), cfg.Scan.Allowlist)
	}
	fmt.Println("\nNext: `notesync doctor`, then `notesync publish --dry-run`.")
	return nil
}

func cmdDoctor(args []string) error {
	fl := newFlags("doctor", "Check that everything notesync needs is in place.")
	fl.Parse(args)
	ok, warnN, bad := green("✓"), yellow("!"), red("✗")
	failed := false
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println(bad, "config:", err)
		return exitCode(1)
	}
	fmt.Println(ok, "config", filepath.Join(cfg.Root, config.FileName))
	fmt.Printf("  publishing %s; never reading %s\n", strings.Join(cfg.Source.Folders, ", "), strings.Join(cfg.ExcludeList(), ", "))

	if _, err := exec.LookPath("osascript"); err != nil {
		fmt.Println(warnN, "osascript not found: export only works on macOS (build/scan still work)")
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		client := newClient(cfg)
		folders, err := client.Folders(ctx)
		client.Close()
		cancel()
		if err != nil {
			fmt.Println(bad, "Apple Notes:", err)
			failed = true
		} else {
			fmt.Printf("%s Apple Notes reachable (%d folders)\n", ok, len(folders))
			for _, want := range cfg.Source.Folders {
				found := false
				for _, f := range folders {
					if f.Parent == "" && config.Norm(f.Name) == config.Norm(want) && (cfg.Source.Account == "" || f.Account == cfg.Source.Account) {
						found = true
					}
				}
				if found {
					fmt.Printf("  %s folder %q\n", ok, want)
				} else {
					fmt.Printf("  %s folder %q not found at the top level of %q\n", bad, want, cfg.Source.Account)
					failed = true
				}
			}
		}
	}
	if _, err := exec.LookPath("sips"); err != nil {
		fmt.Println(warnN, "sips not found: images will not be resized")
	}
	if !gitops.IsRepo(cfg.Root) {
		fmt.Println(bad, "git: not a repository (run `notesync init`)")
		failed = true
	} else {
		fmt.Printf("%s git repository on branch %s\n", ok, gitops.Branch(cfg.Root))
		if r := gitops.Remote(cfg.Root, "origin"); r == "" {
			fmt.Println(warnN, "no `origin` remote; publish will commit locally only")
		} else {
			fmt.Println(ok, "origin", r)
		}
		hooks, _ := gitops.HooksDir(cfg.Root)
		for _, h := range []string{"pre-commit", "pre-push"} {
			data, err := os.ReadFile(filepath.Join(hooks, h))
			if err == nil && strings.Contains(string(data), hookMarker) {
				fmt.Println(ok, h, "hook installed")
			} else {
				fmt.Println(bad, h, "hook missing (run `notesync init`)")
				failed = true
			}
		}
		if out, _ := gitops.Run(cfg.Root, "check-ignore", "CLAUDE.md"); out == "" {
			fmt.Println(warnN, "CLAUDE.md is not git-ignored")
		}
	}
	if exe, err := os.Executable(); err == nil {
		fmt.Println(dim("  binary: " + exe + " (" + version + ")"))
	}
	if failed {
		return exitCode(1)
	}
	return nil
}

func cmdFolders(args []string) error {
	fl := newFlags("folders", "List Apple Notes folder names (never their contents) and whether each is published.")
	fl.Parse(args)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client := newClient(cfg)
	defer client.Close()
	folders, err := client.Folders(ctx)
	if err != nil {
		return err
	}
	byName := map[string]notes.FolderInfo{}
	for _, f := range folders {
		byName[f.Name] = f
	}
	// published reports whether a folder (walking up its parents) is published.
	var published func(f notes.FolderInfo, depth int) bool
	published = func(f notes.FolderInfo, depth int) bool {
		if cfg.IsExcluded(f.Name) || depth > 20 {
			return false
		}
		if f.Parent == "" {
			return cfg.IsIncluded(f.Name)
		}
		p, ok := byName[f.Parent]
		return ok && published(p, depth+1)
	}
	var insideExcluded func(f notes.FolderInfo, depth int) bool
	insideExcluded = func(f notes.FolderInfo, depth int) bool {
		p, ok := byName[f.Parent]
		if f.Parent == "" || !ok || depth > 20 {
			return false
		}
		return cfg.IsExcluded(p.Name) || insideExcluded(p, depth+1)
	}
	for _, f := range folders {
		if cfg.Source.Account != "" && f.Account != cfg.Source.Account {
			continue
		}
		name := f.Name
		if f.Parent != "" {
			name = f.Parent + "/" + f.Name
		}
		status := dim("not published")
		switch {
		case cfg.IsExcluded(f.Name):
			status = red("excluded (never read)")
		case insideExcluded(f, 0):
			status = red("inside an excluded folder (never read)")
		case published(f, 0):
			status = green("published")
		}
		fmt.Printf("  %-32q %s\n", name, status)
	}
	return nil
}
