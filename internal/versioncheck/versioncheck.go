package versioncheck

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/go-version"
)

const (
	EnvDisable = "EC_DISABLE_VERSION_CHECK"

	envCheckpointDisable = "CHECKPOINT_DISABLE"
	envInAutomation      = "TF_IN_AUTOMATION"

	ReleasesURL = "https://github.com/Edge-Center/terraform-provider-edgecenter/releases"

	cacheTTL      = 24 * time.Hour
	sourceTimeout = 2 * time.Second
	totalTimeout  = 6 * time.Second
	maxBodySize   = 1 << 20
)

type checker struct {
	client    *http.Client
	sources   []source
	cacheFile string
	now       func() time.Time
}

var (
	once   sync.Once
	latest string
)

func Available(ctx context.Context, installed string) (string, bool) {
	c, err := defaultChecker()
	if err != nil {
		return "", false
	}

	return c.available(ctx, installed)
}

func defaultChecker() (*checker, error) {
	file, err := cacheFile()
	if err != nil {
		return nil, err
	}

	return &checker{
		client:    newClient(),
		sources:   defaultSources(),
		cacheFile: file,
		now:       time.Now,
	}, nil
}

func newClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (c *checker) available(ctx context.Context, installed string) (string, bool) {
	current, err := version.NewSemver(installed)
	if err != nil {
		return "", false
	}

	if disabled() {
		return "", false
	}

	once.Do(func() {
		latest = c.resolve(ctx)
	})

	return newer(latest, current)
}

func newer(candidate string, current *version.Version) (string, bool) {
	if candidate == "" {
		return "", false
	}

	parsed, err := version.NewSemver(candidate)
	if err != nil {
		return "", false
	}

	if parsed.Prerelease() != "" {
		return "", false
	}

	if !parsed.GreaterThan(current) {
		return "", false
	}

	return candidate, true
}

func (c *checker) resolve(ctx context.Context) string {
	if cached, ok := readCache(c.cacheFile, c.now()); ok {
		return cached
	}

	found := c.fetch(ctx)
	if ctx.Err() == nil {
		writeCache(c.cacheFile, found, c.now())
	}

	return found
}

func (c *checker) fetch(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, totalTimeout)
	defer cancel()

	for _, src := range c.sources {
		found, err := c.fetchOne(ctx, src)
		if err != nil {
			continue
		}

		return found
	}

	return ""
}

func (c *checker) fetchOne(ctx context.Context, src source) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, sourceTimeout)
	defer cancel()

	found, err := src.fetch(ctx, c.client, src.url)
	if err != nil {
		return "", err
	}

	if _, err := version.NewSemver(found); err != nil {
		return "", fmt.Errorf("%s: %w", src.name, err)
	}

	return found, nil
}

func disabled() bool {
	if truthy(os.Getenv(EnvDisable)) {
		return true
	}

	for _, key := range []string{envCheckpointDisable, envInAutomation} {
		if os.Getenv(key) != "" {
			return true
		}
	}

	return false
}

func truthy(value string) bool {
	if value == "" {
		return false
	}

	if parsed, err := strconv.ParseBool(value); err == nil {
		return parsed
	}

	return true
}
