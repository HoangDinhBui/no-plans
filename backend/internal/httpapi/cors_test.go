package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestCORSOnlyAllowsConfiguredOriginAndHeaders(t *testing.T) {
	f := setupApp(t)
	for _, tc := range []struct {
		origin, headers string
		status          int
	}{{"https://app.no-plans.test", "authorization, content-type, if-match", 204}, {"https://evil.example", "authorization", 403}, {"https://app.no-plans.test", "x-secret-header", 403}} {
		r := httptest.NewRequest("OPTIONS", "/v1/me", nil)
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Access-Control-Request-Method", "PATCH")
		r.Header.Set("Access-Control-Request-Headers", tc.headers)
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("CORS %s: got %d body=%s", tc.origin, w.Code, w.Body.String())
		}
		if tc.status == 204 && w.Header().Get("Access-Control-Allow-Origin") != tc.origin {
			t.Fatal("allowed origin missing")
		}
		if tc.status == 403 && w.Header().Get("Access-Control-Allow-Origin") == tc.origin && tc.origin == "https://evil.example" {
			t.Fatal("untrusted origin reflected")
		}
	}
}
