package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const validConfig = `[notion]
data_source_id = "22852bf0-d4c9-4740-9e13-6d5af38794fc"
title_property = "Name"
date_property = "Date"
tags_property = "Tags"
tag = "AI開発"
[defaults]
project = ""
`

func configFile(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConfigValidation(t *testing.T) {
	for _, data := range []string{
		strings.Replace(validConfig, "22852bf0-d4c9-4740-9e13-6d5af38794fc", "REPLACE_WITH_DATA_SOURCE_ID", 1),
		strings.Replace(validConfig, `tag = "AI開発"`, `tag = " "`, 1),
		strings.Replace(validConfig, `date_property = "Date"`, `date_property = "Name"`, 1),
		strings.Replace(validConfig, `project = ""`, `project = 123`, 1),
		validConfig + "token = \"secret-marker\"\n",
		validConfig + "malformed = \"secret-marker\n",
		strings.Replace(validConfig, `tag = "AI開発"`, "", 1),
	} {
		_, err := loadConfig(configFile(t, data))
		if err == nil {
			t.Fatal("invalid configuration accepted")
		}
		if strings.Contains(err.Error(), "secret-marker") {
			t.Fatal("configuration leaked")
		}
	}
	path := configFile(t, validConfig)
	t.Setenv("NOTION_API_TOKEN", "secret-marker")
	var out, errOut bytes.Buffer
	if code := run([]string{"config", "--config", path}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, &errOut)
	}
	if !strings.Contains(out.String(), path) || !strings.Contains(out.String(), "AI開発") || strings.Contains(out.String(), "secret-marker") {
		t.Fatalf("bad output: %s", &out)
	}
}

const goodSource = `{"object":"data_source","id":"22852bf0-d4c9-4740-9e13-6d5af38794fc","properties":{"Name":{"type":"title"},"Date":{"type":"date"},"Tags":{"type":"multi_select","multi_select":{"options":[{"name":"AI開発"}]}}}}`

func TestDiagnosis(t *testing.T) {
	cfg, err := loadConfig(configFile(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, version, user, source string
		failAt                      int
		wantError                   bool
	}{
		{"success", "ntn 0.23.19", `{"object":"user","id":"test"}`, goodSource, -1, false},
		{"version", "ntn 0.23.18", "", "", -1, true},
		{"auth", "ntn 0.23.19", `{"object":"error"}`, "", -1, true},
		{"json", "ntn 0.23.19", `{"object":"user","id":"test"}`, "bad", -1, true},
		{"type", "ntn 0.23.19", `{"object":"user","id":"test"}`, strings.Replace(goodSource, `"type":"date"`, `"type":"rich_text"`, 1), -1, true},
		{"tag", "ntn 0.23.19", `{"object":"user","id":"test"}`, strings.Replace(goodSource, "AI開発", "other", 1), -1, true},
		{"wrong source", "ntn 0.23.19", `{"object":"user","id":"test"}`, strings.Replace(goodSource, "22852bf0", "aaaaaaaa", 1), -1, true},
		{"ntn failure", "ntn 0.23.19", `{"object":"user","id":"test"}`, goodSource, 2, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			want := [][]string{{"--version"}, {"api", "v1/users/me", "-X", "GET"}, {"api", "v1/data_sources/" + cfg.Notion.DataSourceID, "-X", "GET"}}
			fake := func(args ...string) ([]byte, error) {
				if calls >= len(want) || !reflect.DeepEqual(args, want[calls]) {
					t.Fatalf("unexpected request: %v", args)
				}
				i := calls
				calls++
				if i == tt.failAt {
					return nil, errors.New("network unavailable")
				}
				return []byte([]string{tt.version, tt.user, tt.source}[i]), nil
			}
			var out bytes.Buffer
			err := diagnose(cfg, &out, fake)
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v", err)
			}
			if !tt.wantError && calls != 3 {
				t.Fatalf("calls = %d", calls)
			}
		})
	}
}

func TestNTNProcessBoundary(t *testing.T) {
	// A real substitute executable checks argument isolation, inherited auth and discarded output.
	dir := t.TempDir()
	path := filepath.Join(dir, "ntn")
	script := "#!/bin/sh\n[ \"$1\" = api ] || exit 1\n[ \"$2\" = 'v1/users/me' ] || exit 1\n[ \"$NOTION_API_TOKEN\" = secret-marker ] || exit 1\nprintf 'secret-marker' >&2\nexit 1\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("NOTION_API_TOKEN", "secret-marker")
	_, err := executeNTN("api", "v1/users/me", "-X", "GET")
	if err == nil || strings.Contains(err.Error(), "secret-marker") {
		t.Fatalf("unsafe error: %v", err)
	}
}

func TestNTNTimeout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ntn"), []byte("#!/bin/sh\nexec /bin/sleep 10\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	_, err := executeNTNWithTimeout(30*time.Millisecond, "--version")
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timeout: %v", err)
	}
}
