package main

import (
	"fmt"
	"os"
)

var version = "dev"

func usage() {
	fmt.Print(`apex - custom Android SDK manager (apex-tools)

Usage:
  apex --list [--all|--json]          list available/installed packages
  apex install <pkg>... [--force]     install packages (e.g. "platforms;android-34")
  apex uninstall <pkg>...             remove installed packages
  apex update                         upgrade installed packages side-by-side
  apex licenses [--accept]            show/accept license, write licenses/
  apex --version                      print CLI version

Environment:
  ANDROID_HOME      SDK root (fallback: ANDROID_SDK_ROOT)
  APEX_CATALOG_URL  catalog.json URL override
`)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	cmd := args[0]
	rest := args[1:]

	var err error
	switch cmd {
	case "--version", "version":
		fmt.Println("apex", version)
	case "--list", "list":
		err = cmdList(rest)
	case "install":
		err = cmdInstall(rest)
	case "uninstall":
		err = cmdUninstall(rest)
	case "update":
		err = cmdUpdate(rest)
	case "licenses", "--licenses":
		err = cmdLicenses(rest)
	case "--help", "help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "apex: unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "apex:", err)
		os.Exit(1)
	}
}
