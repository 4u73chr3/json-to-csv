package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type stubFetcher struct {
	data  []byte
	err   error
	value string
}

func (s *stubFetcher) Fetch(_ context.Context, value string) ([]byte, error) {
	s.value = value
	return s.data, s.err
}

func newTestHandler(t *testing.T, fetcher Fetcher) *Handler {
	t.Helper()
	handler, err := New(fetcher)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func formRequest(method, target string, form url.Values) *http.Request {
	const token = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	form.Set("csrf_token", token)
	request := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	return request
}

func TestExportDownloadsCSV(t *testing.T) {
	fetcher := &stubFetcher{data: []byte(`[{"id":1,"name":"test"}]`)}
	handler := newTestHandler(t, fetcher)
	form := url.Values{"parameter": {" test "}}
	request := formRequest(http.MethodPost, "/export", form)
	response := httptest.NewRecorder()

	handler.Export(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if fetcher.value != "test" {
		t.Fatalf("parameter = %q", fetcher.value)
	}
	if !strings.HasPrefix(response.Header().Get("Content-Disposition"), "attachment; filename=") {
		t.Fatalf("Content-Disposition = %q", response.Header().Get("Content-Disposition"))
	}
	if response.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Fatalf("Content-Type = %q", response.Header().Get("Content-Type"))
	}
}

func TestExportValidatesParameter(t *testing.T) {
	handler := newTestHandler(t, &stubFetcher{})
	form := url.Values{"parameter": {""}}
	request := formRequest(http.MethodPost, "/export", form)
	response := httptest.NewRecorder()

	handler.Export(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestExportReportsTimeout(t *testing.T) {
	handler := newTestHandler(t, &stubFetcher{err: context.DeadlineExceeded})
	form := url.Values{"parameter": {"test"}}
	request := formRequest(http.MethodPost, "/export", form)
	response := httptest.NewRecorder()

	handler.Export(response, request)

	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestExportReportsInvalidJSON(t *testing.T) {
	handler := newTestHandler(t, &stubFetcher{data: []byte(`{invalid`)})
	form := url.Values{"parameter": {"test"}}
	request := formRequest(http.MethodPost, "/export", form)
	response := httptest.NewRecorder()

	handler.Export(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestExportReportsAPIError(t *testing.T) {
	handler := newTestHandler(t, &stubFetcher{err: errors.New("upstream unavailable")})
	form := url.Values{"parameter": {"test"}}
	request := formRequest(http.MethodPost, "/export", form)
	response := httptest.NewRecorder()

	handler.Export(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestExportRejectsCrossSiteRequest(t *testing.T) {
	handler := newTestHandler(t, &stubFetcher{data: []byte(`[{"id":1}]`)})
	form := url.Values{"parameter": {"test"}}
	request := httptest.NewRequest(http.MethodPost, "http://example.com/export", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()

	handler.Export(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestExportRejectsMissingCSRFToken(t *testing.T) {
	handler := newTestHandler(t, &stubFetcher{data: []byte(`[{"id":1}]`)})
	form := url.Values{"parameter": {"test"}}
	request := httptest.NewRequest(http.MethodPost, "/export", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.Export(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestIndexSetsCSRFToken(t *testing.T) {
	handler := newTestHandler(t, &stubFetcher{})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	handler.Index(response, request)

	if len(response.Result().Cookies()) == 0 {
		t.Fatal("CSRF cookie is missing")
	}
	if !strings.Contains(response.Body.String(), `name="csrf_token"`) {
		t.Fatal("CSRF form field is missing")
	}
}

func TestExportRejectsUnsupportedContentType(t *testing.T) {
	handler := newTestHandler(t, &stubFetcher{})
	request := httptest.NewRequest(http.MethodPost, "/export", strings.NewReader(`{"parameter":"test"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.Export(response, request)

	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	for _, header := range []string{"Content-Security-Policy", "Permissions-Policy", "X-Content-Type-Options"} {
		if response.Header().Get(header) == "" {
			t.Fatalf("%s is missing", header)
		}
	}
}
