package main

import (
	"fmt"
	"os"

	"github.com/nicholls-inc/commit-massage/internal/hook"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: commit-massage <install|uninstall> [--force]")
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	case "install":
		force := len(os.Args) > 2 && os.Args[2] == "--force"
		err = hook.Install(force)
	case "uninstall":
		err = hook.Uninstall()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "usage: commit-massage <install|uninstall> [--force]")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
