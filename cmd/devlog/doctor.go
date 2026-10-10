package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const supportedNTN = "0.23.19"

func executeNTN(args ...string) ([]byte, error) {
	return executeNTNWithTimeout(15*time.Second, args...)
}

func executeNTNWithTimeout(timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	path, err := exec.LookPath("ntn")
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr == nil {
			path, err = exec.LookPath(filepath.Join(home, ".local", "bin", "ntn"))
		}
		if err != nil {
			return nil, fmt.Errorf("ntn not found in PATH or ~/.local/bin; install it or add its directory to PATH")
		}
	}
	cmd := exec.CommandContext(ctx, path, args...)
	// Do not inherit stdin: ntn infers POST when stdin contains JSON.
	cmd.Stdin = nil
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	err = cmd.Run()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("ntn timed out; check connectivity and retry this read-only diagnosis")
	}
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("ntn not found in PATH; install it or add its directory to PATH")
		}
		// Never expose ntn output: errors may contain credentials or request details.
		return nil, fmt.Errorf("ntn failed; check ntn login, workspace, network and access permissions")
	}
	return out.Bytes(), nil
}

func diagnose(cfg configuration, out io.Writer, execute func(...string) ([]byte, error)) error {
	version, err := execute("--version")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(version)) != "ntn "+supportedNTN {
		return fmt.Errorf("unsupported ntn version; validated version is %s", supportedNTN)
	}
	fmt.Fprintf(out, "OK ntn %s\n", supportedNTN)
	me, err := execute("api", "v1/users/me", "-X", "GET")
	if err != nil {
		return err
	}
	var user struct {
		Object string `json:"object"`
		ID     string `json:"id"`
	}
	if json.Unmarshal(me, &user) != nil || user.Object != "user" || user.ID == "" {
		return fmt.Errorf("ntn authentication returned an invalid response; check ntn login")
	}
	fmt.Fprintln(out, "OK authentication")
	data, err := execute("api", "v1/data_sources/"+cfg.Notion.DataSourceID, "-X", "GET")
	if err != nil {
		return err
	}
	var source struct {
		Object     string `json:"object"`
		ID         string `json:"id"`
		InTrash    bool   `json:"in_trash"`
		Properties map[string]struct {
			Type        string `json:"type"`
			MultiSelect struct {
				Options []struct {
					Name string `json:"name"`
				} `json:"options"`
			} `json:"multi_select"`
		} `json:"properties"`
	}
	if json.Unmarshal(data, &source) != nil || source.Object != "data_source" || source.InTrash || !strings.EqualFold(strings.ReplaceAll(source.ID, "-", ""), strings.ReplaceAll(cfg.Notion.DataSourceID, "-", "")) {
		return fmt.Errorf("invalid or trashed data source response; check configured data source")
	}
	for _, field := range []struct{ name, kind string }{{cfg.Notion.TitleProperty, "title"}, {cfg.Notion.DateProperty, "date"}, {cfg.Notion.TagsProperty, "multi_select"}} {
		if source.Properties[field.name].Type != field.kind {
			return fmt.Errorf("configured %s property is missing or has an incompatible type", field.kind)
		}
	}
	for _, option := range source.Properties[cfg.Notion.TagsProperty].MultiSelect.Options {
		if option.Name == cfg.Notion.Tag {
			fmt.Fprintln(out, "OK data source, property types and tag\nRead-only checks passed; posting has not been tested.")
			return nil
		}
	}
	return fmt.Errorf("configured tag is missing; prepare the option in Notion or update local configuration")
}
