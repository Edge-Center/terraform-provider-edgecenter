package edgecenter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/internal/versioncheck"
)

func TestProviderConfigureUserAgent(t *testing.T) {
	cases := []struct {
		name             string
		terraformVersion string
		providerVersion  string
		want             string
	}{
		{
			name:             "release build",
			terraformVersion: "1.9.8",
			providerVersion:  "0.14.7",
			want:             "terraform/1.9.8 terraform-provider-edgecenter/0.14.7",
		},
		{
			name:             "local build",
			terraformVersion: "0.12+compatible",
			providerVersion:  "dev",
			want:             "terraform/0.12+compatible terraform-provider-edgecenter/dev",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(versioncheck.EnvDisable, "1")
			d := schema.TestResourceDataRaw(t, ProviderSchema(), map[string]interface{}{
				ProviderOptPermanentToken:    "token",
				ProviderOptSingleAPIEndpoint: "https://api.example.com",
			})

			config, diags := ProviderConfigure(context.Background(), d, c.terraformVersion, c.providerVersion)
			if diags.HasError() {
				t.Fatalf("configure: %v", diags)
			}
			if config.UserAgent != c.want {
				t.Errorf("user agent = %q, want %q", config.UserAgent, c.want)
			}
		})
	}
}

func TestProviderConfigureStaysQuietWhenCheckDisabled(t *testing.T) {
	t.Setenv(versioncheck.EnvDisable, "1")
	d := schema.TestResourceDataRaw(t, ProviderSchema(), map[string]interface{}{
		ProviderOptPermanentToken:    "token",
		ProviderOptSingleAPIEndpoint: "https://api.example.com",
	})

	_, diags := ProviderConfigure(context.Background(), d, "1.9.8", "0.1.0")
	if len(diags) != 0 {
		t.Errorf("diagnostics = %v, want none", diags)
	}
}

func TestProviderConfigureWebsecClient(t *testing.T) {
	cases := []struct {
		name      string
		websecAPI string
		wantPath  string
	}{
		{
			name:     "default endpoint under api_endpoint",
			wantPath: "/v2/protection/resources/1",
		},
		{
			name:      "endpoint set explicitly",
			websecAPI: "/websec",
			wantPath:  "/websec/resources/1",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(versioncheck.EnvDisable, "1")
			t.Setenv("EC_WEBSEC_API", "")

			var gotPath, gotAuthorization string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotAuthorization = r.Header.Get("Authorization")
				_, _ = w.Write([]byte(`{"id": 1}`))
			}))
			defer server.Close()

			raw := map[string]interface{}{
				ProviderOptPermanentToken:    "token",
				ProviderOptSingleAPIEndpoint: server.URL,
			}
			if c.websecAPI != "" {
				raw["edgecenter_websec_api"] = server.URL + c.websecAPI
			}
			d := schema.TestResourceDataRaw(t, ProviderSchema(), raw)

			config, diags := ProviderConfigure(context.Background(), d, "1.9.8", "0.1.0")
			if diags.HasError() {
				t.Fatalf("configure: %v", diags)
			}
			if _, err := config.WebsecClient.Resources().Get(context.Background(), 1); err != nil {
				t.Fatalf("get resource: %v", err)
			}
			if gotPath != c.wantPath {
				t.Errorf("path = %q, want %q", gotPath, c.wantPath)
			}
			if gotAuthorization != "APIKey token" {
				t.Errorf("authorization = %q, want %q", gotAuthorization, "APIKey token")
			}
		})
	}
}
