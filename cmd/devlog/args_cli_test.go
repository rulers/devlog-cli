package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIArgumentAliasesAndLastValue(t *testing.T) {
	path := configFile(t, validConfig)
	for _, names := range [][2]string{{"project", "p"}, {"today", "t"}, {"decision", "d"}, {"ai", "a"}, {"success", "s"}, {"blocker", "b"}, {"evidence", "e"}, {"learning", "l"}, {"next", "n"}} {
		for _, order := range [][2]string{{"--" + names[0], "-" + names[1]}, {"-" + names[1], "--" + names[0]}} {
			t.Run(strings.Join(order[:], "/"), func(t *testing.T) {
				args := []string{"add", "-c", path, "-p", "Project", "-t", "Work", order[0], "discarded-value", order[1], "selected-value"}
				var out, errOut bytes.Buffer
				code := runWithInput(args, strings.NewReader("n\n"), &out, &errOut, testAddDate())
				if code != 3 || !strings.Contains(out.String(), "selected-value") || strings.Contains(out.String(), "discarded-value") || strings.Contains(out.String(), "(required)") {
					t.Fatalf("exit %d: %s %s", code, &out, &errOut)
				}
			})
		}
	}
}

func TestCLIExplicitEmptyAndRequiredArguments(t *testing.T) {
	path := configFile(t, strings.Replace(validConfig, `project = ""`, `project = "Default"`, 1))
	for _, fields := range [][]string{
		{"--today", ""}, {"-t", "  "}, {"--project", "", "-t", "Work"}, {"-p", "  ", "-t", "Work"}, {"--evidence", ""}, {"-e", ""}, {"--project", "Project"},
	} {
		var out, errOut bytes.Buffer
		code := runWithInput(append([]string{"add", "-c", path}, fields...), strings.NewReader(""), &out, &errOut, testAddDate())
		if code != 1 || !strings.Contains(errOut.String(), "is required") || out.Len() != 0 {
			t.Fatalf("fields %v, exit %d: %s %s", fields, code, &out, &errOut)
		}
	}
	for _, fields := range [][]string{{"-t", "Work"}, {"-p", "Project", "-t", "Work", "--evidence", "  "}} {
		var out, errOut bytes.Buffer
		code := runWithInput(append([]string{"add", "-c", path}, fields...), strings.NewReader("n\n"), &out, &errOut, testAddDate())
		if code != 3 || strings.Contains(out.String(), "## Evidence") || strings.Contains(out.String(), "(required)") {
			t.Fatalf("exit %d: %s %s", code, &out, &errOut)
		}
		if len(fields) == 2 && !strings.Contains(out.String(), " / Default") {
			t.Fatal("default project missing")
		}
	}
}

func TestCLINoPreviewInBothInputModes(t *testing.T) {
	path := configFile(t, validConfig)
	for _, argMode := range []bool{false, true} {
		args := []string{"add", "-c", path, "--no-preview"}
		input := addTestInput("Project", "private-body", make([]string, 7), "n")
		if argMode {
			args = append(args, "-p", "Project", "-t", "private-body")
			input = "n\n"
		}
		var out, errOut bytes.Buffer
		code := runWithInput(args, strings.NewReader(input), &out, &errOut, testAddDate())
		if code != 3 || strings.Contains(out.String(), "Body:") || strings.Contains(out.String(), "private-body") {
			t.Fatalf("exit %d: %s %s", code, &out, &errOut)
		}
		for _, label := range []string{"Destination:", "Title:", "Date:", "Tag:", "Post this log to Notion?", "Cancelled; nothing was sent."} {
			if !strings.Contains(out.String(), label) {
				t.Fatalf("missing %s", label)
			}
		}
		if strings.Contains(out.String(), "(required)") == argMode {
			t.Fatal("wrong input mode")
		}
	}
}

func TestCLIConfigAliasOrderAndEmpty(t *testing.T) {
	path := configFile(t, validConfig)
	for _, args := range [][]string{
		{"-c", path, "config"}, {"config", "-c", path},
		{"--config", "missing", "-c", path, "config"}, {"-c", "missing", "--config", path, "config"},
		{"config", "--config", "missing", "-c", path}, {"config", "-c", "missing", "--config", path},
		{"-c", "missing", "config", "--config", path},
	} {
		var out, errOut bytes.Buffer
		if code := runWithInput(args, strings.NewReader(""), &out, &errOut, testAddDate()); code != 0 || !strings.Contains(out.String(), path) {
			t.Fatalf("%v: %d %s", args, code, &errOut)
		}
	}
	for _, args := range [][]string{{"-c", "", "config"}, {"config", "-c", ""}, {"config", "--config", path, "-c", ""}} {
		var out, errOut bytes.Buffer
		if code := runWithInput(args, strings.NewReader(""), &out, &errOut, testAddDate()); code != 2 {
			t.Fatalf("%v: %d", args, code)
		}
	}
}
