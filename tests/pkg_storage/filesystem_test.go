package pkg_storage_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/lifecycle"
	"github.com/JaimeStill/agent-lab/pkg/storage"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func tempStorageDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestNew_ValidConfig(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}

	sys, err := storage.New(cfg, testLogger())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if sys == nil {
		t.Fatal("New() returned nil system")
	}
}

func TestNew_EmptyBasePath(t *testing.T) {
	cfg := &storage.Config{BasePath: ""}

	_, err := storage.New(cfg, testLogger())
	if err == nil {
		t.Fatal("New() succeeded with empty BasePath, want error")
	}
}

func TestStart_CreatesDirectory(t *testing.T) {
	baseDir := tempStorageDir(t)
	targetDir := filepath.Join(baseDir, "nested", "storage")
	cfg := &storage.Config{BasePath: targetDir}

	sys, err := storage.New(cfg, testLogger())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	lc := lifecycle.New()
	if err := sys.Start(lc); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	lc.WaitForStartup()

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Error("Start() did not create storage directory")
	}
}

func TestStore_Retrieve_RoundTrip(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()
	key := "test/file.txt"
	data := []byte("hello world")

	if err := sys.Store(ctx, key, data); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	retrieved, err := sys.Retrieve(ctx, key)
	if err != nil {
		t.Fatalf("Retrieve() failed: %v", err)
	}

	if string(retrieved) != string(data) {
		t.Errorf("Retrieved data = %q, want %q", retrieved, data)
	}
}

func TestStore_CreatesNestedDirectories(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()
	key := "deeply/nested/path/file.txt"
	data := []byte("nested content")

	if err := sys.Store(ctx, key, data); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	expectedPath := filepath.Join(dir, "deeply", "nested", "path", "file.txt")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Store() did not create nested directories")
	}
}

func TestRetrieve_NotFound(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()
	_, err := sys.Retrieve(ctx, "nonexistent.txt")

	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Retrieve() error = %v, want %v", err, storage.ErrNotFound)
	}
}

func TestDelete_RemovesFile(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()
	key := "to-delete.txt"
	data := []byte("delete me")

	sys.Store(ctx, key, data)

	if err := sys.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	_, err := sys.Retrieve(ctx, key)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Error("File still exists after Delete()")
	}
}

func TestDelete_NonExistent_NoError(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()
	err := sys.Delete(ctx, "nonexistent.txt")

	if err != nil {
		t.Errorf("Delete() on non-existent file returned error: %v", err)
	}
}

func TestValidate_Exists(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()
	key := "exists.txt"
	sys.Store(ctx, key, []byte("content"))

	exists, err := sys.Validate(ctx, key)
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	if !exists {
		t.Error("Validate() = false for existing file, want true")
	}
}

func TestValidate_NotExists(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()
	exists, err := sys.Validate(ctx, "nonexistent.txt")

	if err != nil {
		t.Fatalf("Validate() returned error for non-existent file: %v", err)
	}

	if exists {
		t.Error("Validate() = true for non-existent file, want false")
	}
}

func TestInvalidKey_Empty(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()

	_, err := sys.Retrieve(ctx, "")
	if !errors.Is(err, storage.ErrInvalidKey) {
		t.Errorf("Retrieve('') error = %v, want %v", err, storage.ErrInvalidKey)
	}
}

func TestInvalidKey_PathTraversal(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()

	traversalKeys := []string{
		"../escape.txt",
		"foo/../../escape.txt",
		"/absolute/path.txt",
	}

	for _, key := range traversalKeys {
		t.Run(key, func(t *testing.T) {
			err := sys.Store(ctx, key, []byte("malicious"))
			if !errors.Is(err, storage.ErrInvalidKey) {
				t.Errorf("Store(%q) error = %v, want %v", key, err, storage.ErrInvalidKey)
			}
		})
	}
}

func TestStore_Overwrite(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()
	key := "overwrite.txt"

	sys.Store(ctx, key, []byte("original"))
	sys.Store(ctx, key, []byte("updated"))

	data, _ := sys.Retrieve(ctx, key)
	if string(data) != "updated" {
		t.Errorf("Retrieved = %q after overwrite, want %q", data, "updated")
	}
}

func TestStart_DirectoryAlreadyExists(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	if err := sys.Start(lc); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	lc.WaitForStartup()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Directory should exist after Start()")
	}
}

func TestStore_EmptyData(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()
	key := "empty.txt"

	if err := sys.Store(ctx, key, []byte{}); err != nil {
		t.Fatalf("Store() empty data failed: %v", err)
	}

	data, err := sys.Retrieve(ctx, key)
	if err != nil {
		t.Fatalf("Retrieve() failed: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("Retrieved data length = %d, want 0", len(data))
	}
}

func TestValidate_InvalidKey(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()

	exists, err := sys.Validate(ctx, "")
	if !errors.Is(err, storage.ErrInvalidKey) {
		t.Errorf("Validate('') error = %v, want %v", err, storage.ErrInvalidKey)
	}
	if exists {
		t.Error("Validate('') returned true, want false")
	}
}

func TestDelete_InvalidKey(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()

	err := sys.Delete(ctx, "../escape")
	if !errors.Is(err, storage.ErrInvalidKey) {
		t.Errorf("Delete('../escape') error = %v, want %v", err, storage.ErrInvalidKey)
	}
}

func TestStore_InvalidKey(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()

	err := sys.Store(ctx, "", []byte("data"))
	if !errors.Is(err, storage.ErrInvalidKey) {
		t.Errorf("Store('') error = %v, want %v", err, storage.ErrInvalidKey)
	}
}

func TestDelete_CleansUpEmptyParentDirectory(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()
	key := "documents/abc-123/file.pdf"

	sys.Store(ctx, key, []byte("pdf content"))

	parentDir := filepath.Join(dir, "documents", "abc-123")
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Fatal("Parent directory should exist after Store()")
	}

	if err := sys.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	if _, err := os.Stat(parentDir); !os.IsNotExist(err) {
		t.Error("Empty parent directory should be removed after Delete()")
	}

	docsDir := filepath.Join(dir, "documents")
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		t.Error("Non-empty ancestor directory should not be removed")
	}
}

func TestDelete_PreservesNonEmptyParentDirectory(t *testing.T) {
	dir := tempStorageDir(t)
	cfg := &storage.Config{BasePath: dir}
	sys, _ := storage.New(cfg, testLogger())

	lc := lifecycle.New()
	sys.Start(lc)
	lc.WaitForStartup()

	ctx := context.Background()

	sys.Store(ctx, "shared/file1.txt", []byte("content1"))
	sys.Store(ctx, "shared/file2.txt", []byte("content2"))

	if err := sys.Delete(ctx, "shared/file1.txt"); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	sharedDir := filepath.Join(dir, "shared")
	if _, err := os.Stat(sharedDir); os.IsNotExist(err) {
		t.Error("Non-empty parent directory should not be removed")
	}

	if _, err := sys.Retrieve(ctx, "shared/file2.txt"); err != nil {
		t.Error("Other file in directory should still exist")
	}
}
