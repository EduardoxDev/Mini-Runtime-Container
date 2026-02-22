//go:build linux

package namespace

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// SetupMount prepares the container's mount namespace by mounting essential
// filesystems (proc, tmpfs, devtmpfs) and performing a pivot_root to isolate
// the container's filesystem view from the host.
func SetupMount(rootfs string) error {
	// First, make the mount namespace private so our changes don't propagate
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to make root private: %w", err)
	}

	// Bind mount the rootfs onto itself (required for pivot_root)
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to bind mount rootfs: %w", err)
	}

	// Perform pivot_root
	if err := PivotRoot(rootfs); err != nil {
		return fmt.Errorf("pivot_root failed: %w", err)
	}

	// Mount /proc filesystem (essential for process information)
	if err := mountProc(); err != nil {
		return fmt.Errorf("failed to mount /proc: %w", err)
	}

	// Mount /tmp as tmpfs
	if err := mountTmpfs("/tmp"); err != nil {
		return fmt.Errorf("failed to mount /tmp: %w", err)
	}

	// Mount /dev as devtmpfs
	if err := mountDev(); err != nil {
		return fmt.Errorf("failed to mount /dev: %w", err)
	}

	// Mount /sys as sysfs (read-only)
	if err := mountSys(); err != nil {
		return fmt.Errorf("failed to mount /sys: %w", err)
	}

	return nil
}

// PivotRoot changes the root filesystem to the specified directory using
// the pivot_root(2) system call. This is more secure than chroot because
// it completely replaces the root mount point.
func PivotRoot(rootfs string) error {
	// Create a temporary directory for the old root
	pivotDir := filepath.Join(rootfs, ".pivot_root")
	if err := os.MkdirAll(pivotDir, 0700); err != nil {
		return fmt.Errorf("failed to create pivot dir: %w", err)
	}

	// pivot_root moves the root mount to pivotDir and makes rootfs the new root
	if err := syscall.PivotRoot(rootfs, pivotDir); err != nil {
		return fmt.Errorf("pivot_root syscall failed: %w", err)
	}

	// Change directory to the new root
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("failed to chdir to new root: %w", err)
	}

	// Unmount the old root and remove the pivot directory
	pivotDir = "/.pivot_root"
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("failed to unmount old root: %w", err)
	}

	return os.RemoveAll(pivotDir)
}

// mountProc mounts the proc filesystem at /proc.
func mountProc() error {
	target := "/proc"
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	return syscall.Mount("proc", target, "proc", 0, "")
}

// mountTmpfs mounts a tmpfs at the given target path.
func mountTmpfs(target string) error {
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	return syscall.Mount("tmpfs", target, "tmpfs", 0, "")
}

// mountDev mounts a minimal /dev with tmpfs and creates essential device nodes.
func mountDev() error {
	target := "/dev"
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	if err := syscall.Mount("tmpfs", target, "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755"); err != nil {
		return err
	}

	// Create essential device nodes
	devices := []struct {
		path  string
		mode  uint32
		major uint32
		minor uint32
	}{
		{"/dev/null", syscall.S_IFCHR | 0666, 1, 3},
		{"/dev/zero", syscall.S_IFCHR | 0666, 1, 5},
		{"/dev/random", syscall.S_IFCHR | 0666, 1, 8},
		{"/dev/urandom", syscall.S_IFCHR | 0666, 1, 9},
		{"/dev/tty", syscall.S_IFCHR | 0666, 5, 0},
	}

	for _, dev := range devices {
		devNum := int(dev.major*256 + dev.minor)
		if err := syscall.Mknod(dev.path, dev.mode, devNum); err != nil {
			// Non-fatal: device creation may fail in unprivileged containers
			continue
		}
	}

	// Symlinks
	symlinks := map[string]string{
		"/dev/stdin":  "/proc/self/fd/0",
		"/dev/stdout": "/proc/self/fd/1",
		"/dev/stderr": "/proc/self/fd/2",
		"/dev/fd":     "/proc/self/fd",
	}
	for link, target := range symlinks {
		_ = os.Symlink(target, link)
	}

	return nil
}

// mountSys mounts the sysfs filesystem at /sys as read-only.
func mountSys() error {
	target := "/sys"
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	return syscall.Mount("sysfs", target, "sysfs", syscall.MS_RDONLY, "")
}
