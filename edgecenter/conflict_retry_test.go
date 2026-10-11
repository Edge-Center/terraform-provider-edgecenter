package edgecenter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func setFastConflictBackoff(t *testing.T) {
	t.Helper()

	original := transientConflictBackoff
	transientConflictBackoff = make([]time.Duration, transientConflictRetryMax)
	for i := range transientConflictBackoff {
		transientConflictBackoff[i] = time.Millisecond
	}

	t.Cleanup(func() {
		transientConflictBackoff = original
	})
}

func TestConflictRetryTransportRetriesLockedResource(t *testing.T) {
	setFastConflictBackoff(t)

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) <= 2 {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"message": "Resource 447d2959-8ae0-4ca0-8d47-9f050a3637d7 is locked"}`))

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer srv.Close()

	client := &http.Client{Transport: newConflictRetryTransport(http.DefaultTransport)}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Fatalf("expected 3 requests, got %d", got)
	}
}

func TestConflictRetryTransportRetriesPendingUpdate(t *testing.T) {
	setFastConflictBackoff(t)

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte("Invalid state PENDING_UPDATE of loadbalancer resource 447d2959"))

			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: newConflictRetryTransport(http.DefaultTransport)}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected 2 requests, got %d", got)
	}
}

func TestConflictRetryTransportNoRetryOnOtherConflict(t *testing.T) {
	setFastConflictBackoff(t)

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message": "Quota exceeded for loadbalancers"}`))
	}))
	defer srv.Close()

	client := &http.Client{Transport: newConflictRetryTransport(http.DefaultTransport)}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("cannot read body: %v", err)
	}

	if !bytes.Contains(body, []byte("Quota exceeded")) {
		t.Fatalf("unexpected body: %s", body)
	}

	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 1 request, got %d", got)
	}
}

func TestConflictRetryTransportExhaustsRetries(t *testing.T) {
	setFastConflictBackoff(t)

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte("Resource 447d2959 is locked"))
	}))
	defer srv.Close()

	client := &http.Client{Transport: newConflictRetryTransport(http.DefaultTransport)}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("cannot read body: %v", err)
	}

	if !bytes.Contains(body, []byte("is locked")) {
		t.Fatalf("unexpected body: %s", body)
	}

	if got := atomic.LoadInt32(&hits); got != transientConflictRetryMax+1 {
		t.Fatalf("expected %d requests, got %d", transientConflictRetryMax+1, got)
	}
}

func TestConflictRetryTransportReplaysRequestBody(t *testing.T) {
	setFastConflictBackoff(t)

	var hits int32
	var lastBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		lastBody = string(body)
		if atomic.AddInt32(&hits, 1) <= 2 {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte("Resource port-1 is locked"))

			return
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "lb-1"}`))
	}))
	defer srv.Close()

	client := &http.Client{Transport: newConflictRetryTransport(http.DefaultTransport)}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		srv.URL,
		bytes.NewBufferString(`{"name": "test-lb"}`),
	)
	if err != nil {
		t.Fatalf("cannot create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Fatalf("expected 3 requests, got %d", got)
	}

	if lastBody != `{"name": "test-lb"}` {
		t.Fatalf("request body was not replayed, got %q", lastBody)
	}
}

func TestConflictRetryTransportStopsOnContextCancel(t *testing.T) {
	original := transientConflictBackoff
	transientConflictBackoff = []time.Duration{
		30 * time.Second,
		30 * time.Second,
		30 * time.Second,
		30 * time.Second,
		30 * time.Second,
	}
	t.Cleanup(func() {
		transientConflictBackoff = original
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte("Resource 447d2959 is locked"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	client := &http.Client{Transport: newConflictRetryTransport(http.DefaultTransport)}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("cannot create request: %v", err)
	}

	done := make(chan struct{})
	var resp *http.Response
	var reqErr error
	go func() {
		defer close(done)
		resp, reqErr = client.Do(req)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not return after context cancel")
	}

	if reqErr != nil {
		t.Fatalf("unexpected error: %v", reqErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestNewCloudClientWrapsTransportWithConflictRetry(t *testing.T) {
	cfg := &Config{
		PermanentToken: "token",
		CloudBaseURL:   "https://api.example.com/cloud/",
		UserAgent:      "test",
	}

	cloudClient, err := cfg.NewCloudClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := cloudClient.HTTPClient.Transport.(*conflictRetryTransport); !ok {
		t.Fatalf("expected HTTP client transport to be wrapped with conflictRetryTransport, got %T",
			cloudClient.HTTPClient.Transport)
	}
}
