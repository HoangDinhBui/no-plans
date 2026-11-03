package httpapi

import (
	"net/http"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func (a *application) me(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "PATCH", "DELETE") {
		return
	}
	if r.Method == "DELETE" {
		if err := a.db.DeleteAccount(r.Context(), actor(r)); err != nil {
			apiFailure(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if r.Method == "GET" {
		p, err := a.db.EnsureProfile(r.Context(), actor(r))
		if err != nil {
			apiFailure(w, r, err)
			return
		}
		writeJSON(w, 200, p)
		return
	}
	var body struct {
		DisplayName *string `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
	}
	if !decode(w, r, &body) {
		return
	}
	p, err := a.db.PatchProfile(r.Context(), actor(r), body.DisplayName, body.AvatarURL)
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, p)
}

func (a *application) groups(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "POST") {
		return
	}
	if r.Method == "GET" {
		limit, ok := pageLimit(w, r)
		if !ok {
			return
		}
		page, err := a.db.ListGroups(r.Context(), actor(r), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			apiFailure(w, r, err)
			return
		}
		writeJSON(w, 200, page)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &body) {
		return
	}
	g, err := a.db.CreateGroup(r.Context(), actor(r), body.Name, r.Header.Get("Idempotency-Key"))
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 201, g)
}
func (a *application) group(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET") {
		return
	}
	id, ok := pathID(w, r, "groupID")
	if !ok {
		return
	}
	g, err := a.db.GetGroup(r.Context(), id, actor(r))
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, g)
}
func (a *application) groupMember(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "POST", "DELETE") {
		return
	}
	id, ok := pathID(w, r, "groupID")
	if !ok {
		return
	}
	target, ok := pathID(w, r, "userID")
	if !ok {
		return
	}
	if r.Method == "POST" {
		if !a.verified(w, r) {
			return
		}
		if err := a.db.AddGroupMember(r.Context(), id, actor(r), target); err != nil {
			apiFailure(w, r, err)
			return
		}
	} else {
		if err := a.db.RemoveGroupMember(r.Context(), id, actor(r), target); err != nil {
			apiFailure(w, r, err)
			return
		}
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (a *application) groupOwner(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "PUT") {
		return
	}
	id, ok := pathID(w, r, "groupID")
	if !ok {
		return
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := a.db.TransferGroup(r.Context(), id, actor(r), body.UserID); err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (a *application) friendRequests(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "POST") {
		return
	}
	if r.Method == "GET" {
		limit, ok := pageLimit(w, r)
		if !ok {
			return
		}
		page, err := a.db.ListFriendships(r.Context(), actor(r), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			apiFailure(w, r, err)
			return
		}
		writeJSON(w, 200, page)
		return
	}
	var body struct {
		RecipientID string `json:"recipient_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	if !a.verified(w, r) {
		return
	}
	f, err := a.db.RequestFriend(r.Context(), actor(r), body.RecipientID, r.Header.Get("Idempotency-Key"))
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 201, f)
}
func (a *application) friendAccept(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "POST") {
		return
	}
	id, ok := pathID(w, r, "requestID")
	if !ok {
		return
	}
	f, err := a.db.AcceptFriend(r.Context(), actor(r), id)
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, f)
}

func (a *application) groupNotifications(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "PUT") {
		return
	}
	id, ok := pathID(w, r, "groupID")
	if !ok {
		return
	}
	var body struct {
		Muted *bool `json:"muted"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.Muted == nil {
		apiFailure(w, r, plans.ErrInvalid)
		return
	}
	if err := a.db.SetGroupNotificationsMuted(r.Context(), id, actor(r), *body.Muted); err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"muted": *body.Muted})
}
