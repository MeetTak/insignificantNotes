// Package gitops wraps the handful of git commands notesync needs.
package gitops

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Run executes git in dir and returns trimmed stdout.
func Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// Stream executes git with output going straight to the terminal.
func Stream(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}

func IsRepo(dir string) bool {
	_, err := Run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

func HooksDir(dir string) (string, error) {
	p, err := Run(dir, "rev-parse", "--git-path", "hooks")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	return p, nil
}

func splitZ(s string) []string {
	var out []string
	for _, p := range strings.Split(s, "\x00") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// StagedFiles lists files added/changed in the index.
func StagedFiles(dir string) ([]string, error) {
	cmd := exec.Command("git", "-C", dir, "diff", "--cached", "--name-only", "-z", "--diff-filter=ACMR")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --cached: %w", err)
	}
	return splitZ(string(out)), nil
}

// StagedContent returns the staged (index) version of a file.
func StagedContent(dir, path string) ([]byte, error) {
	return exec.Command("git", "-C", dir, "show", ":"+path).Output()
}

// TrackedFiles lists every file git tracks.
func TrackedFiles(dir string) ([]string, error) {
	out, err := exec.Command("git", "-C", dir, "ls-files", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	return splitZ(string(out)), nil
}

func HasStagedChanges(dir string) bool {
	err := exec.Command("git", "-C", dir, "diff", "--cached", "--quiet").Run()
	return err != nil
}

func Remote(dir, name string) string {
	u, _ := Run(dir, "remote", "get-url", name)
	return u
}

func HasUpstream(dir string) bool {
	_, err := Run(dir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	return err == nil
}

// Branch returns the current branch name (works before the first commit too).
func Branch(dir string) string {
	if b, err := Run(dir, "symbolic-ref", "--short", "HEAD"); err == nil {
		return b
	}
	b, _ := Run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	return b
}
