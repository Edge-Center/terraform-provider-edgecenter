package versioncheck

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-version"
)

const latestRelease = "0.14.7"

func testChecker(t *testing.T, sources []source) *checker {
	t.Helper()

	return &checker{
		client:    newClient(),
		sources:   sources,
		cacheFile: filepath.Join(t.TempDir(), "version-check.json"),
		now:       time.Now,
	}
}

func stubSource(name, value string, err error, calls *int32) source {
	return source{
		name: name,
		url:  "http://stub.invalid",
		fetch: func(context.Context, *http.Client, string) (string, error) {
			atomic.AddInt32(calls, 1)

			return value, err
		},
	}
}

func TestFetchWalksSourcesUntilSuccess(t *testing.T) {
	t.Parallel()
	var first, second, third int32
	c := testChecker(t, []source{
		stubSource("first", "", errors.New("unreachable"), &first),
		stubSource("second", latestRelease, nil, &second),
		stubSource("third", "0.1.0", nil, &third),
	})

	if got := c.fetch(context.Background()); got != latestRelease {
		t.Errorf("fetch = %q, want %q", got, latestRelease)
	}
	if first != 1 || second != 1 {
		t.Errorf("calls: first = %d, second = %d, want 1 and 1", first, second)
	}
	if third != 0 {
		t.Errorf("third source called %d times, want 0", third)
	}
}

func TestFetchSkipsSourceWithUnparsableVersion(t *testing.T) {
	t.Parallel()
	var first, second int32
	c := testChecker(t, []source{
		stubSource("first", "releases", nil, &first),
		stubSource("second", latestRelease, nil, &second),
	})

	if got := c.fetch(context.Background()); got != latestRelease {
		t.Errorf("fetch = %q, want %q", got, latestRelease)
	}
	if second != 1 {
		t.Errorf("second source called %d times, want 1", second)
	}
}

func TestFetchReturnsEmptyWhenEverySourceFails(t *testing.T) {
	t.Parallel()
	var calls int32
	c := testChecker(t, []source{
		stubSource("first", "", errors.New("timeout"), &calls),
		stubSource("second", "", errors.New("forbidden"), &calls),
	})

	if got := c.fetch(context.Background()); got != "" {
		t.Errorf("fetch = %q, want empty", got)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestResolveUsesCacheWithoutNetwork(t *testing.T) {
	t.Parallel()
	var calls int32
	c := testChecker(t, []source{stubSource("only", latestRelease, nil, &calls)})

	if got := c.resolve(context.Background()); got != latestRelease {
		t.Fatalf("first resolve = %q, want %q", got, latestRelease)
	}
	if got := c.resolve(context.Background()); got != latestRelease {
		t.Fatalf("second resolve = %q, want %q", got, latestRelease)
	}
	if calls != 1 {
		t.Errorf("source called %d times, want 1", calls)
	}
}

func TestResolveDoesNotRetryFailureWithinTTL(t *testing.T) {
	t.Parallel()
	var calls int32
	c := testChecker(t, []source{stubSource("only", "", errors.New("unreachable"), &calls)})

	if got := c.resolve(context.Background()); got != "" {
		t.Fatalf("first resolve = %q, want empty", got)
	}
	if got := c.resolve(context.Background()); got != "" {
		t.Fatalf("second resolve = %q, want empty", got)
	}
	if calls != 1 {
		t.Errorf("source called %d times, want 1", calls)
	}
}

func TestResolveRefetchesAfterTTL(t *testing.T) {
	t.Parallel()
	var calls int32
	c := testChecker(t, []source{stubSource("only", latestRelease, nil, &calls)})
	base := time.Now()
	c.now = func() time.Time { return base }

	c.resolve(context.Background())

	c.now = func() time.Time { return base.Add(cacheTTL + time.Minute) }
	c.resolve(context.Background())

	if calls != 2 {
		t.Errorf("source called %d times, want 2", calls)
	}
}

func TestResolveKeepsCacheUntouchedWhenCallerCancels(t *testing.T) {
	t.Parallel()
	var calls int32
	c := testChecker(t, []source{stubSource("only", latestRelease, nil, &calls)})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c.resolve(ctx)

	if _, ok := readCache(c.cacheFile, time.Now()); ok {
		t.Error("cancelled lookup must not persist a result")
	}
}

func TestNewer(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		installed string
		candidate string
		want      string
		wantOK    bool
	}{
		{name: "update available", installed: "0.13.8", candidate: latestRelease, want: latestRelease, wantOK: true},
		{name: "same version", installed: latestRelease, candidate: latestRelease},
		{name: "installed is ahead", installed: latestRelease, candidate: "0.13.8"},
		{name: "patch update", installed: "1.0.0", candidate: "1.0.1", want: "1.0.1", wantOK: true},
		{name: "minor over lexical", installed: "0.9.9", candidate: latestRelease, want: latestRelease, wantOK: true},
		{name: "empty candidate", installed: "0.13.8", candidate: ""},
		{name: "unparsable candidate", installed: "0.13.8", candidate: "nightly"},
		{name: "prerelease is not offered", installed: "0.14.0", candidate: "0.15.0-rc1"},
		{name: "prerelease build metadata", installed: "0.14.0", candidate: "0.15.0-beta+exp"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			current, err := version.NewSemver(c.installed)
			if err != nil {
				t.Fatalf("installed %q: %v", c.installed, err)
			}

			got, ok := newer(c.candidate, current)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if got != c.want {
				t.Errorf("version = %q, want %q", got, c.want)
			}
		})
	}
}

func TestTruthy(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"":      false,
		"0":     false,
		"false": false,
		"FALSE": false,
		"f":     false,
		"1":     true,
		"true":  true,
		"TRUE":  true,
		"yes":   true,
		"on":    true,
	}
	for in, want := range cases {
		if got := truthy(in); got != want {
			t.Errorf("truthy(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDisabledByEnv(t *testing.T) {
	cases := []struct {
		key   string
		value string
		want  bool
	}{
		{key: EnvDisable, value: "1", want: true},
		{key: EnvDisable, value: "0", want: false},
		{key: EnvDisable, value: "false", want: false},
		{key: envCheckpointDisable, value: "1", want: true},
		{key: envCheckpointDisable, value: "0", want: true},
		{key: envCheckpointDisable, value: "", want: false},
		{key: envInAutomation, value: "true", want: true},
		{key: envInAutomation, value: "", want: false},
	}
	for _, c := range cases {
		t.Run(c.key+"="+c.value, func(t *testing.T) {
			t.Setenv(c.key, c.value)
			if got := disabled(); got != c.want {
				t.Errorf("disabled = %v, want %v", got, c.want)
			}
		})
	}
}

func TestAvailableSkipsUnreleasedBuild(t *testing.T) {
	t.Parallel()
	for _, installed := range []string{"dev", "", "unknown", "0.1.0-dev+dirty!"} {
		installed := installed
		t.Run(installed, func(t *testing.T) {
			t.Parallel()
			var calls int32
			c := testChecker(t, []source{stubSource("only", "9.9.9", nil, &calls)})

			if _, ok := c.available(context.Background(), installed); ok {
				t.Errorf("available(%q) reported an update", installed)
			}
			if calls != 0 {
				t.Errorf("source called %d times, want 0", calls)
			}
		})
	}
}

func TestAvailableRespectsDisableSwitch(t *testing.T) {
	for _, key := range []string{EnvDisable, envCheckpointDisable, envInAutomation} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "1")
			var calls int32
			c := testChecker(t, []source{stubSource("only", "9.9.9", nil, &calls)})

			if _, ok := c.available(context.Background(), "0.1.0"); ok {
				t.Error("available reported an update while disabled")
			}
			if calls != 0 {
				t.Errorf("source called %d times, want 0", calls)
			}
		})
	}
}
