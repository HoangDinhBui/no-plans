package httpapi

import "net/http"

func (a *application) inviteLinks(w http.ResponseWriter, r *http.Request) {
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
		page, err := a.db.ListInviteLinks(r.Context(), id, actor(r), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			apiFailure(w, r, err)
			return
		}
		writeJSON(w, 200, page)
		return
	}
	var body struct {
		MaxUses *int `json:"max_uses"`
	}
	if !decode(w, r, &body) {
		return
	}
	maximum := 100
	if body.MaxUses != nil {
		maximum = *body.MaxUses
	}
	if !a.verified(w, r) {
		return
	}
	link, err := a.db.CreateInviteLink(r.Context(), id, actor(r), r.Header.Get("Idempotency-Key"), maximum, a.inviteSecret)
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 201, link)
}
func (a *application) revokeInviteLink(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "DELETE") {
		return
	}
	id, ok := pathID(w, r, "linkID")
	if !ok {
		return
	}
	if err := a.db.RevokeInviteLink(r.Context(), id, actor(r)); err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (a *application) previewInvite(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET") {
		return
	}
	preview, err := a.db.PreviewInvite(r.Context(), r.Header.Get("X-Invite-Token"))
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, preview)
}
func (a *application) redeemInvite(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "POST") {
		return
	}
	id, err := a.db.RedeemInvite(r.Context(), r.Header.Get("X-Invite-Token"), actor(r))
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]string{"plan_id": id})
}
