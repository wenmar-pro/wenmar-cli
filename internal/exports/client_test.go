package exports

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSchema_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/exports/schema.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("auth header = %q", got)
		}
		if got := r.Header.Get("X-Wenmar-Location"); got != "42" {
			t.Errorf("location header = %q", got)
		}
		_ = json.NewEncoder(w).Encode(SchemaResponse{
			Resources: []SchemaResource{{Name: "customers", Formats: []string{"csv"}}},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "42", srv.Client())
	got, err := c.Schema(context.Background())
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if len(got.Resources) != 1 || got.Resources[0].Name != "customers" {
		t.Fatalf("unexpected schema: %+v", got)
	}
}

func TestSchema_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "forbidden", "message": "nope"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "", srv.Client())
	if _, err := c.Schema(context.Background()); err == nil {
		t.Fatal("expected error on 403")
	}
}

func TestCreate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/exports.json" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q", ct)
		}
		var req CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if req.Resource != "customers" || req.Format != "csv" {
			t.Errorf("unexpected body: %+v", req)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateResponse{Status: "complete", DownloadURL: "/exports/1/download"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "", srv.Client())
	got, err := c.Create(context.Background(), CreateRequest{Resource: "customers", Format: "csv"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Status != "complete" || got.DownloadURL != "/exports/1/download" {
		t.Fatalf("unexpected create response: %+v", got)
	}
}

func TestCreate_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "validation_error", "message": "bad format"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "", srv.Client())
	if _, err := c.Create(context.Background(), CreateRequest{Resource: "x", Format: "y"}); err == nil {
		t.Fatal("expected error on 422")
	}
}

func TestDownload_PollsUntilReady(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(DownloadPendingResponse{Status: "pending", RetryAfter: 1})
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="export.csv"`)
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("a,b\n1,2\n"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "", srv.Client())
	data, filename, contentType, err := c.Download(context.Background(), "/exports/1/download", 10*time.Second)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 requests (202 then 200), got %d", calls)
	}
	if string(data) != "a,b\n1,2\n" {
		t.Errorf("unexpected data: %q", data)
	}
	if filename != "export.csv" {
		t.Errorf("filename = %q", filename)
	}
	if contentType != "text/csv" {
		t.Errorf("contentType = %q", contentType)
	}
}

func TestDownload_GoneIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "expired", "message": "export expired"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "", srv.Client())
	_, _, _, err := c.Download(context.Background(), "/exports/1/download", time.Second)
	if err == nil {
		t.Fatal("expected error on 410")
	}
}

func TestDownload_TimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(DownloadPendingResponse{Status: "pending", RetryAfter: 1})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "", srv.Client())
	_, _, _, err := c.Download(context.Background(), "/exports/1/download", time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestDownload_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(DownloadPendingResponse{Status: "pending", RetryAfter: 5})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	c := NewClient(srv.URL, "tok", "", srv.Client())
	_, _, _, err := c.Download(ctx, "/exports/1/download", time.Minute)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestDownload_AbsoluteURL(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient("http://unused.example", "tok", "", srv.Client())
	_, _, _, err := c.Download(context.Background(), srv.URL+"/blob", time.Second)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if gotPath != "/blob" {
		t.Errorf("absolute URL not honored, server saw %q", gotPath)
	}
}

func TestDownloadInline(t *testing.T) {
	payload := []byte("hello")
	resp := &CreateResponse{Data: base64.StdEncoding.EncodeToString(payload)}
	got, err := DownloadInline(resp)
	if err != nil {
		t.Fatalf("DownloadInline: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("decoded = %q", got)
	}
}

func TestDownloadInline_Guards(t *testing.T) {
	if _, err := DownloadInline(nil); err == nil {
		t.Error("expected error for nil response")
	}
	if _, err := DownloadInline(&CreateResponse{}); err == nil {
		t.Error("expected error for empty data")
	}
}

func TestFilenameFromHeader(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"quoted", `attachment; filename="export.csv"`, "export.csv"},
		{"unquoted", `attachment; filename=export.csv`, "export.csv"},
		{"utf8", "attachment; filename*=UTF-8''caf%C3%A9.csv", "café.csv"},
		{"missing", `attachment`, ""},
		{"empty", ``, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filenameFromHeader(tt.in); got != tt.want {
				t.Errorf("filenameFromHeader(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveURL(t *testing.T) {
	c := NewClient("https://api.example/", "tok", "", nil)
	if got := c.resolveURL("/exports/1/download"); got != "https://api.example/exports/1/download" {
		t.Errorf("relative resolve = %q", got)
	}
	if got := c.resolveURL("http://other.example/x"); got != "http://other.example/x" {
		t.Errorf("absolute resolve = %q", got)
	}
}

func TestNewClient_DefaultHTTPClient(t *testing.T) {
	c := NewClient("https://api.example/", "tok", "", nil)
	if c.HTTPClient == nil {
		t.Fatal("expected a default http client")
	}
	if c.BaseURL != "https://api.example" {
		t.Errorf("trailing slash not trimmed: %q", c.BaseURL)
	}
}
