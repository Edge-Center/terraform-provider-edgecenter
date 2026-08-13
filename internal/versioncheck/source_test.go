package versioncheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchGitHubRelease(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		status   int
		location string
		want     string
		wantErr  bool
	}{
		{name: "tag redirect", status: http.StatusFound, location: "/Edge-Center/terraform-provider-edgecenter/releases/tag/v0.14.7", want: latestRelease},
		{name: "permanent redirect", status: http.StatusMovedPermanently, location: "/releases/tag/1.0.0", want: "1.0.0"},
		{name: "temporary redirect", status: http.StatusTemporaryRedirect, location: "/releases/tag/v1.2.3", want: "1.2.3"},
		{name: "permanent redirect 308", status: http.StatusPermanentRedirect, location: "/releases/tag/v2.0.0", want: "2.0.0"},
		{name: "no location", status: http.StatusFound, wantErr: true},
		{name: "no redirect", status: http.StatusOK, wantErr: true},
		{name: "not found", status: http.StatusNotFound, wantErr: true},
		{name: "no releases yet", status: http.StatusFound, location: "/Edge-Center/terraform-provider-edgecenter/releases", want: "releases"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodHead {
					t.Errorf("method = %s, want HEAD", r.Method)
				}
				if c.location != "" {
					w.Header().Set("Location", c.location)
				}
				w.WriteHeader(c.status)
			}))
			defer srv.Close()

			got, err := fetchGitHubRelease(context.Background(), newClient(), srv.URL)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("version = %q, want %q", got, c.want)
			}
		})
	}
}

func TestFetchRegistry(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr bool
	}{
		{name: "version field", status: http.StatusOK, body: `{"version":"0.14.7","versions":[]}`, want: latestRelease},
		{name: "tag prefix", status: http.StatusOK, body: `{"version":"v0.14.7"}`, want: latestRelease},
		{name: "empty version", status: http.StatusOK, body: `{"version":""}`, wantErr: true},
		{name: "broken json", status: http.StatusOK, body: `{"version":`, wantErr: true},
		{name: "geo blocked", status: http.StatusForbidden, body: "Content not available in your region", wantErr: true},
		{name: "not found", status: http.StatusNotFound, body: "", wantErr: true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(c.status)
				_, _ = w.Write([]byte(c.body))
			}))
			defer srv.Close()

			got, err := fetchRegistry(context.Background(), newClient(), srv.URL)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("version = %q, want %q", got, c.want)
			}
		})
	}
}

func TestFetchMirrorIndex(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		body    string
		want    string
		wantErr bool
	}{
		{name: "highest by semver", body: `{"versions":{"0.9.9":{},"0.14.7":{},"0.13.8":{}}}`, want: latestRelease},
		{name: "single", body: `{"versions":{"1.2.3":{}}}`, want: "1.2.3"},
		{name: "unparsable entries skipped", body: `{"versions":{"nightly":{},"0.2.0":{}}}`, want: "0.2.0"},
		{name: "prerelease not promoted", body: `{"versions":{"0.14.7":{},"0.15.0-rc1":{}}}`, want: latestRelease},
		{name: "only prereleases", body: `{"versions":{"0.15.0-rc1":{}}}`, wantErr: true},
		{name: "empty", body: `{"versions":{}}`, wantErr: true},
		{name: "missing key", body: `{}`, wantErr: true},
		{name: "broken json", body: `{"versions"`, wantErr: true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(c.body))
			}))
			defer srv.Close()

			got, err := fetchMirrorIndex(context.Background(), newClient(), srv.URL)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("version = %q, want %q", got, c.want)
			}
		})
	}
}

func TestHighestPrefersSemverOverLexical(t *testing.T) {
	t.Parallel()
	versions := map[string]json.RawMessage{
		"0.9.9":       json.RawMessage(`{}`),
		latestRelease: json.RawMessage(`{}`),
	}

	got, err := highest(versions)
	if err != nil {
		t.Fatalf("highest: %v", err)
	}
	if got != latestRelease {
		t.Errorf("highest = %q, want %q", got, latestRelease)
	}
}

func TestDefaultSourcesOrder(t *testing.T) {
	t.Parallel()
	sources := defaultSources()

	want := []struct {
		name string
		url  string
	}{
		{name: "github", url: githubLatestURL},
		{name: "registry", url: registryURL},
		{name: "mirror", url: yandexMirrorIndex},
	}
	if len(sources) != len(want) {
		t.Fatalf("sources = %d, want %d", len(sources), len(want))
	}
	for i, w := range want {
		if sources[i].name != w.name {
			t.Errorf("source %d name = %q, want %q", i, sources[i].name, w.name)
		}
		if sources[i].url != w.url {
			t.Errorf("source %d url = %q, want %q", i, sources[i].url, w.url)
		}
		if sources[i].fetch == nil {
			t.Errorf("source %d has no fetch function", i)
		}
	}
}

func TestTrimTagPrefix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{in: "v0.14.7", want: latestRelease},
		{in: latestRelease, want: latestRelease},
		{in: " v1.0.0 ", want: "1.0.0"},
		{in: "\nv2.0.0\t", want: "2.0.0"},
		{in: "", want: ""},
	}
	for _, c := range cases {
		if got := trimTagPrefix(c.in); got != c.want {
			t.Errorf("trimTagPrefix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
