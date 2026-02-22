//go:build linux

package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "/bin/sh", cfg.Command)
	assert.Equal(t, "/rootfs", cfg.Rootfs)
	assert.Equal(t, "container", cfg.Hostname)
	assert.Equal(t, int64(256*1024*1024), cfg.MemoryLimit)
	assert.Equal(t, 50, cfg.CPUQuota)
	assert.Equal(t, 64, cfg.PidsLimit)
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := DefaultConfig()
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_MissingCommand(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Command = ""
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
}

func TestConfig_Validate_MissingRootfs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rootfs = ""
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rootfs path is required")
}

func TestConfig_Validate_NegativeMemory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MemoryLimit = -1
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "memory limit must be non-negative")
}

func TestConfig_Validate_InvalidCPUQuota(t *testing.T) {
	tests := []struct {
		name  string
		quota int
	}{
		{"negative", -1},
		{"over 100", 101},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.CPUQuota = tt.quota
			err := cfg.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "CPU quota must be between 0 and 100")
		})
	}
}

func TestConfig_Validate_NegativePids(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PidsLimit = -1
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pids limit must be non-negative")
}

func TestConfig_String(t *testing.T) {
	cfg := DefaultConfig()
	str := cfg.String()
	assert.Contains(t, str, "/bin/sh")
	assert.Contains(t, str, "256 MB")
	assert.Contains(t, str, "50%")
	assert.Contains(t, str, "64")
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	assert.Len(t, id1, 12)
	assert.Len(t, id2, 12)
	assert.NotEqual(t, id1, id2, "IDs should be unique")
}

func TestNew_ValidConfig(t *testing.T) {
	cfg := DefaultConfig()
	c, err := New(cfg)
	require.NoError(t, err)

	assert.NotEmpty(t, c.ID)
	assert.Equal(t, StatusCreated, c.Status)
	assert.Equal(t, cfg.Command, c.Config.Command)
}

func TestNew_InvalidConfig(t *testing.T) {
	cfg := Config{} // empty = invalid
	c, err := New(cfg)
	assert.Nil(t, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config")
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusCreated, "created"},
		{StatusRunning, "running"},
		{StatusStopped, "stopped"},
		{StatusFailed, "failed"},
		{Status(99), "unknown"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.status.String())
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{256 * 1024 * 1024, "256.0 MB"},
		{2 * 1024 * 1024 * 1024, "2.0 GB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, FormatSize(tt.bytes))
	}
}

func TestContainer_Info(t *testing.T) {
	cfg := DefaultConfig()
	c, err := New(cfg)
	require.NoError(t, err)

	info := c.Info()
	assert.Contains(t, info, c.ID)
	assert.Contains(t, info, "created")
	assert.Contains(t, info, "/bin/sh")
	assert.Contains(t, info, "256.0 MB")
	assert.Contains(t, info, "50%")
}

func TestContainer_Run_InvalidState(t *testing.T) {
	cfg := DefaultConfig()
	c, err := New(cfg)
	require.NoError(t, err)

	c.Status = StatusRunning
	err = c.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected created")
}

func TestContainer_Wait_NotStarted(t *testing.T) {
	cfg := DefaultConfig()
	c, err := New(cfg)
	require.NoError(t, err)

	err = c.Wait()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has not been started")
}

func TestContainer_Stop_NilProcess(t *testing.T) {
	cfg := DefaultConfig()
	c, err := New(cfg)
	require.NoError(t, err)

	// Stop on a container that hasn't started should be a no-op
	err = c.Stop()
	assert.NoError(t, err)
}
