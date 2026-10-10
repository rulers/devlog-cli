package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestYesPostsWithoutConfirmation(t *testing.T) {
	cfg, err := loadConfig(configFile(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}
	for _, noPreview := range []bool{false, true} {
		for _, argMode := range []bool{false, true} {
			args := addArgs{yes: true, noPreview: noPreview}
			input := ""
			if argMode {
				args.project = ptrString("Project")
				args.today = ptrString("Work")
			} else {
				input = "Project\nWork\n" + strings.Repeat("\n", 7)
			}
			calls := 0
			var out bytes.Buffer
			code, err := runAddWithPoster(cfg, strings.NewReader(input), &out, testAddDate(), func(configuration, string, string, workLog) (string, error) {
				calls++
				return "https://www.notion.so/page", nil
			}, args)
			if err != nil || code != 0 || calls != 1 || strings.Contains(out.String(), "Post this log to Notion?") || !strings.Contains(out.String(), "Created:") {
				t.Fatalf("exit %d calls %d error %v: %s", code, calls, err, &out)
			}
			if strings.Contains(out.String(), "Body:") == noPreview {
				t.Fatal("preview option ignored")
			}
		}
	}
}

func TestYesKeepsValidationEOFAndFailure(t *testing.T) {
	cfg, _ := loadConfig(configFile(t, validConfig))
	for _, args := range []addArgs{{yes: true}, {yes: true, project: ptrString("Project"), today: ptrString(" ")}} {
		var out bytes.Buffer
		code, err := runAddWithPoster(cfg, strings.NewReader(""), &out, testAddDate(), func(configuration, string, string, workLog) (string, error) {
			t.Fatal("invalid input posted")
			return "", nil
		}, args)
		if isArgMode(args) {
			if code != 1 || err == nil {
				t.Fatal("validation bypassed")
			}
		} else if code != 3 || err != nil {
			t.Fatal("EOF did not cancel")
		}
	}
	calls := 0
	var out bytes.Buffer
	code, err := runAddWithPoster(cfg, strings.NewReader(""), &out, testAddDate(), func(configuration, string, string, workLog) (string, error) {
		calls++
		return "", errors.New("posting failed")
	}, addArgs{yes: true, project: ptrString("Project"), today: ptrString("Work")})
	if code != 1 || err == nil || calls != 1 || strings.Contains(out.String(), "Created:") {
		t.Fatal("failure retried or reported success")
	}
}

func TestYesAliasesParseAndFalseRestoresConfirmation(t *testing.T) {
	path := configFile(t, validConfig)
	for _, flag := range []string{"-y", "--yes"} {
		var out, errOut bytes.Buffer
		code := runWithInput([]string{"add", "-c", path, "-p", "Project", "-t", " ", flag}, strings.NewReader(""), &out, &errOut, testAddDate())
		if code != 1 || !strings.Contains(errOut.String(), "is required") {
			t.Fatalf("flag %s exit %d: %s", flag, code, &errOut)
		}
	}
	var out, errOut bytes.Buffer
	code := runWithInput([]string{"add", "-c", path, "-p", "Project", "-t", "Work", "--yes", "-y=false"}, strings.NewReader(""), &out, &errOut, testAddDate())
	if code != 3 || !strings.Contains(out.String(), "Post this log to Notion?") {
		t.Fatalf("exit %d: %s %s", code, &out, &errOut)
	}
}
