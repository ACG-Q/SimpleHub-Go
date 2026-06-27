package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const Version = "1.0.0"

func main() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to get executable path:", err)
		os.Exit(1)
	}
	exeDir := filepath.Dir(exe)

	args := os.Args[1:]
	cmd := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "":
		startServer(exeDir)
	case "help":
		printHelp()
	case "info":
		cmdInfo(exeDir)
	case "version":
		fmt.Println("SimpleHub version " + Version)
	case "reset":
		cmdReset(exeDir)
	case "re-encrypt":
		cmdReEncrypt(exeDir)
	case "db":
		runDBCommand(exeDir, args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		fmt.Fprintln(os.Stderr, "Run 'server.exe help' for usage")
		os.Exit(1)
	}
}
