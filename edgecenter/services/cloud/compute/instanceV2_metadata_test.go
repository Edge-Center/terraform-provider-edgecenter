package compute

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func newMetadataTestClient(t *testing.T, baseURL string, projectID, regionID int) *edgecloudV2.Client {
	t.Helper()

	client, err := edgecloudV2.New(nil,
		edgecloudV2.SetAPIKey("token"),
		edgecloudV2.SetBaseURL(baseURL),
	)
	if err != nil {
		t.Fatalf("create cloud client: %v", err)
	}
	client.Project = projectID
	client.Region = regionID

	return client
}

func TestFilterManagedMetadataFiltersUnmanagedAndRetainsManagedReadOnly(t *testing.T) {
	t.Parallel()

	mds := []edgecloudV2.MetadataDetailed{
		{Key: "managed", Value: "new-value", ReadOnly: false},
		{Key: "unmanaged", Value: "x", ReadOnly: false},
		{Key: "readonly-managed", Value: "y", ReadOnly: true},
		{Key: "readonly-unmanaged", Value: "z", ReadOnly: true},
	}

	managedKeys := map[string]struct{}{
		"managed":          {},
		"readonly-managed": {},
		"deleted":          {},
	}

	got := filterManagedMetadata(mds, managedKeys)

	if len(got) != 2 {
		t.Fatalf("expected 2 managed metadata entries, got %d: %#v", len(got), got)
	}
	if got["managed"] != "new-value" {
		t.Fatalf("expected managed value to be updated, got %#v", got["managed"])
	}
	if _, ok := got["unmanaged"]; ok {
		t.Fatalf("unmanaged metadata must not be added to state")
	}
	if got["readonly-managed"] != "y" {
		t.Fatalf("managed read-only metadata must be retained in state, got %#v", got["readonly-managed"])
	}
	if _, ok := got["readonly-unmanaged"]; ok {
		t.Fatalf("unmanaged read-only metadata must not be added to state")
	}
	if _, ok := got["deleted"]; ok {
		t.Fatalf("deleted managed metadata must be removed from state")
	}
}

func TestFilterManagedMetadataDrift(t *testing.T) {
	t.Parallel()

	mds := []edgecloudV2.MetadataDetailed{
		{Key: "managed", Value: "drifted", ReadOnly: false},
	}

	got := filterManagedMetadata(mds, map[string]struct{}{"managed": {}})
	if got["managed"] != "drifted" {
		t.Fatalf("expected drifted value, got %#v", got["managed"])
	}

	got = filterManagedMetadata(nil, map[string]struct{}{"managed": {}})
	if len(got) != 0 {
		t.Fatalf("expected empty state after deletion, got %#v", got)
	}
}

func TestFilterWritableMetadataForImport(t *testing.T) {
	t.Parallel()

	got := filterWritableMetadata([]edgecloudV2.MetadataDetailed{
		{Key: "configured", Value: "value", ReadOnly: false},
		{Key: "service", Value: "internal", ReadOnly: true},
	})

	if got["configured"] != "value" {
		t.Fatalf("expected writable metadata in imported state, got %#v", got["configured"])
	}
	if _, ok := got["service"]; ok {
		t.Fatalf("read-only metadata must not be added during import")
	}
}

func TestReadManagedMetadataSkipsListWithoutManagedKeys(t *testing.T) {
	t.Parallel()

	client := newMetadataTestClient(t, "http://127.0.0.1:1", 352183, 1)
	got, err := readManagedMetadata(t.Context(), client, "instance-id", map[string]string{})
	if err != nil {
		t.Fatalf("readManagedMetadata returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty metadata, got %#v", got)
	}
}

func TestReadManagedMetadataUsesSingleListNoGetItem(t *testing.T) {
	t.Parallel()

	var listCalls atomic.Int32
	var itemCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/instances/352183/1/00000000-0000-0000-0000-000000000001/metadata" {
			listCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []map[string]interface{}{
					{"key": "managed", "value": "from-api", "read_only": false},
					{"key": "unmanaged", "value": "x", "read_only": false},
					{"key": "readonly", "value": "y", "read_only": true},
				},
			})
			return
		}
		if r.URL.Path == "/v2/instances/352183/1/00000000-0000-0000-0000-000000000001/metadata_item" {
			itemCalls.Add(1)
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := newMetadataTestClient(t, server.URL, 352183, 1)
	got, err := readManagedMetadata(t.Context(), client, "00000000-0000-0000-0000-000000000001", map[string]string{"managed": "old", "deleted": "old"})
	if err != nil {
		t.Fatalf("readManagedMetadata returned error: %v", err)
	}

	if listCalls.Load() != 1 {
		t.Fatalf("expected exactly one MetadataList request, got %d", listCalls.Load())
	}
	if itemCalls.Load() != 0 {
		t.Fatalf("expected no MetadataGetItem requests, got %d", itemCalls.Load())
	}

	if got["managed"] != "from-api" {
		t.Fatalf("expected managed value from API, got %#v", got["managed"])
	}
	if _, ok := got["deleted"]; ok {
		t.Fatalf("deleted managed metadata must be removed")
	}
	if _, ok := got["unmanaged"]; ok {
		t.Fatalf("unmanaged metadata must not be added")
	}
	if _, ok := got["readonly"]; ok {
		t.Fatalf("read-only metadata must not be added")
	}
}
