package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/HoangDinhBui/no-plans/backend/internal/store"
)

func (a *application) options(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "POST") {
		return
	}
	id, ok := pathID(w, r, "planID")
	if !ok {
		return
	}
	if r.Method == "GET" {
		limit, ok := pageLimit(w, r)
		if !ok {
			return
		}
		page, err := a.db.ListOptions(r.Context(), id, actor(r), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			apiFailure(w, r, err)
			return
		}
		writeJSON(w, 200, page)
		return
	}
	var body store.OptionInput
	if !decode(w, r, &body) {
		return
	}
	option, err := a.db.AddOption(r.Context(), id, actor(r), r.Header.Get("Idempotency-Key"), body)
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 201, option)
}
func (a *application) vote(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "PUT") {
		return
	}
	id, ok := pathID(w, r, "planID")
	if !ok {
		return
	}
	var body struct {
		OptionID string `json:"option_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	changed, err := a.db.CastVote(r.Context(), id, actor(r), body.OptionID)
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"changed": changed})
}
func (a *application) selection(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "PUT") {
		return
	}
	id, ok := pathID(w, r, "planID")
	if !ok {
		return
	}
	version, ok := expectedVersion(w, r)
	if !ok {
		return
	}
	var body struct {
		OptionID string `json:"option_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := a.db.SelectOption(r.Context(), id, actor(r), version, body.OptionID); err != nil {
		apiFailure(w, r, err)
		return
	}
	p, err := a.db.GetPlan(r.Context(), id, actor(r))
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	w.Header().Set("ETag", fmt.Sprintf("\"%d\"", p.Version))
	writeJSON(w, 200, p)
}
func expectedVersion(w http.ResponseWriter, r *http.Request) (int64, bool) {
	header := strings.TrimSpace(r.Header.Get("If-Match"))
	if header == "" {
		apiError(w, r, 428, "version_required")
		return 0, false
	}
	if len(header) >= 2 && header[0] == '"' && header[len(header)-1] == '"' {
		header = header[1 : len(header)-1]
	}
	version, err := strconv.ParseInt(header, 10, 64)
	if err != nil || version < 1 {
		apiError(w, r, 400, "invalid_version")
		return 0, false
	}
	return version, true
}
