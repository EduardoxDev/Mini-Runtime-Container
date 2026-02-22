//go:build linux

package cgroup

import (
	"testing"
)

func BenchmarkManager_Set(b *testing.B) {
	tmpDir := b.TempDir()
	mgr := NewManagerWithRoot("bench-container", tmpDir)
	resources := DefaultResources()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mgr.Set(resources)
	}
}

func BenchmarkManager_Apply(b *testing.B) {
	tmpDir := b.TempDir()
	mgr := NewManagerWithRoot("bench-container", tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mgr.Apply(12345)
	}
}

func BenchmarkManager_GetResources(b *testing.B) {
	tmpDir := b.TempDir()
	mgr := NewManagerWithRoot("bench-container", tmpDir)
	_ = mgr.Set(DefaultResources())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.GetResources()
	}
}

func BenchmarkManager_Lifecycle(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		mgr := NewManagerWithRoot("bench-lifecycle", tmpDir)
		_ = mgr.Set(DefaultResources())
		_ = mgr.Apply(12345)
		_, _ = mgr.GetResources()
		_ = mgr.Cleanup()
	}
}
