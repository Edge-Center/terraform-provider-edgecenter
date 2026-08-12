package versioncheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/hashicorp/go-version"
)

const (
	githubLatestURL   = "https://github.com/Edge-Center/terraform-provider-edgecenter/releases/latest"
	registryURL       = "https://registry.terraform.io/v1/providers/Edge-Center/edgecenter"
	yandexMirrorIndex = "https://terraform-mirror.yandexcloud.net/registry.terraform.io/Edge-Center/edgecenter/index.json"
)

type fetchFunc func(ctx context.Context, client *http.Client, url string) (string, error)

type source struct {
	name  string
	url   string
	fetch fetchFunc
}

func defaultSources() []source {
	return []source{
		{name: "github", url: githubLatestURL, fetch: fetchGitHubRelease},
		{name: "registry", url: registryURL, fetch: fetchRegistry},
		{name: "mirror", url: yandexMirrorIndex, fetch: fetchMirrorIndex},
	}
}

func fetchGitHubRelease(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", fmt.Errorf("github: build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github: request failed: %w", err)
	}
	defer closeBody(resp)

	if resp.StatusCode < http.StatusMultipleChoices || resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("github: unexpected status %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return "", errors.New("github: redirect without location")
	}

	return trimTagPrefix(path.Base(location)), nil
}

func fetchRegistry(ctx context.Context, client *http.Client, url string) (string, error) {
	var payload struct {
		Version string `json:"version"`
	}

	if err := getJSON(ctx, client, url, &payload); err != nil {
		return "", err
	}

	if payload.Version == "" {
		return "", errors.New("registry: empty version")
	}

	return trimTagPrefix(payload.Version), nil
}

func fetchMirrorIndex(ctx context.Context, client *http.Client, url string) (string, error) {
	var payload struct {
		Versions map[string]json.RawMessage `json:"versions"`
	}

	if err := getJSON(ctx, client, url, &payload); err != nil {
		return "", err
	}

	return highest(payload.Versions)
}

func highest(versions map[string]json.RawMessage) (string, error) {
	var top *version.Version

	for raw := range versions {
		parsed, err := version.NewSemver(trimTagPrefix(raw))
		if err != nil || parsed.Prerelease() != "" {
			continue
		}

		if top == nil || parsed.GreaterThan(top) {
			top = parsed
		}
	}

	if top == nil {
		return "", errors.New("mirror: no parsable versions")
	}

	return top.Original(), nil
}

func getJSON(ctx context.Context, client *http.Client, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodySize)).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func trimTagPrefix(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}

func closeBody(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodySize))
	_ = resp.Body.Close()
}
