package edgecenter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const (
	projectTestID   = 352183
	projectTestPath = "/v1/projects/352183"
)

func newProjectTestClient(t *testing.T, baseURL string) *edgecloudV2.Client {
	t.Helper()

	client, err := edgecloudV2.New(nil,
		edgecloudV2.SetAPIKey("token"),
		edgecloudV2.SetBaseURL(baseURL),
	)
	if err != nil {
		t.Fatalf("create cloud client: %v", err)
	}

	return client
}

func newProjectTestData(t *testing.T, raw map[string]interface{}) *schema.ResourceData {
	t.Helper()

	return schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"project_id": {
			Type:     schema.TypeInt,
			Optional: true,
		},
		"project_name": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}, raw)
}

func TestGetProjectIDUsesConfiguredIDWithoutCloudRequest(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newProjectTestClient(t, server.URL)
	data := newProjectTestData(t, map[string]interface{}{
		"project_id":   projectTestID,
		"project_name": "ignored",
	})

	projectID, err := GetProjectID(context.Background(), client, data)
	if err != nil {
		t.Fatalf("get project ID: %v", err)
	}
	if projectID != projectTestID {
		t.Fatalf("project ID = %d, want %d", projectID, projectTestID)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("cloud requests = %d, want 0", got)
	}
}

func TestGetProjectIDResolvesNameWithProjectsList(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":2,"results":[{"id":1,"name":"other"},{"id":352183,"name":"target"}]}`))
	}))
	defer server.Close()

	client := newProjectTestClient(t, server.URL)
	data := newProjectTestData(t, map[string]interface{}{
		"project_name": "target",
	})

	projectID, err := GetProjectID(context.Background(), client, data)
	if err != nil {
		t.Fatalf("get project ID: %v", err)
	}
	if projectID != projectTestID {
		t.Fatalf("project ID = %d, want %d", projectID, projectTestID)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 1 || paths[0] != "/v1/projects" {
		t.Fatalf("request paths = %v, want [/v1/projects]", paths)
	}
}

func TestGetProjectByIDV2UsesSingleProjectRequest(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":352183,"name":"target"}`))
	}))
	defer server.Close()

	client := newProjectTestClient(t, server.URL)

	project, err := GetProjectByIDV2(context.Background(), client, projectTestID)
	if err != nil {
		t.Fatalf("get project by ID: %v", err)
	}
	if project.ID != projectTestID {
		t.Fatalf("project ID = %d, want %d", project.ID, projectTestID)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 1 || paths[0] != projectTestPath {
		t.Fatalf("request paths = %v, want [%s]", paths, projectTestPath)
	}
}

func TestGetProjectByIDV2FallsBackToListOnForbidden(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()

		if r.URL.Path == projectTestPath {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":1,"results":[{"id":352183,"name":"target"}]}`))
	}))
	defer server.Close()

	client := newProjectTestClient(t, server.URL)

	project, err := GetProjectByIDV2(context.Background(), client, projectTestID)
	if err != nil {
		t.Fatalf("get project by ID: %v", err)
	}
	if project.ID != projectTestID {
		t.Fatalf("project ID = %d, want %d", project.ID, projectTestID)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 2 || paths[0] != projectTestPath || paths[1] != "/v1/projects" {
		t.Fatalf("request paths = %v, want [%s /v1/projects]", paths, projectTestPath)
	}
}

func TestGetProjectByIDV2DoesNotFallBackToListOnNotFound(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := newProjectTestClient(t, server.URL)

	_, err := GetProjectByIDV2(context.Background(), client, projectTestID)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("cloud requests = %d, want 1", got)
	}
}

func TestGetProjectByIDV2DoesNotFallBackToListOnServerError(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newProjectTestClient(t, server.URL)

	_, err := GetProjectByIDV2(context.Background(), client, projectTestID)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("cloud requests = %d, want 1", got)
	}
}

func TestGetProjectByIDV2ReturnsErrorWhenForbiddenFallbackFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == projectTestPath {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newProjectTestClient(t, server.URL)

	_, err := GetProjectByIDV2(context.Background(), client, projectTestID)
	if err == nil {
		t.Fatal("expected error")
	}
}
