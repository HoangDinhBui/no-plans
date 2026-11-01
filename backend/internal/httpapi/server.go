package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

// New exposes public probes. A missing readiness dependency fails closed.
func New(readiness func(context.Context) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			http.Error(w, "request unavailable", 503)
			return
		}
		requestID := hex.EncodeToString(id)
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.URL.Path != "/healthz" && r.URL.Path != "/readyz" {
			writeJSON(w, 404, map[string]string{"code": "not_found", "request_id": requestID})
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			writeJSON(w, 405, map[string]string{"code": "method_not_allowed", "request_id": requestID})
			return
		}
		status, value := 200, "ok"
		if r.URL.Path == "/readyz" {
			value = "ready"
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if readiness == nil || readiness(ctx) != nil {
				status, value = 503, "unavailable"
			}
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(status)
			return
		}
		writeJSON(w, status, map[string]string{"status": value, "request_id": requestID})
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
