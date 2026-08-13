package versioncheck

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "version-check.json")
	now := time.Now()

	writeCache(path, latestRelease, now)

	got, ok := readCache(path, now.Add(time.Hour))
	if !ok {
		t.Fatal("cache miss right after write")
	}
	if got != latestRelease {
		t.Errorf("cached = %q, want %q", got, latestRelease)
	}
}

func TestCacheStoresNegativeResult(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "version-check.json")
	now := time.Now()

	writeCache(path, "", now)

	got, ok := readCache(path, now.Add(time.Minute))
	if !ok {
		t.Fatal("negative result must be cached")
	}
	if got != "" {
		t.Errorf("cached = %q, want empty", got)
	}
}

func TestCacheExpiry(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "version-check.json")
	now := time.Now()
	writeCache(path, latestRelease, now)

	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{name: "fresh", at: now.Add(time.Hour), want: true},
		{name: "edge of ttl", at: now.Add(cacheTTL), want: true},
		{name: "expired", at: now.Add(cacheTTL + time.Second), want: false},
		{name: "clock moved back", at: now.Add(-time.Hour), want: false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if _, ok := readCache(path, c.at); ok != c.want {
				t.Errorf("ok = %v, want %v", ok, c.want)
			}
		})
	}
}

func TestReadCacheRejectsUnusableFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cases := []struct {
		name    string
		content string
		write   bool
	}{
		{name: "missing file", write: false},
		{name: "broken json", content: `{"latest":`, write: true},
		{name: "zero timestamp", content: `{"latest":"0.14.7"}`, write: true},
		{name: "empty file", content: "", write: true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(dir, c.name+".json")
			if c.write {
				if err := os.WriteFile(path, []byte(c.content), 0o600); err != nil {
					t.Fatalf("write: %v", err)
				}
			}
			if _, ok := readCache(path, time.Now()); ok {
				t.Error("unusable cache reported as hit")
			}
		})
	}
}

func TestWriteCacheCreatesDirectory(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "edgecenter", "version-check.json")
	now := time.Now()

	writeCache(path, latestRelease, now)

	if _, ok := readCache(path, now); !ok {
		t.Fatal("cache not written into a new directory")
	}
}

func TestCacheFileLocation(t *testing.T) {
	t.Parallel()
	path, err := cacheFile()
	if err != nil {
		t.Skipf("user cache dir unavailable: %v", err)
	}

	dir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("user cache dir: %v", err)
	}
	want := filepath.Join(dir, "edgecenter", "version-check.json")
	if path != want {
		t.Errorf("cache file = %q, want %q", path, want)
	}
}

func TestWriteCacheLeavesNoTempFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "version-check.json")

	writeCache(path, latestRelease, time.Now())

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "version-check.json" {
		t.Errorf("directory contains %d entries, want only the cache file", len(entries))
	}
}
