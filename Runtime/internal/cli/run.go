//go:build linux

package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/gocontainer/internal/container"
)

var (
	memory   string
	cpuQuota int
	pidsMax  int
	hostname string
	rootfs   string
)

var runCmd = &cobra.Command{
	Use:   "run [flags] <command> [args...]",
	Short: "Run a command in an isolated container",
	Long: `Run creates a new container with Linux namespace isolation and
cgroup resource limits, then executes the specified command inside it.

Examples:
  gocontainer run /bin/sh
  gocontainer run --memory 128m --cpu 25 /bin/echo hello
  gocontainer run --memory 512m --pids 100 /bin/sh -c "ls -la"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runContainer,
}

func init() {
	runCmd.Flags().StringVarP(&memory, "memory", "m", "256m", "Memory limit (e.g., 128m, 1g)")
	runCmd.Flags().IntVarP(&cpuQuota, "cpu", "c", 50, "CPU quota as percentage (1-100)")
	runCmd.Flags().IntVarP(&pidsMax, "pids", "p", 64, "Maximum number of processes")
	runCmd.Flags().StringVar(&hostname, "hostname", "container", "Container hostname")
	runCmd.Flags().StringVar(&rootfs, "rootfs", "/rootfs", "Path to root filesystem")

	rootCmd.AddCommand(runCmd)
}

func runContainer(cmd *cobra.Command, args []string) error {
	memBytes, err := parseMemory(memory)
	if err != nil {
		return fmt.Errorf("invalid memory value %q: %w", memory, err)
	}

	cfg := container.Config{
		Command:     args[0],
		Args:        args[1:],
		Rootfs:      rootfs,
		Hostname:    hostname,
		MemoryLimit: memBytes,
		CPUQuota:    cpuQuota,
		PidsLimit:   pidsMax,
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Container Configuration:\n%s\n", cfg.String())
	}

	c, err := container.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Container %s created, starting...\n", c.ID)
	}

	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to run container: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Container %s running (PID: %d)\n", c.ID, c.PID)
	}

	// Wait for the container process to complete
	if err := c.Wait(); err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Container %s exited successfully\n", c.ID)
	}

	return nil
}

// parseMemory converts a human-readable memory string to bytes.
// Supports suffixes: b, k, m, g (case-insensitive).
func parseMemory(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty memory value")
	}

	var multiplier int64 = 1
	numStr := s

	switch {
	case strings.HasSuffix(s, "g"):
		multiplier = 1024 * 1024 * 1024
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "m"):
		multiplier = 1024 * 1024
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "k"):
		multiplier = 1024
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "b"):
		numStr = s[:len(s)-1]
	}

	val, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", numStr)
	}
	if val < 0 {
		return 0, fmt.Errorf("memory must be positive")
	}

	return val * multiplier, nil
}
