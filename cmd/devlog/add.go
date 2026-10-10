package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const addCancelledExitCode = 3

type addSection struct {
	title    string
	required bool
}

var addSections = []addSection{
	{title: "今日やったこと", required: true},
	{title: "判断・変更"},
	{title: "AIに任せたこと"},
	{title: "うまくいったこと"},
	{title: "詰まったこと・失敗"},
	{title: "Evidence"},
	{title: "学び"},
	{title: "次にやること"},
}

type workLog struct {
	project  string
	sections []string
}

func runAdd(cfg configuration, in io.Reader, out io.Writer, now time.Time) (int, error) {
	reader := bufio.NewReader(in)
	entry, canceled, err := collectWorkLog(reader, out, cfg.Defaults.Project)
	if err != nil {
		return 1, err
	}
	if canceled {
		fmt.Fprintln(out, "Cancelled; nothing was sent.")
		return addCancelledExitCode, nil
	}

	date := now.Format("2006-01-02")
	title := date + " / " + entry.project
	body := renderMarkdown(entry)
	printAddPreview(out, cfg, title, date, body)

	confirmed, canceled, err := confirmAdd(reader, out)
	if err != nil {
		return 1, err
	}
	if canceled || !confirmed {
		fmt.Fprintln(out, "Cancelled; nothing was sent.")
		return addCancelledExitCode, nil
	}

	fmt.Fprintln(out, "Posting is not implemented; nothing was sent.")
	return 0, nil
}

func collectWorkLog(reader *bufio.Reader, out io.Writer, defaultProject string) (workLog, bool, error) {
	entry := workLog{sections: make([]string, len(addSections))}
	defaultProject = strings.TrimSpace(defaultProject)
	for entry.project == "" {
		label := "Project"
		if defaultProject != "" {
			label += " [" + defaultProject + "]"
		}
		value, canceled, err := readPrompt(reader, out, label+": ")
		if err != nil || canceled {
			return entry, canceled, err
		}
		entry.project = strings.TrimSpace(value)
		if entry.project == "" {
			entry.project = defaultProject
		}
	}

	for i, section := range addSections {
		for {
			label := section.title
			if section.required {
				label += " (required)"
			} else {
				label += " (optional)"
			}
			value, canceled, err := readPrompt(reader, out, label+": ")
			if err != nil || canceled {
				return entry, canceled, err
			}
			value = strings.TrimSpace(value)
			if section.required {
				if err := validateRequiredInput(entry.project, value); err != nil {
					fmt.Fprintf(out, "%s\n", err)
					continue
				}
			}
			entry.sections[i] = value
			break
		}
	}
	return entry, false, nil
}

func validateRequiredInput(project, today string) error {
	if strings.TrimSpace(project) == "" {
		return fmt.Errorf("Project is required")
	}
	if strings.TrimSpace(today) == "" {
		return fmt.Errorf("今日やったこと is required")
	}
	return nil
}

func readPrompt(reader *bufio.Reader, out io.Writer, prompt string) (string, bool, error) {
	fmt.Fprint(out, prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "", true, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(line), false, nil
}

func renderMarkdown(entry workLog) string {
	sections := make([]string, 0, len(addSections))
	for i, section := range addSections {
		if value := strings.TrimSpace(entry.sections[i]); value != "" {
			sections = append(sections, "## "+section.title+"\n\n"+value)
		}
	}
	return strings.Join(sections, "\n\n") + "\n"
}

func printAddPreview(out io.Writer, cfg configuration, title, date, body string) {
	fmt.Fprintf(out, "Destination: Notion data source %s\n", cfg.Notion.DataSourceID)
	fmt.Fprintf(out, "Title: %s\n", title)
	fmt.Fprintf(out, "Date: %s\n", date)
	fmt.Fprintf(out, "Tag: %s\n\n", cfg.Notion.Tag)
	fmt.Fprintf(out, "Body:\n%s", body)
}

func confirmAdd(reader *bufio.Reader, out io.Writer) (bool, bool, error) {
	answer, canceled, err := readPrompt(reader, out, "Confirm preview (no posting will occur)? [y/N]: ")
	if err != nil || canceled {
		return false, canceled, err
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true, false, nil
	default:
		return false, false, nil
	}
}
