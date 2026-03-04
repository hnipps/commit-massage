package main

import (
	"fmt"
	"os"

	"github.com/nicholls-inc/commit-massage/internal/generate"
	"github.com/nicholls-inc/commit-massage/internal/hook"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: commit-massage <install|uninstall|generate> [args]")
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	case "install":
		force := len(os.Args) > 2 && os.Args[2] == "--force"
		err = hook.Install(force)
	case "uninstall":
		err = hook.Uninstall()
	case "generate":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: commit-massage generate <msg-file> [source]")
			os.Exit(1)
		}
		msgFile := os.Args[2]
		var source string
		if len(os.Args) > 3 {
			source = os.Args[3]
		}
		err = generate.Run(msgFile, source)
		if err != nil {
			fmt.Fprintf(os.Stderr, "commit-massage: %s\n", err)
			os.Exit(0)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "usage: commit-massage <install|uninstall|generate> [args]")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
