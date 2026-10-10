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
		{"config shorthand -c", []string{"-c", "missing.toml", "config"}, 1, "cannot read configuration"},
		{"add shorthand -c", []string{"add", "-c", "missing.toml"}, 1, "cannot read configuration"},
	}
	for _, command := range []string{"add", "doctor", "config"} {
		tests = append(tests,
			struct {
				name string
				args []string
				code int
				text string
			}{command, []string{command, "--config", "missing.toml"}, 1, failureText(command)},
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
			}{command + " global config", []string{"--config", "custom.toml", command}, 1, failureText(command)},
			struct {
				name string
				args []string
				code int
				text string
			}{command + " config", []string{command, "--config", "custom.toml"}, 1, failureText(command)},
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

func TestAddCommandWithArguments(t *testing.T) {
	cfg := configFile(t, validConfig)
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"project short", []string{"add", "-c", cfg, "-p", "MyProject", "-t", "Work"}, "MyProject"},
		{"project long", []string{"add", "-c", cfg, "--project", "LongProject", "-t", "Work"}, "LongProject"},
		{"today short", []string{"add", "-c", cfg, "-p", "P", "-t", "ShortToday"}, "ShortToday"},
		{"today long", []string{"add", "-c", cfg, "-p", "P", "--today", "LongToday"}, "LongToday"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := runWithInput(tt.args, strings.NewReader("n\n"), &out, &errOut, testAddDate())
			if code != addCancelledExitCode {
				t.Fatalf("exit = %d, want %d; error: %s", code, addCancelledExitCode, errOut.String())
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("output missing %q: %s", tt.want, &out)
			}
		})
	}
}

func TestAddCommandUnknownFlag(t *testing.T) {
	cfg := configFile(t, validConfig)
	var out, errOut bytes.Buffer
	code := runWithInput([]string{"add", "-c", cfg, "--unknown", "value"}, strings.NewReader(""), &out, &errOut, testAddDate())
	if code != 2 {
		t.Fatalf("exit = %d, want 2 for unknown flag", code)
	}
}

func failureText(_ string) string {
	return "cannot read configuration"
}
