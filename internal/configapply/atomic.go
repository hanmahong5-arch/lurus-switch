package configapply

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// WriteAtomic writes content to path via tmp file + fsync + rename. On Windows
// rename races against AV scanners and Explorer file locks; we retry up to
// 3 times with backoff before giving up.
func WriteAtomic(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-configapply-*.swp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("fsync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil && runtime.GOOS != "windows" {
		cleanup()
		return fmt.Errorf("chmod temp: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := os.Rename(tmpName, path); err == nil {
			return nil
		} else {
			lastErr = err
			time.Sleep(time.Duration(200*(attempt+1)) * time.Millisecond)
		}
	}
	cleanup()
	return fmt.Errorf("rename after 3 retries: %w", lastErr)
}

// ReadFileOrEmpty reads the file at path, returning empty string if missing.
// Used to capture pre-state for the Before side of a FileChange.
func ReadFileOrEmpty(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// FileSizeMatches verifies a file exists at path and has the expected byte length.
// Used during PhaseVerify to catch partial writes or truncation by AV.
func FileSizeMatches(path string, expected int) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return stat.Size() == int64(expected), nil
}

// CopyFile is the cross-volume fallback when os.Rename fails (different drive
// letters on Windows). Reads source fully into memory; OK for config files.
func CopyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
