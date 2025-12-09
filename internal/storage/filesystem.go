package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
)

// filesystem implements System using the local filesystem.
// It stores blobs as files under a configurable base path,
// with keys mapping directly to relative file paths.
type filesystem struct {
	basePath string
	logger   *slog.Logger
}

// New creates a new filesystem storage system.
// The base path is resolved to an absolute path during construction.
// Directory creation is deferred to Start() for lifecycle integration.
func New(cfg *config.StorageConfig, logger *slog.Logger) (System, error) {
	if cfg.BasePath == "" {
		return nil, fmt.Errorf("base_path required")
	}

	absPath, err := filepath.Abs(cfg.BasePath)
	if err != nil {
		return nil, fmt.Errorf("resolve base_path: %w", err)
	}

	return &filesystem{
		basePath: absPath,
		logger:   logger.With("system", "storage"),
	}, nil
}

func (f *filesystem) Start(lc *lifecycle.Coordinator) error {
	f.logger.Info("starting storage system", "base_path", f.basePath)

	lc.OnStartup(func() {
		if err := os.MkdirAll(f.basePath, 0755); err != nil {
			f.logger.Error("storage initialization failed", "error", err)
			return
		}
		f.logger.Info("storage directory initialized")
	})

	return nil
}

func (f *filesystem) Store(ctx context.Context, key string, data []byte) error {
	path, err := f.fullPath(key)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

func (f *filesystem) Retrieve(ctx context.Context, key string) ([]byte, error) {
	path, err := f.fullPath(key)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
		if errors.Is(err, fs.ErrPermission) {
			return nil, ErrPermissionDenied
		}
		return nil, fmt.Errorf("read file: %w", err)
	}

	return data, nil
}

func (f *filesystem) Delete(ctx context.Context, key string) error {
	path, err := f.fullPath(key)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)

	if err := os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if errors.Is(err, fs.ErrPermission) {
			return ErrPermissionDenied
		}
		return fmt.Errorf("remove file: %w", err)
	}

	if dir != f.basePath && strings.HasPrefix(dir, f.basePath) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			f.logger.Warn("failed to read directory for cleanup", "dir", dir, "error", err)
			return nil
		}

		if len(entries) == 0 {
			if err := os.Remove(dir); err != nil && !errors.Is(err, fs.ErrNotExist) {
				f.logger.Warn("failed to remove empty directory", "dir", dir, "error", err)
			}
		}
	}

	return nil
}

func (f *filesystem) Validate(ctx context.Context, key string) (bool, error) {
	path, err := f.fullPath(key)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		if errors.Is(err, fs.ErrPermission) {
			return false, ErrPermissionDenied
		}
		return false, fmt.Errorf("stat file: %w", err)
	}

	return true, nil
}

func (f *filesystem) fullPath(key string) (string, error) {
	if key == "" {
		return "", ErrInvalidKey
	}

	cleaned := filepath.Clean(key)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", ErrInvalidKey
	}

	fullPath := filepath.Join(f.basePath, cleaned)

	if !strings.HasPrefix(fullPath, f.basePath) {
		return "", ErrInvalidKey
	}

	return fullPath, nil
}
