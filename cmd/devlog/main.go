package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

const usage = `Usage: devlog [--config <path>] <command>

Commands:
  add      Capture a log interactively or with arguments and post to Notion
  doctor   Diagnose configuration and ntn (read-only)
  config   Show effective configuration

Options:
  -c, --config <path>  Configuration file (default: config/local.toml)
  -h, --help           Show help

Add options (after "add"):
` + addOptions + `
Examples:
  devlog add
  devlog add -p devlog-cli -t "Implemented a feature" -e "Tests passed"
  devlog add -p devlog-cli -t "Implemented a feature" --no-preview -y

Use devlog <command> --help for command help.
`

const addOptions = `  -p, --project <name>     Project name (default: configured project)
  -t, --today <text>       What you did today (required in argument mode)
  -d, --decision <text>    Decisions and changes
  -a, --ai <text>          Tasks delegated to AI
  -s, --success <text>     What went well
  -b, --blocker <text>     Blockers and failures
  -e, --evidence <text>    Evidence
  -l, --learning <text>    Learning
  -n, --next <text>        Next steps
  --no-preview            Skip body preview (confirmation remains unless -y)
  -y, --yes               Post without confirmation (validation remains)
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errOut io.Writer) int {
	return runWithInput(args, os.Stdin, out, errOut, time.Now())
}

func runWithInput(args []string, in io.Reader, out, errOut io.Writer, now time.Time) int {
	flags := flag.NewFlagSet("devlog", flag.ContinueOnError)
	flags.SetOutput(errOut)
	configPath := flags.String("config", "config/local.toml", "Configuration file")
	flags.StringVar(configPath, "c", *configPath, "Configuration file")
	flags.Usage = func() { fmt.Fprint(out, usage) }
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() == 0 {
		fmt.Fprint(out, usage)
		return 0
	}
	command := flags.Arg(0)
	switch command {
	case "add", "doctor", "config":
	default:
		fmt.Fprintf(errOut, "Unknown command: %s\n", command)
		return 2
	}

	sub := flag.NewFlagSet(command, flag.ContinueOnError)
	sub.SetOutput(errOut)
	sub.StringVar(configPath, "config", *configPath, "Configuration file")
	sub.StringVar(configPath, "c", *configPath, "Configuration file")

	var logArgs addArgs
	if command == "add" {
		for _, option := range []struct {
			long, short, help string
			value             **string
		}{
			{"project", "p", "Project name", &logArgs.project},
			{"today", "t", "What you did today", &logArgs.today},
			{"decision", "d", "Decisions and changes", &logArgs.decision},
			{"ai", "a", "Tasks delegated to AI", &logArgs.ai},
			{"success", "s", "What went well", &logArgs.success},
			{"blocker", "b", "Blockers and failures", &logArgs.blocker},
			{"evidence", "e", "Evidence", &logArgs.evidence},
			{"learning", "l", "Learning", &logArgs.learning},
			{"next", "n", "Next steps", &logArgs.next},
		} {
			set := func(value string) error {
				*option.value = &value
				return nil
			}
			sub.Func(option.long, option.help, set)
			sub.Func(option.short, option.help, set)
		}
		sub.BoolVar(&logArgs.noPreview, "no-preview", false, "Skip body preview")
		sub.BoolVar(&logArgs.yes, "yes", false, "Post without confirmation")
		sub.BoolVar(&logArgs.yes, "y", false, "Post without confirmation")
	}

	sub.Usage = func() {
		fmt.Fprintf(out, "Usage: devlog %s [options]\n", command)
		if command == "add" {
			fmt.Fprintln(out, "Collect a log interactively or with arguments. Confirm before posting unless -y is set.")
			fmt.Fprintln(out, "\nOptions:")
			fmt.Fprint(out, addOptions)
		}
		fmt.Fprintln(out, "  -c, --config <path>      Configuration file")
		fmt.Fprintln(out, "  -h, --help               Show help")
	}

	if err := sub.Parse(flags.Args()[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	if sub.NArg() != 0 || *configPath == "" {
		fmt.Fprintln(errOut, "Unexpected arguments or empty --config path")
		return 2
	}
	cfg, err := loadConfig(*configPath)
	if err == nil {
		switch command {
		case "add":
			var code int
			code, err = runAddWithArgs(cfg, in, out, now, logArgs)
			if err == nil {
				return code
			}
		case "config":
			var path string
			path, err = filepath.Abs(*configPath)
			if err == nil {
				fmt.Fprintf(out, "# Configuration: %q\n", path)
				err = toml.NewEncoder(out).Encode(cfg)
			}
		case "doctor":
			fmt.Fprintln(out, "OK configuration")
			err = diagnose(cfg, out, executeNTN)
		}
	}
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}
