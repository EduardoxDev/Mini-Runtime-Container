//go:build linux

// Package container provides the core container lifecycle management.
// It orchestrates namespace isolation, cgroup resource limits, and
// filesystem setup to create and run isolated processes.
package container

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/user/gocontainer/internal/cgroup"
	"github.com/user/gocontainer/internal/namespace"
)

// Status represents the current state of a container.
type Status int

const (
	StatusCreated Status = iota
	StatusRunning
	StatusStopped
	StatusFailed
)

func (s Status) String() string {
	switch s {
	case StatusCreated:
		return "created"
	case StatusRunning:
		return "running"
	case StatusStopped:
		return "stopped"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Container represents an isolated process environment with its own
// namespaces, cgroup limits, and filesystem.
type Container struct {
	// ID is a unique identifier for this container.
	ID string
	// Config holds the container configuration.
	Config Config
	// Status is the current lifecycle state.
	Status Status
	// PID is the process ID of the container's init process.
	PID int

	cmd     *exec.Cmd
	cgroups *cgroup.Manager
}

// New creates a new container with the given configuration.
// It generates a unique ID and initializes the cgroup manager.
func New(config Config) (*Container, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	id := generateID()
	return &Container{
		ID:      id,
		Config:  config,
		Status:  StatusCreated,
		cgroups: cgroup.NewManager(id),
	}, nil
}

// Run starts the container by forking the current process with new namespaces,
// applying cgroup limits, and executing the specified command.
//
// It uses the re-exec pattern: the parent process creates a child with
// new namespaces via /proc/self/exe, and the child then sets up mounts,
// hostname, and executes the user command.
func (c *Container) Run() error {
	if c.Status != StatusCreated {
		return fmt.Errorf("container %s is in state %s, expected created", c.ID, c.Status)
	}

	// Build the child command using the re-exec pattern.
	// The child is invoked as: /proc/self/exe __init__ <rootfs> <hostname> <command> [args...]
	initArgs := []string{"__init__", c.Config.Rootfs, c.Config.Hostname, c.Config.Command}
	initArgs = append(initArgs, c.Config.Args...)

	cmd := exec.Command("/proc/self/exe", initArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Configure namespace isolation
	nsConfig := namespace.DefaultConfig(c.Config.Hostname)
	cmd.SysProcAttr = namespace.SetupSysProcAttr(nsConfig)

	// Start the child process
	if err := cmd.Start(); err != nil {
		c.Status = StatusFailed
		return fmt.Errorf("failed to start container process: %w", err)
	}

	c.cmd = cmd
	c.PID = cmd.Process.Pid
	c.Status = StatusRunning

	// Apply cgroup limits to the child process
	if err := c.cgroups.Set(cgroup.Resources{
		MemoryMax: c.Config.MemoryLimit,
		CPUQuota:  c.Config.CPUQuota,
		CPUPeriod: 100000,
		PidsMax:   c.Config.PidsLimit,
	}); err != nil {
		// Non-fatal: log warning but continue
		fmt.Fprintf(os.Stderr, "warning: failed to set cgroup limits: %v\n", err)
	}

	if err := c.cgroups.Apply(c.PID); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to apply cgroup to pid %d: %v\n", c.PID, err)
	}

	return nil
}

// Wait blocks until the container's process exits and returns the result.
func (c *Container) Wait() error {
	if c.cmd == nil {
		return fmt.Errorf("container %s has not been started", c.ID)
	}

	err := c.cmd.Wait()
	c.Status = StatusStopped

	// Cleanup cgroups
	if cleanErr := c.cgroups.Cleanup(); cleanErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to cleanup cgroups: %v\n", cleanErr)
	}

	if err != nil {
		c.Status = StatusFailed
		return fmt.Errorf("container process exited with error: %w", err)
	}

	return nil
}

// Stop sends SIGTERM to the container process, then SIGKILL if it doesn't exit.
func (c *Container) Stop() error {
	if c.cmd == nil || c.cmd.Process == nil {
		return nil
	}

	// Try graceful termination first
	if err := c.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited
		return nil
	}

	// Force kill (in production, you'd wait with a timeout first)
	if err := c.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill container process: %w", err)
	}

	c.Status = StatusStopped
	return c.cgroups.Cleanup()
}

// InitContainer is called inside the child process (the container's init).
// It sets up the mount namespace, hostname, and executes the user command.
// This function is invoked via the re-exec pattern when os.Args[1] == "__init__".
func InitContainer(rootfs, hostname, command string, args []string) error {
	// Set hostname inside UTS namespace
	if err := namespace.SetHostname(hostname); err != nil {
		return fmt.Errorf("failed to set hostname: %w", err)
	}

	// Setup mount namespace (proc, dev, sys, pivot_root)
	if err := namespace.SetupMount(rootfs); err != nil {
		return fmt.Errorf("failed to setup mounts: %w", err)
	}

	// Execute the user command
	// Use syscall.Exec to replace the current process (no fork)
	binary, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("command not found: %s: %w", command, err)
	}

	argv := append([]string{command}, args...)
	if err := syscall.Exec(binary, argv, os.Environ()); err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}

	// This line is never reached
	return nil
}

// generateID creates a random 12-character hex container ID.
func generateID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// FormatSize converts bytes to a human-readable string.
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Info returns a formatted string with container information.
func (c *Container) Info() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Container ID: %s\n", c.ID))
	sb.WriteString(fmt.Sprintf("Status:       %s\n", c.Status))
	if c.PID > 0 {
		sb.WriteString(fmt.Sprintf("PID:          %d\n", c.PID))
	}
	sb.WriteString(fmt.Sprintf("Command:      %s %s\n", c.Config.Command, strings.Join(c.Config.Args, " ")))
	sb.WriteString(fmt.Sprintf("Hostname:     %s\n", c.Config.Hostname))
	if c.Config.MemoryLimit > 0 {
		sb.WriteString(fmt.Sprintf("Memory Limit: %s\n", FormatSize(c.Config.MemoryLimit)))
	}
	if c.Config.CPUQuota > 0 {
		sb.WriteString(fmt.Sprintf("CPU Quota:    %d%%\n", c.Config.CPUQuota))
	}
	if c.Config.PidsLimit > 0 {
		sb.WriteString(fmt.Sprintf("PIDs Limit:   %d\n", c.Config.PidsLimit))
	}
	return sb.String()
}
