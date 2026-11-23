package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPIClientSendsParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Query().Get("search") != "golang" {
			t.Errorf("search = %q", r.URL.Query().Get("search"))
		}
		_, _ = w.Write([]byte(`[{"id":1}]`))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL+"?limit=10", "search", time.Second)
	data, err := client.Fetch(context.Background(), "golang")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `[{"id":1}]` {
		t.Fatalf("data = %q", data)
	}
}

func TestAPIClientRejectsErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "q", time.Second)
	if _, err := client.Fetch(context.Background(), "test"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestAPIClientDefaultsToMock(t *testing.T) {
	client := NewAPIClient("", "q", time.Second)
	data, err := client.Fetch(context.Background(), "sample")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got == "" {
		t.Fatal("mock response is empty")
	}
}
