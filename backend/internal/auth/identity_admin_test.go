package auth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type identityTransport func(*http.Request) (*http.Response, error)

func (f identityTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestIdentityAdminDeleteRetry(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	for _, tc := range []struct {
		name   string
		status int
		body   string
		fail   bool
	}{
		{"deleted", 200, "{}", false},
		{"already deleted", 404, `{"error_code":"user_not_found"}`, false},
		{"wrong route", 404, "Not Found", true},
		{"unavailable", 503, "Unavailable", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			admin, err := NewIdentityAdmin("https://fixture.example", "fixture-key")
			if err != nil {
				t.Fatal(err)
			}
			admin.client.Transport = identityTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != "DELETE" || r.URL.Path != "/auth/v1/admin/users/"+id || r.Header.Get("Authorization") != "Bearer fixture-key" || r.Header.Get("apikey") != "fixture-key" {
					t.Fatal("unexpected admin request")
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})
			if err := admin.DeleteUser(context.Background(), id); (err != nil) != tc.fail {
				t.Fatalf("unexpected result: %v", err)
			}
		})
	}
}
