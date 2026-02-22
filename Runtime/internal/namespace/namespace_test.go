//go:build linux

package namespace

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultFlags(t *testing.T) {
	flags := DefaultFlags()
	assert.True(t, flags.UTS, "UTS namespace should be enabled by default")
	assert.True(t, flags.PID, "PID namespace should be enabled by default")
	assert.True(t, flags.Mount, "Mount namespace should be enabled by default")
	assert.True(t, flags.IPC, "IPC namespace should be enabled by default")
	assert.True(t, flags.Net, "Net namespace should be enabled by default")
	assert.True(t, flags.User, "User namespace should be enabled by default")
}

func TestCloneFlags_AllEnabled(t *testing.T) {
	flags := DefaultFlags()
	result := flags.CloneFlags()

	expected := uintptr(
		syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWUSER,
	)
	assert.Equal(t, expected, result, "All clone flags should be set")
}

func TestCloneFlags_Selective(t *testing.T) {
	tests := []struct {
		name     string
		flags    Flags
		expected uintptr
	}{
		{
			name:     "UTS only",
			flags:    Flags{UTS: true},
			expected: syscall.CLONE_NEWUTS,
		},
		{
			name:     "PID only",
			flags:    Flags{PID: true},
			expected: syscall.CLONE_NEWPID,
		},
		{
			name:     "Mount only",
			flags:    Flags{Mount: true},
			expected: syscall.CLONE_NEWNS,
		},
		{
			name:     "None enabled",
			flags:    Flags{},
			expected: 0,
		},
		{
			name:     "PID and Net",
			flags:    Flags{PID: true, Net: true},
			expected: uintptr(syscall.CLONE_NEWPID | syscall.CLONE_NEWNET),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.flags.CloneFlags())
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("test-container")
	assert.Equal(t, "test-container", config.Hostname)
	assert.True(t, config.Flags.UTS)
	require.Len(t, config.UIDMappings, 1)
	assert.Equal(t, 0, config.UIDMappings[0].ContainerID)
	assert.Equal(t, 0, config.UIDMappings[0].HostID)
	assert.Equal(t, 1, config.UIDMappings[0].Size)
	require.Len(t, config.GIDMappings, 1)
	assert.Equal(t, 0, config.GIDMappings[0].ContainerID)
}

func TestSetupSysProcAttr(t *testing.T) {
	config := DefaultConfig("myhost")
	attr := SetupSysProcAttr(config)

	// Verify clone flags include all namespaces
	assert.NotEqual(t, uintptr(0), attr.Cloneflags&syscall.CLONE_NEWUTS)
	assert.NotEqual(t, uintptr(0), attr.Cloneflags&syscall.CLONE_NEWPID)
	assert.NotEqual(t, uintptr(0), attr.Cloneflags&syscall.CLONE_NEWNS)
	assert.NotEqual(t, uintptr(0), attr.Cloneflags&syscall.CLONE_NEWIPC)
	assert.NotEqual(t, uintptr(0), attr.Cloneflags&syscall.CLONE_NEWNET)
	assert.NotEqual(t, uintptr(0), attr.Cloneflags&syscall.CLONE_NEWUSER)

	// Verify UID mappings
	require.Len(t, attr.UidMappings, 1)
	assert.Equal(t, 0, attr.UidMappings[0].ContainerID)
	assert.Equal(t, 0, attr.UidMappings[0].HostID)
	assert.Equal(t, 1, attr.UidMappings[0].Size)

	// Verify GID mappings
	require.Len(t, attr.GidMappings, 1)
	assert.Equal(t, 0, attr.GidMappings[0].ContainerID)
}

func TestSetupSysProcAttr_NoUserNamespace(t *testing.T) {
	config := DefaultConfig("myhost")
	config.Flags.User = false
	config.UIDMappings = nil
	config.GIDMappings = nil

	attr := SetupSysProcAttr(config)

	// User namespace flag should not be set
	assert.Equal(t, uintptr(0), attr.Cloneflags&syscall.CLONE_NEWUSER)
	assert.Nil(t, attr.UidMappings)
	assert.Nil(t, attr.GidMappings)
}

func TestSetupSysProcAttr_MultipleUIDMappings(t *testing.T) {
	config := DefaultConfig("myhost")
	config.UIDMappings = []UIDMapping{
		{ContainerID: 0, HostID: 1000, Size: 1},
		{ContainerID: 1, HostID: 100000, Size: 65536},
	}

	attr := SetupSysProcAttr(config)
	require.Len(t, attr.UidMappings, 2)
	assert.Equal(t, 1000, attr.UidMappings[0].HostID)
	assert.Equal(t, 100000, attr.UidMappings[1].HostID)
	assert.Equal(t, 65536, attr.UidMappings[1].Size)
}
