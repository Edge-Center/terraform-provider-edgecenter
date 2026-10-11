package edgecenter

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const transientConflictRetryMax = 5

var transientConflictBackoff = []time.Duration{
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
	32 * time.Second,
}

func isTransientConflictBody(body []byte) bool {
	lower := strings.ToLower(string(body))
	return strings.Contains(lower, "is locked") ||
		strings.Contains(lower, "invalid state pending_")
}

type conflictRetryTransport struct {
	base http.RoundTripper
}

func newConflictRetryTransport(base http.RoundTripper) *conflictRetryTransport {
	if base == nil {
		base = http.DefaultTransport
	}

	return &conflictRetryTransport{base: base}
}

func (t *conflictRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	for attempt := 0; attempt < transientConflictRetryMax; attempt++ {
		if resp.StatusCode != http.StatusConflict {
			return resp, nil
		}

		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}

		if !isTransientConflictBody(body) {
			resp.Body = io.NopCloser(bytes.NewReader(body))

			return resp, nil
		}

		log.Printf("[DEBUG] transient 409 conflict on %s %s, retrying (attempt %d/%d)",
			req.Method, req.URL.String(), attempt+1, transientConflictRetryMax)

		select {
		case <-req.Context().Done():
			resp.Body = io.NopCloser(bytes.NewReader(body))

			return resp, nil
		case <-time.After(transientConflictBackoff[attempt]):
		}

		if req.Body != nil {
			if req.GetBody == nil {
				resp.Body = io.NopCloser(bytes.NewReader(body))

				return resp, nil
			}

			replayBody, bodyErr := req.GetBody()
			if bodyErr != nil {
				resp.Body = io.NopCloser(bytes.NewReader(body))

				return resp, bodyErr
			}

			req.Body = replayBody
		}

		resp, err = t.base.RoundTrip(req)
		if err != nil {
			return resp, err
		}
	}

	if resp.StatusCode == http.StatusConflict {
		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}

		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	return resp, nil
}
