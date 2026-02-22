//go:build linux

// Package namespace provides Linux namespace isolation for container processes.
// It configures clone flags and UID/GID mappings to create fully isolated
// process environments using the Linux kernel's namespace features.
package namespace

import (
	"syscall"
)

// Flags defines which Linux namespaces to isolate.
type Flags struct {
	UTS   bool // Hostname and domain name isolation
	PID   bool // Process ID isolation
	Mount bool // Mount point isolation
	IPC   bool // Inter-process communication isolation
	Net   bool // Network stack isolation
	User  bool // User and group ID isolation
}

// DefaultFlags returns a Flags struct with all namespaces enabled.
func DefaultFlags() Flags {
	return Flags{
		UTS:   true,
		PID:   true,
		Mount: true,
		IPC:   true,
		Net:   true,
		User:  true,
	}
}

// CloneFlags converts namespace Flags into the corresponding Linux clone flags
// used by the clone(2) system call. Each enabled namespace creates a new
// isolated instance for the child process.
func (f Flags) CloneFlags() uintptr {
	var flags uintptr
	if f.UTS {
		flags |= syscall.CLONE_NEWUTS
	}
	if f.PID {
		flags |= syscall.CLONE_NEWPID
	}
	if f.Mount {
		flags |= syscall.CLONE_NEWNS
	}
	if f.IPC {
		flags |= syscall.CLONE_NEWIPC
	}
	if f.Net {
		flags |= syscall.CLONE_NEWNET
	}
	if f.User {
		flags |= syscall.CLONE_NEWUSER
	}
	return flags
}

// UIDMapping represents a user ID mapping from host to container.
type UIDMapping struct {
	ContainerID int
	HostID      int
	Size        int
}

// GIDMapping represents a group ID mapping from host to container.
type GIDMapping struct {
	ContainerID int
	HostID      int
	Size        int
}

// Config holds the full namespace configuration for a container process.
type Config struct {
	Flags       Flags
	UIDMappings []UIDMapping
	GIDMappings []GIDMapping
	Hostname    string
}

// DefaultConfig returns a Config with all namespaces enabled and default
// UID/GID mappings that map root inside the container to the current user
// on the host.
func DefaultConfig(hostname string) Config {
	return Config{
		Flags:    DefaultFlags(),
		Hostname: hostname,
		UIDMappings: []UIDMapping{
			{ContainerID: 0, HostID: 0, Size: 1},
		},
		GIDMappings: []GIDMapping{
			{ContainerID: 0, HostID: 0, Size: 1},
		},
	}
}

// SetupSysProcAttr creates a syscall.SysProcAttr configured with the
// appropriate clone flags and UID/GID mappings for namespace isolation.
// This is applied to exec.Cmd.SysProcAttr before starting the child process.
func SetupSysProcAttr(config Config) *syscall.SysProcAttr {
	attr := &syscall.SysProcAttr{
		Cloneflags: config.Flags.CloneFlags(),
	}

	// Set UID mappings
	if len(config.UIDMappings) > 0 {
		attr.UidMappings = make([]syscall.SysProcIDMap, len(config.UIDMappings))
		for i, m := range config.UIDMappings {
			attr.UidMappings[i] = syscall.SysProcIDMap{
				ContainerID: m.ContainerID,
				HostID:      m.HostID,
				Size:        m.Size,
			}
		}
	}

	// Set GID mappings
	if len(config.GIDMappings) > 0 {
		attr.GidMappings = make([]syscall.SysProcIDMap, len(config.GIDMappings))
		for i, m := range config.GIDMappings {
			attr.GidMappings[i] = syscall.SysProcIDMap{
				ContainerID: m.ContainerID,
				HostID:      m.HostID,
				Size:        m.Size,
			}
		}
	}

	// Ensure new user namespace is created before other namespaces
	if config.Flags.User {
		attr.Cloneflags |= syscall.CLONE_NEWUSER
	}

	return attr
}

// SetHostname sets the hostname inside the UTS namespace.
// Must be called from within the new namespace (child process).
func SetHostname(hostname string) error {
	if hostname == "" {
		hostname = "container"
	}
	return syscall.Sethostname([]byte(hostname))
}
