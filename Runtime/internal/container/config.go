//go:build linux

package container

import (
	"fmt"
	"strings"
)

// Config holds all configuration for a container instance.
type Config struct {
	// Command is the executable to run inside the container.
	Command string

	// Args are the arguments to pass to the command.
	Args []string

	// Rootfs is the path to the container's root filesystem.
	Rootfs string

	// Hostname is the hostname to set inside the container's UTS namespace.
	Hostname string

	// MemoryLimit is the hard memory limit in bytes. 0 means no limit.
	MemoryLimit int64

	// CPUQuota is the CPU quota as a percentage (e.g., 50 = 50% of one core).
	CPUQuota int

	// PidsLimit is the maximum number of processes. 0 means no limit.
	PidsLimit int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Command:     "/bin/sh",
		Rootfs:      "/rootfs",
		Hostname:    "container",
		MemoryLimit: 256 * 1024 * 1024, // 256 MB
		CPUQuota:    50,                // 50% CPU
		PidsLimit:   64,
	}
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.Command == "" {
		return fmt.Errorf("command is required")
	}
	if c.Rootfs == "" {
		return fmt.Errorf("rootfs path is required")
	}
	if c.MemoryLimit < 0 {
		return fmt.Errorf("memory limit must be non-negative, got %d", c.MemoryLimit)
	}
	if c.CPUQuota < 0 || c.CPUQuota > 100 {
		return fmt.Errorf("CPU quota must be between 0 and 100, got %d", c.CPUQuota)
	}
	if c.PidsLimit < 0 {
		return fmt.Errorf("pids limit must be non-negative, got %d", c.PidsLimit)
	}
	return nil
}

// String returns a human-readable representation of the config.
func (c Config) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Command: %s %s\n", c.Command, strings.Join(c.Args, " ")))
	sb.WriteString(fmt.Sprintf("Rootfs:  %s\n", c.Rootfs))
	sb.WriteString(fmt.Sprintf("Host:    %s\n", c.Hostname))
	if c.MemoryLimit > 0 {
		sb.WriteString(fmt.Sprintf("Memory:  %d MB\n", c.MemoryLimit/(1024*1024)))
	}
	if c.CPUQuota > 0 {
		sb.WriteString(fmt.Sprintf("CPU:     %d%%\n", c.CPUQuota))
	}
	if c.PidsLimit > 0 {
		sb.WriteString(fmt.Sprintf("PIDs:    %d\n", c.PidsLimit))
	}
	return sb.String()
}
