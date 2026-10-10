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
  add      Capture and preview a development log (posting not implemented)
  doctor   Diagnose configuration and ntn (read-only)
  config   Show effective configuration

Options:
  --config <path>  Configuration file (default: config/local.toml)
  -h, --help       Show help

Use devlog <command> --help for command help.
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
	sub.Usage = func() {
		fmt.Fprintf(out, "Usage: devlog %s [--config <path>]\n", command)
		if command == "add" {
			fmt.Fprintln(out, "Collect a log, show its preview, and confirm. Posting is not implemented.")
		}
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
			code, err = runAdd(cfg, in, out, now)
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
