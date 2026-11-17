package persistence

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/BREJJNEVV/metrics/internal/repository/memory"
)

func TestSaveAndLoadMetrics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.json")
	ctx := context.Background()

	src := memory.Create()
	_ = src.Set(ctx, "Alloc", 123.45)
	_ = src.Set(ctx, "Sys", 1000)
	_ = src.Add(ctx, "PollCount", 5)

	if err := SaveMetrics(ctx, src, path); err != nil {
		t.Fatalf("SaveMetrics: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("file is empty")
	}

	dst := memory.Create()
	if err := LoadMetrics(ctx, dst, path); err != nil {
		t.Fatalf("LoadMetrics: %v", err)
	}

	g, ok := dst.GetGauge(ctx, "Alloc")
	if !ok || g != 123.45 {
		t.Errorf("Alloc: got %v, ok=%v", g, ok)
	}
	g, ok = dst.GetGauge(ctx, "Sys")
	if !ok || g != 1000 {
		t.Errorf("Sys: got %v, ok=%v", g, ok)
	}

	c, ok := dst.GetCounter(ctx, "PollCount")
	if !ok || c != 5 {
		t.Errorf("PollCount: got %v, ok=%v", c, ok)
	}
}

func TestLoadMetrics_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")
	ctx := context.Background()

	ms := memory.Create()
	if err := LoadMetrics(ctx, ms, path); err != nil {
		t.Errorf("missing file should not error, got: %v", err)
	}
	if len(ms.Gauges(ctx)) != 0 {
		t.Error("gauge storage should be empty")
	}
}

func TestLoadMetrics_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	ctx := context.Background()
	ms := memory.Create()
	if err := LoadMetrics(ctx, ms, path); err != nil {
		t.Errorf("empty file should not error, got: %v", err)
	}
}

func TestLoadMetrics_BadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	ctx := context.Background()
	ms := memory.Create()
	if err := LoadMetrics(ctx, ms, path); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestNewStorage_NoRestore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.json")
	ctx := context.Background()

	src := memory.Create()
	_ = src.Set(ctx, "X", 1)
	_ = SaveMetrics(ctx, src, path)

	ms, err := NewStorage(ctx, false, path)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	if len(ms.Gauges(ctx)) != 0 {
		t.Error("restore=false should not load data")
	}
}

func TestNewStorage_WithRestore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.json")
	ctx := context.Background()

	src := memory.Create()
	_ = src.Set(ctx, "X", 42)
	_ = SaveMetrics(ctx, src, path)

	ms, err := NewStorage(ctx, true, path)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	v, ok := ms.GetGauge(ctx, "X")
	if !ok || v != 42 {
		t.Errorf("X: got %v, ok=%v, want 42", v, ok)
	}
}

func TestSyncSaver(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.json")
	ctx := context.Background()

	base := memory.Create()
	saver := CreateSyncSaver(base, path)

	_ = saver.Set(ctx, "Alloc", 100)
	_ = saver.Add(ctx, "PollCount", 3)

	loaded := memory.Create()
	if err := LoadMetrics(ctx, loaded, path); err != nil {
		t.Fatalf("LoadMetrics: %v", err)
	}

	g, ok := loaded.GetGauge(ctx, "Alloc")
	if !ok || g != 100 {
		t.Errorf("Alloc: got %v", g)
	}
	c, ok := loaded.GetCounter(ctx, "PollCount")
	if !ok || c != 3 {
		t.Errorf("PollCount: got %v", c)
	}
}
