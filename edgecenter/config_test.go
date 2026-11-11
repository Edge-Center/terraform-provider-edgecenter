package edgecenter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCloudClientConfiguresTimeoutAndRetries(t *testing.T) {
	t.Parallel()

	config := Config{
		PermanentToken: "token",
		CloudBaseURL:   "https://api.example.test/cloud",
		UserAgent:      "test",
	}

	client, err := config.NewCloudClient()
	if err != nil {
		t.Fatalf("create cloud client: %v", err)
	}

	if client.HTTPClient.Timeout != cloudHTTPTimeout {
		t.Fatalf("HTTP client timeout = %s, want %s", client.HTTPClient.Timeout, cloudHTTPTimeout)
	}
	if client.RetryConfig.RetryMax != cloudRetryMax {
		t.Fatalf("retry max = %d, want %d", client.RetryConfig.RetryMax, cloudRetryMax)
	}
	if client.RetryConfig.RetryWaitMin == nil || *client.RetryConfig.RetryWaitMin != cloudRetryWaitMin {
		t.Fatalf("retry wait min = %v, want %v", client.RetryConfig.RetryWaitMin, cloudRetryWaitMin)
	}
	if client.RetryConfig.RetryWaitMax == nil || *client.RetryConfig.RetryWaitMax != cloudRetryWaitMax {
		t.Fatalf("retry wait max = %v, want %v", client.RetryConfig.RetryWaitMax, cloudRetryWaitMax)
	}
	if client.RetryConfig.Logger == nil {
		t.Fatal("retry logger is nil")
	}
}

func TestNewCloudClientBoundsWholeRetryCycle(t *testing.T) {
	t.Parallel()

	const timeout = 200 * time.Millisecond

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	config := Config{
		PermanentToken: "token",
		CloudBaseURL:   server.URL,
		UserAgent:      "test",
	}

	client, err := config.newCloudClient(timeout)
	if err != nil {
		t.Fatalf("create cloud client: %v", err)
	}

	start := time.Now()
	_, _, err = client.Projects.Get(context.Background(), "352183")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("request took %s, want no more than %s", elapsed, 2*time.Second)
	}
}
