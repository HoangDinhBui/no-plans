package httpapi

import "net/http"

func (a *application) blocks(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "POST") {
		return
	}
	if r.Method == "GET" {
		limit, ok := pageLimit(w, r)
		if !ok {
			return
		}
		page, err := a.db.ListBlocked(r.Context(), actor(r), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			apiFailure(w, r, err)
			return
		}
		writeJSON(w, 200, page)
		return
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := a.db.BlockUser(r.Context(), actor(r), body.UserID); err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (a *application) unblock(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "DELETE") {
		return
	}
	target, ok := pathID(w, r, "userID")
	if !ok {
		return
	}
	if err := a.db.UnblockUser(r.Context(), actor(r), target); err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
