package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPagePayload(t *testing.T) {
	cfg, err := loadConfig(configFile(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("開", 2001) + `" $(secret)`
	entry := workLog{project: "Project", sections: make([]string, len(addSections))}
	entry.sections[0] = text
	payload, err := pagePayload(cfg, "2026-10-11 / Project", "2026-10-11", entry)
	if err != nil {
		t.Fatal(err)
	}
	var page struct {
		Parent     map[string]string          `json:"parent"`
		Properties map[string]json.RawMessage `json:"properties"`
		Children   []struct {
			Type      string `json:"type"`
			Paragraph struct {
				RichText []struct {
					Text struct {
						Content string `json:"content"`
					} `json:"text"`
				} `json:"rich_text"`
			} `json:"paragraph"`
		} `json:"children"`
	}
	if err := json.Unmarshal(payload, &page); err != nil {
		t.Fatal(err)
	}
	if page.Parent["data_source_id"] != cfg.Notion.DataSourceID || len(page.Properties) != 3 || len(page.Children) != 3 {
		t.Fatalf("invalid page: %s", payload)
	}
	if page.Children[0].Type != "heading_2" {
		t.Fatal("missing heading")
	}
	var reconstructed string
	for _, block := range page.Children[1:] {
		content := block.Paragraph.RichText[0].Text.Content
		if len([]rune(content)) > 2000 {
			t.Fatal("oversized rich text")
		}
		reconstructed += content
	}
	if reconstructed != text {
		t.Fatal("content changed")
	}
	if _, err := pagePayload(cfg, strings.Repeat("a", 2001), "2026-10-11", entry); err == nil {
		t.Fatal("oversized title accepted")
	}
	entry.sections[0] = strings.Repeat("a", 200000)
	if _, err := pagePayload(cfg, "title", "2026-10-11", entry); err == nil {
		t.Fatal("oversized block list accepted")
	}
}

func TestCreatePageResultAndNoRetry(t *testing.T) {
	for _, response := range []string{`{"object":"page","id":"id","url":"https://www.notion.so/page"}`, `{}`, `not-json`, `{"object":"page","id":"id","url":"https://evil.test/page"}`} {
		calls := 0
		url, err := createPage([]byte(`{"body":"work"}`), func(in io.Reader) ([]byte, error) {
			calls++
			data, _ := io.ReadAll(in)
			if string(data) != `{"body":"work"}` {
				t.Fatal("stdin changed")
			}
			return []byte(response), nil
		})
		if calls != 1 || (err == nil) != (url != "") || (err == nil) != strings.Contains(response, "www.notion.so") {
			t.Fatalf("result %q %v calls %d", url, err, calls)
		}
	}
	calls := 0
	_, err := createPage(nil, func(io.Reader) ([]byte, error) { calls++; return nil, errors.New("secret-token") })
	if calls != 1 || err == nil || strings.Contains(err.Error(), "secret-token") || !strings.Contains(err.Error(), "before retrying") {
		t.Fatalf("unsafe failure %v", err)
	}
}

func TestAddCancellationAndFailureBoundary(t *testing.T) {
	cfg, _ := loadConfig(configFile(t, validConfig))
	for _, answer := range []string{"", "n", "yes"} {
		calls := 0
		var out bytes.Buffer
		code, err := runAddWithPoster(cfg, strings.NewReader(addTestInput("Project", "Work", make([]string, 7), answer)), &out, testAddDate(), func(configuration, string, string, workLog) (string, error) {
			calls++
			return "", errors.New("posting failed")
		})
		if answer == "yes" {
			if calls != 1 || code != 1 || err == nil || strings.Contains(out.String(), "Created:") {
				t.Fatal("failure reported as success")
			}
		} else if calls != 0 || code != 3 || err != nil {
			t.Fatal("cancel invoked posting")
		}
	}
}

func TestPostNTNProcessStdin(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\n[ \"$*\" = 'api v1/pages -X POST --data @-' ] || exit 1\nIFS= read -r line\n[ \"$line\" = '{\"body\":\"$(secret)\"}' ] || exit 1\nprintf '{\"object\":\"page\",\"id\":\"id\",\"url\":\"https://www.notion.so/page\"}'\n"
	if err := os.WriteFile(filepath.Join(dir, "ntn"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	data, err := executeNTNInput(time.Second, strings.NewReader("{\"body\":\"$(secret)\"}\n"), "api", "v1/pages", "-X", "POST", "--data", "@-")
	if err != nil || !strings.Contains(string(data), "https://www.notion.so/page") {
		t.Fatalf("stdin transport: %s %v", data, err)
	}
}
