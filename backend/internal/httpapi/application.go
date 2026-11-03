package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/auth"
	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/HoangDinhBui/no-plans/backend/internal/store"
)

type application struct {
	db             *store.Store
	verifier       *auth.Verifier
	email          *auth.EmailGate
	inviteSecret   []byte
	previewLimiter *publicLimiter
}
type identity struct{ actor, token string }
type identityKey struct{}

func NewApplication(db *store.Store, verifier *auth.Verifier, email *auth.EmailGate, origins ...string) http.Handler {
	return NewApplicationWithOptions(db, verifier, email, Options{Origins: origins})
}

type Options struct {
	Origins      []string
	InviteSecret []byte
}

func NewApplicationWithOptions(db *store.Store, verifier *auth.Verifier, email *auth.EmailGate, options Options) http.Handler {
	a := &application{db: db, verifier: verifier, email: email, inviteSecret: append([]byte(nil), options.InviteSecret...), previewLimiter: newPublicLimiter(120, time.Minute, 4096)}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", a.me)
	mux.HandleFunc("/v1/devices/{deviceID}", a.device)
	mux.HandleFunc("/v1/reports", a.report)
	mux.HandleFunc("/v1/blocks", a.blocks)
	mux.HandleFunc("/v1/blocks/{userID}", a.unblock)
	mux.HandleFunc("/v1/groups", a.groups)
	mux.HandleFunc("/v1/groups/{groupID}", a.group)
	mux.HandleFunc("/v1/groups/{groupID}/members/{userID}", a.groupMember)
	mux.HandleFunc("/v1/groups/{groupID}/owner", a.groupOwner)
	mux.HandleFunc("/v1/groups/{groupID}/notifications", a.groupNotifications)
	mux.HandleFunc("/v1/friend-requests", a.friendRequests)
	mux.HandleFunc("/v1/friend-requests/{requestID}/accept", a.friendAccept)
	mux.HandleFunc("/v1/plans", a.plans)
	mux.HandleFunc("/v1/plans/{planID}", a.plan)
	mux.HandleFunc("/v1/plans/{planID}/rsvp", a.rsvp)
	mux.HandleFunc("/v1/plans/{planID}/members", a.planMembers)
	mux.HandleFunc("/v1/plans/{planID}/options", a.options)
	mux.HandleFunc("/v1/plans/{planID}/vote", a.vote)
	mux.HandleFunc("/v1/plans/{planID}/selection", a.selection)
	mux.HandleFunc("/v1/places", a.places)
	mux.HandleFunc("/v1/plans/{planID}/invite-links", a.inviteLinks)
	mux.HandleFunc("/v1/invite-links/{linkID}", a.revokeInviteLink)
	mux.HandleFunc("/v1/invite-redeem", a.redeemInvite)
	mux.HandleFunc("/v1/plans/{planID}/invitations", a.invite)
	mux.HandleFunc("/v1/plans/{planID}/invitations/{userID}", a.revoke)
	mux.HandleFunc("/v1/plans/{planID}/{action}", a.transition)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { apiError(w, r, 404, "not_found") })
	return withHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cors(w, r, options.Origins) {
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/v1/") && r.URL.Path != "/v1" {
			var ready func(context.Context) error
			if db != nil {
				ready = db.Ping
			}
			serveProbe(w, r, ready, w.Header().Get("X-Request-ID"))
			return
		}
		if db == nil {
			apiError(w, r, 503, "unavailable")
			return
		}
		if r.URL.Path == "/v1/invite-preview" {
			if !a.previewLimiter.allow(r.RemoteAddr, time.Now()) {
				apiFailure(w, r, store.ErrRateLimited)
				return
			}
			a.previewInvite(w, r)
			return
		}
		if verifier == nil {
			apiError(w, r, 503, "unavailable")
			return
		}
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			apiFailure(w, r, auth.ErrUnauthorized)
			return
		}
		p, err := verifier.Verify(r.Context(), parts[1])
		if err != nil {
			apiFailure(w, r, err)
			return
		}
		_, err = db.EnsureProfile(r.Context(), p.UserID)
		retryDeletion := r.Method == http.MethodDelete && errors.Is(err, plans.ErrForbidden) &&
			(r.URL.Path == "/v1/me" || strings.HasPrefix(r.URL.Path, "/v1/devices/"))
		if err != nil && !retryDeletion {
			apiFailure(w, r, err)
			return
		}
		if err = db.CheckAPIRate(r.Context(), p.UserID); err != nil {
			apiFailure(w, r, err)
			return
		}
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, identity{actor: p.UserID, token: parts[1]})))
	}))
}

func actor(r *http.Request) string { return r.Context().Value(identityKey{}).(identity).actor }
func (a *application) verified(w http.ResponseWriter, r *http.Request) bool {
	principal := r.Context().Value(identityKey{}).(identity)
	if err := a.email.Check(r.Context(), principal.token, principal.actor); err != nil {
		apiFailure(w, r, err)
		return false
	}
	return true
}

func method(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	for _, m := range allowed {
		if r.Method == m {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	apiError(w, r, 405, "method_not_allowed")
	return false
}
func pathID(w http.ResponseWriter, r *http.Request, key string) (string, bool) {
	id := r.PathValue(key)
	if !ids.Valid(id) {
		apiError(w, r, 400, "invalid_input")
		return "", false
	}
	return strings.ToLower(id), true
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		apiError(w, r, 415, "unsupported_media_type")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(target)
	if err == nil {
		var extra any
		if next := decoder.Decode(&extra); !errors.Is(next, io.EOF) {
			if next == nil {
				err = errors.New("multiple JSON values")
			} else {
				err = next
			}
		}
	}
	if err != nil {
		var size *http.MaxBytesError
		if errors.As(err, &size) {
			apiError(w, r, 413, "body_too_large")
		} else {
			apiError(w, r, 400, "invalid_json")
		}
		return false
	}
	return true
}

func apiError(w http.ResponseWriter, r *http.Request, status int, code string) {
	writeJSON(w, status, map[string]string{"code": code, "request_id": w.Header().Get("X-Request-ID")})
}
func apiFailure(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrVenueRequired):
		apiError(w, r, 409, "venue_required")
	case errors.Is(err, store.ErrInviteUnavailable):
		apiError(w, r, 404, "invite_unavailable")
	case errors.Is(err, store.ErrInviteKeyChanged):
		apiError(w, r, 409, "invite_key_changed")
	case errors.Is(err, store.ErrInviteNotConfigured):
		apiError(w, r, 503, "invite_not_configured")
	case errors.Is(err, store.ErrRateLimited):
		w.Header().Set("Retry-After", "60")
		apiError(w, r, 429, "rate_limited")
	case errors.Is(err, auth.ErrUnauthorized):
		w.Header().Set("WWW-Authenticate", "Bearer")
		apiError(w, r, 401, "unauthorized")
	case errors.Is(err, auth.ErrEmailUnverified):
		apiError(w, r, 403, "email_unverified")
	case errors.Is(err, auth.ErrAuthUnavailable):
		apiError(w, r, 503, "auth_unavailable")
	case errors.Is(err, plans.ErrForbidden):
		apiError(w, r, 403, "forbidden")
	case errors.Is(err, plans.ErrInvalid):
		apiError(w, r, 400, "invalid_input")
	case errors.Is(err, plans.ErrFull):
		apiError(w, r, 409, "plan_full")
	case errors.Is(err, plans.ErrClosed):
		apiError(w, r, 409, "plan_closed")
	case errors.Is(err, plans.ErrConflict):
		apiError(w, r, 409, "version_conflict")
	case errors.Is(err, store.ErrIdempotencyConflict):
		apiError(w, r, 409, "idempotency_conflict")
	default:
		slog.Error("request failed", "request_id", w.Header().Get("X-Request-ID"), "error_type", fmt.Sprintf("%T", err))
		apiError(w, r, 500, "internal_error")
	}
}
