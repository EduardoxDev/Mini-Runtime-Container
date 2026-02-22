//go:build linux

package cgroup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestCgroup(t *testing.T) (*Manager, string) {
	t.Helper()
	tmpDir := t.TempDir()
	mgr := NewManagerWithRoot("test-container-001", tmpDir)
	return mgr, tmpDir
}

func TestNewManager(t *testing.T) {
	mgr := NewManager("abc123")
	assert.Equal(t, "abc123", mgr.ID)
	assert.Equal(t, CgroupRoot, mgr.CgroupRoot)
	assert.Contains(t, mgr.Path, "gocontainer/abc123")
}

func TestNewManagerWithRoot(t *testing.T) {
	mgr := NewManagerWithRoot("abc123", "/tmp/test-cgroup")
	assert.Equal(t, "/tmp/test-cgroup", mgr.CgroupRoot)
	assert.Equal(t, filepath.Join("/tmp/test-cgroup", CgroupPrefix, "abc123"), mgr.Path)
}

func TestDefaultResources(t *testing.T) {
	res := DefaultResources()
	assert.Equal(t, int64(256*1024*1024), res.MemoryMax)
	assert.Equal(t, 50, res.CPUQuota)
	assert.Equal(t, 100000, res.CPUPeriod)
	assert.Equal(t, 64, res.PidsMax)
}

func TestManager_Set_MemoryLimit(t *testing.T) {
	mgr, _ := setupTestCgroup(t)

	err := mgr.Set(Resources{MemoryMax: 128 * 1024 * 1024})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(mgr.Path, "memory.max"))
	require.NoError(t, err)
	assert.Equal(t, "134217728", string(data)) // 128 * 1024 * 1024
}

func TestManager_Set_CPULimit(t *testing.T) {
	mgr, _ := setupTestCgroup(t)

	err := mgr.Set(Resources{CPUQuota: 25, CPUPeriod: 100000})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(mgr.Path, "cpu.max"))
	require.NoError(t, err)
	assert.Equal(t, "25000 100000", string(data))
}

func TestManager_Set_PidsLimit(t *testing.T) {
	mgr, _ := setupTestCgroup(t)

	err := mgr.Set(Resources{PidsMax: 32})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(mgr.Path, "pids.max"))
	require.NoError(t, err)
	assert.Equal(t, "32", string(data))
}

func TestManager_Set_AllResources(t *testing.T) {
	mgr, _ := setupTestCgroup(t)

	resources := Resources{
		MemoryMax: 512 * 1024 * 1024,
		CPUQuota:  75,
		CPUPeriod: 100000,
		PidsMax:   128,
	}

	err := mgr.Set(resources)
	require.NoError(t, err)

	// Verify memory
	memData, err := os.ReadFile(filepath.Join(mgr.Path, "memory.max"))
	require.NoError(t, err)
	assert.Equal(t, "536870912", string(memData))

	// Verify CPU
	cpuData, err := os.ReadFile(filepath.Join(mgr.Path, "cpu.max"))
	require.NoError(t, err)
	assert.Equal(t, "75000 100000", string(cpuData))

	// Verify PIDs
	pidsData, err := os.ReadFile(filepath.Join(mgr.Path, "pids.max"))
	require.NoError(t, err)
	assert.Equal(t, "128", string(pidsData))
}

func TestManager_Apply(t *testing.T) {
	mgr, _ := setupTestCgroup(t)

	err := mgr.Apply(12345)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(mgr.Path, "cgroup.procs"))
	require.NoError(t, err)
	assert.Equal(t, "12345", string(data))
}

func TestManager_GetResources(t *testing.T) {
	mgr, _ := setupTestCgroup(t)

	// Write test data
	err := mgr.Set(Resources{
		MemoryMax: 256 * 1024 * 1024,
		CPUQuota:  50,
		CPUPeriod: 100000,
		PidsMax:   64,
	})
	require.NoError(t, err)

	// Read it back
	resources, err := mgr.GetResources()
	require.NoError(t, err)
	assert.Equal(t, int64(256*1024*1024), resources.MemoryMax)
	assert.Equal(t, 50, resources.CPUQuota)
	assert.Equal(t, 100000, resources.CPUPeriod)
	assert.Equal(t, 64, resources.PidsMax)
}

func TestManager_Cleanup(t *testing.T) {
	mgr, _ := setupTestCgroup(t)

	err := mgr.Set(Resources{MemoryMax: 1024})
	require.NoError(t, err)

	// Verify directory exists
	_, err = os.Stat(mgr.Path)
	require.NoError(t, err)

	// Cleanup
	err = mgr.Cleanup()
	require.NoError(t, err)

	// Verify directory is removed
	_, err = os.Stat(mgr.Path)
	assert.True(t, os.IsNotExist(err))
}

func TestManager_Set_ZeroValues_Skipped(t *testing.T) {
	mgr, _ := setupTestCgroup(t)

	// Setting zero values should not create files
	err := mgr.Set(Resources{})
	require.NoError(t, err)

	// Directory should exist but no control files
	_, err = os.ReadFile(filepath.Join(mgr.Path, "memory.max"))
	assert.True(t, os.IsNotExist(err))

	_, err = os.ReadFile(filepath.Join(mgr.Path, "cpu.max"))
	assert.True(t, os.IsNotExist(err))

	_, err = os.ReadFile(filepath.Join(mgr.Path, "pids.max"))
	assert.True(t, os.IsNotExist(err))
}
