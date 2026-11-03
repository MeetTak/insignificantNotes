// Package notes talks to Apple Notes (through osascript + JXA) and exports
// the configured folders into Markdown files under content/.
package notes

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

//go:embed notes.js
var jxaSource string

// NoteMeta is the cheap listing data for one note (no body).
type NoteMeta struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Folder   []string `json:"folder"`
	Created  string   `json:"created"`
	Modified string   `json:"modified"`
	Locked   bool     `json:"locked"`
}

type Skip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type Listing struct {
	Notes   []NoteMeta `json:"notes"`
	Skipped []Skip     `json:"skipped"`
	Found   []string   `json:"found"`
}

type FolderInfo struct {
	Account string `json:"account"`
	Name    string `json:"name"`
	Parent  string `json:"parent"`
}

// Source is what the exporter needs from Apple Notes; tests swap in a fake.
type Source interface {
	List(ctx context.Context) (*Listing, error)
	Body(ctx context.Context, id string) (string, error)
}

// Client runs the embedded JXA script with osascript.
type Client struct {
	Account string
	Include []string
	Exclude []string

	once       sync.Once
	scriptPath string
	scriptErr  error
}

// ErrNotAuthorized means macOS blocked automation of Notes.
var ErrNotAuthorized = errors.New("macOS denied access to Notes: allow it in System Settings → Privacy & Security → Automation (enable Notes for your terminal app), then retry")

func (c *Client) script() (string, error) {
	c.once.Do(func() {
		f, err := os.CreateTemp("", "notesync-*.js")
		if err != nil {
			c.scriptErr = err
			return
		}
		defer f.Close()
		if _, err := f.WriteString(jxaSource); err != nil {
			c.scriptErr = err
			return
		}
		c.scriptPath = f.Name()
	})
	return c.scriptPath, c.scriptErr
}

// Close removes the temporary script file.
func (c *Client) Close() {
	if c.scriptPath != "" {
		os.Remove(c.scriptPath)
	}
}

func (c *Client) call(ctx context.Context, req map[string]any, out any) error {
	path, err := c.script()
	if err != nil {
		return err
	}
	arg, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("osascript"); err != nil {
		return errors.New("osascript not found: exporting from Apple Notes only works on macOS")
	}
	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", path, string(arg))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "-1743") || strings.Contains(strings.ToLower(msg), "not authorized") {
			return ErrNotAuthorized
		}
		return fmt.Errorf("osascript %s: %v: %s", req["mode"], err, msg)
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout), out); err != nil {
		return fmt.Errorf("decode osascript %s output: %w", req["mode"], err)
	}
	return nil
}

func (c *Client) List(ctx context.Context) (*Listing, error) {
	var l Listing
	err := c.call(ctx, map[string]any{"mode": "list", "account": c.Account, "include": c.Include, "exclude": c.Exclude}, &l)
	return &l, err
}

func (c *Client) Body(ctx context.Context, id string) (string, error) {
	var r struct {
		Body string `json:"body"`
	}
	err := c.call(ctx, map[string]any{"mode": "get", "id": id}, &r)
	return r.Body, err
}

// Folders lists folder names (never contents) across all accounts.
func (c *Client) Folders(ctx context.Context) ([]FolderInfo, error) {
	var f []FolderInfo
	err := c.call(ctx, map[string]any{"mode": "folders"}, &f)
	return f, err
}
