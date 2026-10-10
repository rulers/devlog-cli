package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestValidateRequiredInput(t *testing.T) {
	tests := []struct {
		name    string
		project string
		today   string
		wantErr string
	}{
		{"valid", "Project", "Work", ""},
		{"empty project", "", "Work", "Project is required"},
		{"whitespace project", "  ", "Work", "Project is required"},
		{"empty today", "Project", "", "今日やったこと is required"},
		{"whitespace today", "Project", " \t", "今日やったこと is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequiredInput(tt.project, tt.today)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestCollectWorkLogUsesConfiguredProjectDefault(t *testing.T) {
	input := "\nToday\nDecision\nAI task\nSuccess\nBlocker\nhttps://example.test\nLearning\nNext\n"
	var prompts bytes.Buffer
	entry, canceled, err := collectWorkLog(bufio.NewReader(strings.NewReader(input)), &prompts, "  Configured project ")
	if err != nil || canceled {
		t.Fatalf("collectWorkLog = canceled %v, error %v", canceled, err)
	}
	if entry.project != "Configured project" {
		t.Fatalf("project = %q", entry.project)
	}
	want := []string{"Today", "Decision", "AI task", "Success", "Blocker", "https://example.test", "Learning", "Next"}
	if len(entry.sections) != len(want) {
		t.Fatalf("got %d sections, want %d", len(entry.sections), len(want))
	}
	for i := range want {
		if entry.sections[i] != want[i] {
			t.Errorf("section %d = %q, want %q", i, entry.sections[i], want[i])
		}
	}
	promptText := prompts.String()
	last := -1
	for _, section := range addSections {
		index := strings.Index(promptText, section.title+" ")
		if index <= last {
			t.Fatalf("prompt for %q was missing or out of order: %s", section.title, promptText)
		}
		last = index
	}
}

func TestRenderMarkdownPreservesOrderAndOmitsEmptyOptionalSections(t *testing.T) {
	entry := workLog{project: "Project", sections: []string{
		"Work done", "", "AI delegation", "", "", "https://example.test", "", "",
	}}
	got := renderMarkdown(entry)
	want := "## 今日やったこと\n\nWork done\n\n## AIに任せたこと\n\nAI delegation\n\n## Evidence\n\nhttps://example.test\n"
	if got != want {
		t.Fatalf("Markdown:\n%s\nwant:\n%s", got, want)
	}
}

func TestRunAddPreviewAndCancel(t *testing.T) {
	data := strings.Replace(validConfig, `project = ""`, `project = "Preview project"`, 1)
	cfg, err := loadConfig(configFile(t, data))
	if err != nil {
		t.Fatal(err)
	}
	input := addTestInput("", "Finished feature", []string{"", "", "", "", "", "", ""}, "n")
	var out bytes.Buffer
	code, err := runAdd(cfg, strings.NewReader(input), &out, testAddDate())
	if err != nil {
		t.Fatal(err)
	}
	if code != addCancelledExitCode {
		t.Fatalf("exit = %d, want %d", code, addCancelledExitCode)
	}
	for _, text := range []string{
		"Destination: Notion data source 22852bf0-d4c9-4740-9e13-6d5af38794fc",
		"Title: 2026-10-11 / Preview project",
		"Date: 2026-10-11",
		"Tag: AI開発",
		"## 今日やったこと\n\nFinished feature",
		"Confirm preview (no posting will occur)? [y/N]:",
		"Cancelled; nothing was sent.",
	} {
		if !strings.Contains(out.String(), text) {
			t.Errorf("preview missing %q:\n%s", text, &out)
		}
	}
	if strings.Contains(out.String(), "Posting is not implemented") {
		t.Fatal("cancelled preview reported confirmation")
	}
}

func TestConfirmationDefaultsToCancel(t *testing.T) {
	var out bytes.Buffer
	confirmed, canceled, err := confirmAdd(bufio.NewReader(strings.NewReader("\n")), &out)
	if err != nil || canceled || confirmed {
		t.Fatalf("confirmAdd = confirmed %v, canceled %v, error %v", confirmed, canceled, err)
	}
}

func TestRunAddEOFAlwaysCancels(t *testing.T) {
	cfg, err := loadConfig(configFile(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		input string
	}{
		{"partial project line", "unfinished project"},
		{"EOF at confirmation", strings.TrimSuffix(addTestInput("Project", "Work", []string{"", "", "", "", "", "", ""}, ""), "\n")},
		{"partial confirmation line", strings.TrimSuffix(addTestInput("Project", "Work", []string{"", "", "", "", "", "", ""}, "y"), "\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			code, err := runAdd(cfg, strings.NewReader(tt.input), &out, testAddDate())
			if err != nil {
				t.Fatal(err)
			}
			if code != addCancelledExitCode {
				t.Fatalf("exit = %d, want %d", code, addCancelledExitCode)
			}
			if !strings.Contains(out.String(), "Cancelled; nothing was sent.") {
				t.Fatalf("EOF was not reported as cancellation: %s", &out)
			}
			if strings.Contains(tt.name, "partial project") && strings.Contains(out.String(), "Destination:") {
				t.Fatal("partial line was accepted despite EOF")
			}
		})
	}
}

func TestConfirmedAddDoesNotPost(t *testing.T) {
	path := configFile(t, strings.Replace(validConfig, `project = ""`, `project = "Configured project"`, 1))
	input := addTestInput("", "Work completed", []string{"", "", "", "", "", "", ""}, "y")
	var out, errOut bytes.Buffer
	code := runWithInput([]string{"add", "--config", path}, strings.NewReader(input), &out, &errOut, testAddDate())
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr: %s", code, &errOut)
	}
	for _, text := range []string{
		"Title: 2026-10-11 / Configured project",
		"Date: 2026-10-11",
		"Tag: AI開発",
		"Posting is not implemented; nothing was sent.",
	} {
		if !strings.Contains(out.String(), text) {
			t.Errorf("confirmed preview missing %q:\n%s", text, &out)
		}
	}
	if strings.Contains(errOut.String(), "error") {
		t.Fatalf("unexpected error output: %s", &errOut)
	}
}

func TestRunAddRetriesInvalidRequiredInput(t *testing.T) {
	cfg, err := loadConfig(configFile(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}
	input := "\nProject\n\nWork\n" + strings.Repeat("\n", 7) + "y\n"
	var out bytes.Buffer
	code, err := runAdd(cfg, strings.NewReader(input), &out, testAddDate())
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "今日やったこと is required") {
		t.Fatalf("empty required work was not rejected: %s", &out)
	}
	if !strings.Contains(out.String(), "Title: 2026-10-11 / Project") {
		t.Fatalf("blank project was not re-prompted: %s", &out)
	}
}

func addTestInput(project, today string, optional []string, answer string) string {
	lines := []string{project, today}
	lines = append(lines, optional...)
	lines = append(lines, answer)
	return strings.Join(lines, "\n") + "\n"
}

func testAddDate() time.Time {
	return time.Date(2026, time.October, 11, 0, 0, 0, 0, time.Local)
}
