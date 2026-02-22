//go:build linux

// Package cgroup provides cgroup v2 resource management for container processes.
// It creates and manages cgroup hierarchies to enforce CPU, memory, and process
// count limits on containerized processes.
package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// CgroupRoot is the default cgroup v2 unified hierarchy mount point.
	CgroupRoot = "/sys/fs/cgroup"
	// CgroupPrefix is the prefix for all gocontainer cgroups.
	CgroupPrefix = "gocontainer"
)

// Resources defines the resource limits to apply to a container.
type Resources struct {
	// MemoryMax is the hard memory limit in bytes. 0 means no limit.
	MemoryMax int64
	// CPUQuota is the CPU quota as a percentage (e.g., 50 = 50% of one core).
	// Internally converted to cpu.max format (quota period).
	CPUQuota int
	// CPUPeriod is the CPU period in microseconds. Default: 100000 (100ms).
	CPUPeriod int
	// PidsMax is the maximum number of processes. 0 means no limit.
	PidsMax int
}

// DefaultResources returns sensible default resource limits.
// 256MB memory, 50% CPU, 64 processes.
func DefaultResources() Resources {
	return Resources{
		MemoryMax: 256 * 1024 * 1024, // 256 MB
		CPUQuota:  50,                // 50% of one CPU
		CPUPeriod: 100000,            // 100ms (standard period)
		PidsMax:   64,
	}
}

// Manager handles cgroup creation, process assignment, and cleanup.
type Manager struct {
	// Path is the full path to the cgroup directory.
	Path string
	// ID is the unique container identifier used in the cgroup path.
	ID string
	// CgroupRoot allows overriding the cgroup filesystem root (for testing).
	CgroupRoot string
}

// NewManager creates a new cgroup manager for the given container ID.
func NewManager(containerID string) *Manager {
	return &Manager{
		ID:         containerID,
		CgroupRoot: CgroupRoot,
		Path:       filepath.Join(CgroupRoot, CgroupPrefix, containerID),
	}
}

// NewManagerWithRoot creates a new cgroup manager with a custom root (for testing).
func NewManagerWithRoot(containerID, root string) *Manager {
	return &Manager{
		ID:         containerID,
		CgroupRoot: root,
		Path:       filepath.Join(root, CgroupPrefix, containerID),
	}
}

// Apply creates the cgroup directory and adds the given PID to it.
func (m *Manager) Apply(pid int) error {
	// Create cgroup directory
	if err := os.MkdirAll(m.Path, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup directory %s: %w", m.Path, err)
	}

	// Write PID to cgroup.procs
	procsPath := filepath.Join(m.Path, "cgroup.procs")
	if err := writeFile(procsPath, strconv.Itoa(pid)); err != nil {
		return fmt.Errorf("failed to add pid %d to cgroup: %w", pid, err)
	}

	return nil
}

// Set writes the resource limits to the appropriate cgroup control files.
func (m *Manager) Set(resources Resources) error {
	if err := os.MkdirAll(m.Path, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup directory: %w", err)
	}

	// Set memory limit
	if resources.MemoryMax > 0 {
		memPath := filepath.Join(m.Path, "memory.max")
		if err := writeFile(memPath, strconv.FormatInt(resources.MemoryMax, 10)); err != nil {
			return fmt.Errorf("failed to set memory limit: %w", err)
		}
	}

	// Set CPU limit (cpu.max format: "quota period")
	if resources.CPUQuota > 0 {
		period := resources.CPUPeriod
		if period == 0 {
			period = 100000 // 100ms default
		}
		quota := (resources.CPUQuota * period) / 100
		cpuPath := filepath.Join(m.Path, "cpu.max")
		cpuValue := fmt.Sprintf("%d %d", quota, period)
		if err := writeFile(cpuPath, cpuValue); err != nil {
			return fmt.Errorf("failed to set CPU limit: %w", err)
		}
	}

	// Set PID limit
	if resources.PidsMax > 0 {
		pidsPath := filepath.Join(m.Path, "pids.max")
		if err := writeFile(pidsPath, strconv.Itoa(resources.PidsMax)); err != nil {
			return fmt.Errorf("failed to set pids limit: %w", err)
		}
	}

	return nil
}

// GetResources reads the current resource limits from the cgroup control files.
func (m *Manager) GetResources() (Resources, error) {
	var resources Resources

	// Read memory limit
	memPath := filepath.Join(m.Path, "memory.max")
	if data, err := os.ReadFile(memPath); err == nil {
		val := strings.TrimSpace(string(data))
		if val != "max" {
			if mem, err := strconv.ParseInt(val, 10, 64); err == nil {
				resources.MemoryMax = mem
			}
		}
	}

	// Read CPU limit
	cpuPath := filepath.Join(m.Path, "cpu.max")
	if data, err := os.ReadFile(cpuPath); err == nil {
		parts := strings.Fields(strings.TrimSpace(string(data)))
		if len(parts) == 2 && parts[0] != "max" {
			quota, _ := strconv.Atoi(parts[0])
			period, _ := strconv.Atoi(parts[1])
			if period > 0 {
				resources.CPUQuota = (quota * 100) / period
				resources.CPUPeriod = period
			}
		}
	}

	// Read PID limit
	pidsPath := filepath.Join(m.Path, "pids.max")
	if data, err := os.ReadFile(pidsPath); err == nil {
		val := strings.TrimSpace(string(data))
		if val != "max" {
			if pids, err := strconv.Atoi(val); err == nil {
				resources.PidsMax = pids
			}
		}
	}

	return resources, nil
}

// Cleanup removes the cgroup directory. The cgroup must have no running
// processes before it can be removed.
func (m *Manager) Cleanup() error {
	return os.RemoveAll(m.Path)
}

// writeFile writes a value to a file, creating it if necessary.
func writeFile(path, value string) error {
	return os.WriteFile(path, []byte(value), 0644)
}
