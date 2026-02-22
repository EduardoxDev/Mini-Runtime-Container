//go:build linux

// GoContainer — A minimal container runtime using Linux namespaces and cgroups.
//
// This is the main entry point for the gocontainer binary. It handles both
// the CLI interface (parent process) and the container init (child process)
// using the re-exec pattern.
//
// Re-exec pattern:
//  1. User runs: gocontainer run /bin/sh
//  2. Parent creates child via: /proc/self/exe __init__ <rootfs> <hostname> /bin/sh
//  3. Child detects __init__ arg and calls InitContainer()
//  4. InitContainer sets up mounts, hostname, and execs the user command
package main

import (
	"fmt"
	"os"

	"github.com/user/gocontainer/internal/cli"
	"github.com/user/gocontainer/internal/container"
)

func main() {
	// Check if we're being re-invoked as the container init process.
	// The parent process calls /proc/self/exe with "__init__" as the first arg.
	if len(os.Args) > 1 && os.Args[1] == "__init__" {
		if len(os.Args) < 5 {
			fmt.Fprintf(os.Stderr, "usage: %s __init__ <rootfs> <hostname> <command> [args...]\n", os.Args[0])
			os.Exit(1)
		}

		rootfs := os.Args[2]
		hostname := os.Args[3]
		command := os.Args[4]
		args := os.Args[5:]

		if err := container.InitContainer(rootfs, hostname, command, args); err != nil {
			fmt.Fprintf(os.Stderr, "container init error: %v\n", err)
			os.Exit(1)
		}

		// InitContainer calls syscall.Exec, so we never reach here
		os.Exit(0)
	}

	// Normal CLI execution
	cli.Execute()
}
