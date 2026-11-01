package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthAndReadiness(t *testing.T) {
	for _, tc := range []struct {
		path     string
		readyErr error
		status   int
		value    string
	}{
		{"/healthz", errors.New("secret db password"), 200, "ok"},
		{"/readyz", nil, 200, "ready"},
		{"/readyz", errors.New("secret db password"), 503, "unavailable"},
	} {
		t.Run(tc.path+tc.value, func(t *testing.T) {
			h := New(func(context.Context) error { return tc.readyErr })
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("status %d", w.Code)
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["status"] != tc.value {
				t.Fatalf("body %v", body)
			}
			if strings.Contains(w.Body.String(), "secret") {
				t.Fatal("dependency error leaked")
			}
			if w.Header().Get("X-Request-ID") == "" {
				t.Fatal("missing request id")
			}
			if w.Header().Get("Content-Type") != "application/json" {
				t.Fatal("not JSON")
			}
		})
	}
}

func TestUnknownAndMethodErrorsAreJSON(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		code         int
	}{{"GET", "/missing", 404}, {"POST", "/healthz", 405}} {
		w := httptest.NewRecorder()
		New(nil).ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.code || !json.Valid(w.Body.Bytes()) {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
		if w.Header().Get("X-Request-ID") == "" {
			t.Fatal("missing request ID")
		}
		if tc.code == http.StatusMethodNotAllowed && w.Header().Get("Allow") != "GET, HEAD" {
			t.Fatal("missing Allow")
		}
	}
}
