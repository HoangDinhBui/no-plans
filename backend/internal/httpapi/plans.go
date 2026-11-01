package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/store"
)

func (a *application) plans(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "POST") {
		return
	}
	if r.Method == "GET" {
		limit, ok := pageLimit(w, r)
		if !ok {
			return
		}
		status := r.URL.Query().Get("status")
		if status == "" {
			status = "upcoming"
		}
		page, err := a.db.ListPlans(r.Context(), actor(r), status, r.URL.Query().Get("cursor"), limit)
		if err != nil {
			apiFailure(w, r, err)
			return
		}
		writeJSON(w, 200, page)
		return
	}
	var body store.CreatePlanInput
	if !decode(w, r, &body) {
		return
	}
	if !a.verified(w, r) {
		return
	}
	p, err := a.db.CreatePlanFor(r.Context(), actor(r), r.Header.Get("Idempotency-Key"), body)
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	w.Header().Set("ETag", fmt.Sprintf("\"%d\"", p.Version))
	writeJSON(w, 201, p)
}

func (a *application) plan(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "PATCH") {
		return
	}
	id, ok := pathID(w, r, "planID")
	if !ok {
		return
	}
	if r.Method == "PATCH" {
		header := strings.TrimSpace(r.Header.Get("If-Match"))
		if header == "" {
			apiError(w, r, 428, "version_required")
			return
		}
		if len(header) >= 2 && header[0] == '"' && header[len(header)-1] == '"' {
			header = header[1 : len(header)-1]
		}
		version, err := strconv.ParseInt(header, 10, 64)
		if err != nil || version < 1 {
			apiError(w, r, 400, "invalid_version")
			return
		}
		var body struct {
			Title    *string    `json:"title"`
			StartsAt *time.Time `json:"starts_at"`
			Capacity *int       `json:"capacity"`
			Location *string    `json:"location"`
		}
		if !decode(w, r, &body) {
			return
		}
		p, err := a.db.GetPlan(r.Context(), id, actor(r))
		if err != nil {
			apiFailure(w, r, err)
			return
		}
		if body.Title != nil {
			p.Title = *body.Title
		}
		if body.StartsAt != nil {
			p.StartsAt = *body.StartsAt
		}
		if body.Capacity != nil {
			p.Capacity = *body.Capacity
		}
		if body.Location != nil {
			p.Location = *body.Location
		}
		if err = a.db.EditPlan(r.Context(), id, actor(r), version, p.Title, p.StartsAt, p.Capacity, p.Location); err != nil {
			apiFailure(w, r, err)
			return
		}
	}
	p, err := a.db.GetPlan(r.Context(), id, actor(r))
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	w.Header().Set("ETag", fmt.Sprintf("\"%d\"", p.Version))
	writeJSON(w, 200, p)
}

func (a *application) rsvp(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "PUT") {
		return
	}
	id, ok := pathID(w, r, "planID")
	if !ok {
		return
	}
	var body struct {
		Response string `json:"response"`
	}
	if !decode(w, r, &body) {
		return
	}
	changed, err := a.db.Respond(r.Context(), id, actor(r), body.Response)
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"changed": changed})
}

func (a *application) planMembers(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET") {
		return
	}
	id, ok := pathID(w, r, "planID")
	if !ok {
		return
	}
	limit, ok := pageLimit(w, r)
	if !ok {
		return
	}
	page, err := a.db.ListPlanMembers(r.Context(), id, actor(r), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, page)
}

func (a *application) transition(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	if action != "confirm" && action != "cancel" && action != "complete" {
		apiError(w, r, 404, "not_found")
		return
	}
	if !method(w, r, "POST") {
		return
	}
	id, ok := pathID(w, r, "planID")
	if !ok {
		return
	}
	if err := a.db.TransitionPlan(r.Context(), id, actor(r), action); err != nil {
		apiFailure(w, r, err)
		return
	}
	p, err := a.db.GetPlan(r.Context(), id, actor(r))
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, p)
}

func (a *application) invite(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "POST") {
		return
	}
	id, ok := pathID(w, r, "planID")
	if !ok {
		return
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	if !a.verified(w, r) {
		return
	}
	if err := a.db.InvitePlanMember(r.Context(), id, actor(r), body.UserID); err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (a *application) revoke(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "DELETE") {
		return
	}
	id, ok := pathID(w, r, "planID")
	if !ok {
		return
	}
	target, ok := pathID(w, r, "userID")
	if !ok {
		return
	}
	if err := a.db.RevokePlanMember(r.Context(), id, actor(r), target); err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
