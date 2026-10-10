package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type configuration struct {
	Notion struct {
		DataSourceID  string `toml:"data_source_id"`
		TitleProperty string `toml:"title_property"`
		DateProperty  string `toml:"date_property"`
		TagsProperty  string `toml:"tags_property"`
		Tag           string `toml:"tag"`
	} `toml:"notion"`
	Defaults struct {
		Project string `toml:"project"`
	} `toml:"defaults"`
}

var sourceID = regexp.MustCompile(`^(?:[0-9a-fA-F]{32}|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$`)

func loadConfig(path string) (configuration, error) {
	var cfg configuration
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("cannot read configuration file; check --config and file permissions")
	}
	meta, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return cfg, fmt.Errorf("invalid TOML configuration; check syntax and value types")
	}
	if len(meta.Undecoded()) != 0 {
		return cfg, fmt.Errorf("unsupported configuration keys; use config/example.toml (credentials belong in ntn or the environment)")
	}
	if !sourceID.MatchString(cfg.Notion.DataSourceID) {
		return cfg, fmt.Errorf("notion.data_source_id must be a data source UUID; replace the example placeholder")
	}
	for _, item := range []struct{ name, value string }{
		{"title_property", cfg.Notion.TitleProperty}, {"date_property", cfg.Notion.DateProperty},
		{"tags_property", cfg.Notion.TagsProperty}, {"tag", cfg.Notion.Tag},
	} {
		if strings.TrimSpace(item.value) == "" {
			return cfg, fmt.Errorf("notion.%s is required", item.name)
		}
	}
	if cfg.Notion.TitleProperty == cfg.Notion.DateProperty || cfg.Notion.TitleProperty == cfg.Notion.TagsProperty || cfg.Notion.DateProperty == cfg.Notion.TagsProperty {
		return cfg, fmt.Errorf("title, date and tags property names must be distinct")
	}
	return cfg, nil
}
