package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

func postWorkLog(cfg configuration, title, date string, entry workLog) (string, error) {
	payload, err := pagePayload(cfg, title, date, entry)
	if err != nil {
		return "", err
	}
	// Check existing tag and schema before creating; never create tag options implicitly.
	if err := diagnose(cfg, io.Discard, executeNTN); err != nil {
		return "", err
	}
	return createPage(payload, func(input io.Reader) ([]byte, error) {
		return executeNTNInput(30*time.Second, input, "api", "v1/pages", "-X", "POST", "--data", "@-")
	})
}

func pagePayload(cfg configuration, title, date string, entry workLog) ([]byte, error) {
	richText := func(text string) []any {
		return []any{map[string]any{"type": "text", "text": map[string]string{"content": text}}}
	}
	if len([]rune(title)) > 2000 {
		return nil, fmt.Errorf("title is too long (maximum 2000 characters)")
	}
	children := []any{}
	block := func(kind, text string) {
		children = append(children, map[string]any{"object": "block", "type": kind, kind: map[string]any{"rich_text": richText(text)}})
	}
	for i, section := range addSections {
		text := strings.TrimSpace(entry.sections[i])
		if text == "" {
			continue
		}
		block("heading_2", section.title)
		runes := []rune(text)
		for len(runes) > 0 {
			n := len(runes)
			if n > 2000 {
				n = 2000
			}
			block("paragraph", string(runes[:n]))
			runes = runes[n:]
		}
	}
	if len(children) > 100 {
		return nil, fmt.Errorf("log is too long (maximum 100 blocks)")
	}
	payload, err := json.Marshal(map[string]any{
		"parent": map[string]string{"type": "data_source_id", "data_source_id": cfg.Notion.DataSourceID},
		"properties": map[string]any{
			cfg.Notion.TitleProperty: map[string]any{"title": richText(title)},
			cfg.Notion.DateProperty:  map[string]any{"date": map[string]string{"start": date}},
			cfg.Notion.TagsProperty:  map[string]any{"multi_select": []any{map[string]string{"name": cfg.Notion.Tag}}},
		},
		"children": children,
	})
	if err == nil && len(payload) > 500000 {
		return nil, fmt.Errorf("log payload is too large")
	}
	return payload, err
}

func createPage(payload []byte, execute func(io.Reader) ([]byte, error)) (string, error) {
	data, err := execute(bytes.NewReader(payload))
	unknown := fmt.Errorf("posting result is unknown; check the destination in Notion before retrying manually (no automatic retry)")
	if err != nil {
		return "", unknown
	}
	var page struct {
		Object string `json:"object"`
		ID     string `json:"id"`
		URL    string `json:"url"`
	}
	if json.Unmarshal(data, &page) != nil || page.Object != "page" || page.ID == "" {
		return "", unknown
	}
	u, err := url.Parse(page.URL)
	if err != nil || u.Scheme != "https" || u.User != nil || (u.Host != "www.notion.so" && u.Host != "notion.so" && u.Host != "app.notion.com") || u.Path == "" {
		return "", unknown
	}
	return page.URL, nil
}
