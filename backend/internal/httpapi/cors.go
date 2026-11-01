package httpapi

import (
	"net/http"
	"strings"
)

func cors(w http.ResponseWriter, r *http.Request, origins []string) bool {
	w.Header().Add("Vary", "Origin")
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	allowed := false
	for _, candidate := range origins {
		if origin == candidate {
			allowed = true
			break
		}
	}
	if !allowed {
		apiError(w, r, 403, "origin_not_allowed")
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Expose-Headers", "ETag, X-Request-ID, Retry-After")
	if r.Method != "OPTIONS" {
		return true
	}
	w.Header().Add("Vary", "Access-Control-Request-Method")
	w.Header().Add("Vary", "Access-Control-Request-Headers")
	switch r.Header.Get("Access-Control-Request-Method") {
	case "GET", "POST", "PATCH", "PUT", "DELETE":
	default:
		apiError(w, r, 403, "cors_method_not_allowed")
		return false
	}
	for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
		switch strings.ToLower(strings.TrimSpace(header)) {
		case "", "authorization", "content-type", "idempotency-key", "if-match", "x-invite-token":
		default:
			apiError(w, r, 403, "cors_header_not_allowed")
			return false
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, If-Match, X-Invite-Token")
	w.Header().Set("Access-Control-Max-Age", "600")
	w.WriteHeader(204)
	return false
}
