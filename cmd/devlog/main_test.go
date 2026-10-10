package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		args []string
		code int
		text string
	}{
		{"no args", nil, 0, "Usage: devlog"},
		{"help", []string{"--help"}, 0, "Commands:"},
		{"unknown", []string{"unknown"}, 2, "Unknown command"},
		{"unknown flag", []string{"--unknown"}, 2, "flag provided but not defined"},
		{"empty config", []string{"config", "--config", ""}, 2, "empty --config"},
		{"extra argument", []string{"add", "extra"}, 2, "Unexpected arguments"},
	}
	for _, command := range []string{"add", "doctor", "config"} {
		tests = append(tests,
			struct {
				name string
				args []string
				code int
				text string
			}{command, []string{command}, 1, "not implemented yet"},
			struct {
				name string
				args []string
				code int
				text string
			}{command + " help", []string{command, "--help"}, 0, "Usage: devlog " + command},
			struct {
				name string
				args []string
				code int
				text string
			}{command + " global config", []string{"--config", "custom.toml", command}, 1, "not implemented yet"},
			struct {
				name string
				args []string
				code int
				text string
			}{command + " config", []string{command, "--config", "custom.toml"}, 1, "not implemented yet"},
		)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run(tt.args, &out, &errOut); code != tt.code {
				t.Fatalf("exit = %d, want %d", code, tt.code)
			}
			if !strings.Contains(out.String()+errOut.String(), tt.text) {
				t.Fatalf("missing %q in output: %s%s", tt.text, &out, &errOut)
			}
		})
	}
}
