package versioncheck

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const maxCacheSize = 64 << 10

type cacheEntry struct {
	Latest    string    `json:"latest"`
	CheckedAt time.Time `json:"checkedAt"`
}

func cacheFile() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("user cache dir: %w", err)
	}

	return filepath.Join(dir, "edgecenter", "version-check.json"), nil
}

func readCache(path string, now time.Time) (string, bool) {
	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return "", false
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxCacheSize))
	if err != nil {
		return "", false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", false
	}

	if entry.CheckedAt.IsZero() {
		return "", false
	}

	age := now.Sub(entry.CheckedAt)
	if age < 0 || age > cacheTTL {
		return "", false
	}

	return entry.Latest, true
}

func writeCache(path, latest string, now time.Time) {
	data, err := json.Marshal(cacheEntry{Latest: latest, CheckedAt: now})
	if err != nil {
		return
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}

	tmp, err := os.CreateTemp(dir, "version-check-*.json")
	if err != nil {
		return
	}
	name := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)

		return
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)

		return
	}

	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
	}
}
