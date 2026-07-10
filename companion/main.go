package main

import (
	"fmt"
	"os"

	"xeneoncc/internal/cmds"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: xeneon-bridge <serve|service|statusline|hook>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		if err := cmds.Serve(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "service":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: xeneon-bridge service <install|uninstall|start|stop|status|run> [--purge]")
			os.Exit(2)
		}
		var serr error
		switch os.Args[2] {
		case "install":
			serr = cmds.ServiceInstall()
		case "uninstall":
			serr = cmds.ServiceUninstall(hasFlag(os.Args[3:], "--purge"))
		case "start":
			serr = cmds.ServiceStart()
		case "stop":
			serr = cmds.ServiceStop()
		case "status":
			serr = cmds.ServiceStatus()
		case "run":
			serr = cmds.ServiceRun()
		default:
			fmt.Fprintln(os.Stderr, "unknown service command:", os.Args[2])
			os.Exit(2)
		}
		if serr != nil {
			fmt.Fprintln(os.Stderr, "error:", serr)
			os.Exit(1)
		}
	case "statusline":
		if err := cmds.Statusline(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "hook":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: xeneon-bridge hook <notify>")
			os.Exit(2)
		}
		switch os.Args[2] {
		case "notify":
			if err := cmds.Notify(os.Stdin); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintln(os.Stderr, "unknown hook:", os.Args[2])
			os.Exit(2)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		os.Exit(2)
	}
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
