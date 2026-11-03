package httpapi

import "net/http"

func (a *application) report(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "POST") {
		return
	}
	var body struct {
		TargetID string `json:"target_id"`
		PlanID   string `json:"plan_id"`
		Reason   string `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	id, err := a.db.CreateReport(r.Context(), actor(r), body.TargetID, body.PlanID, body.Reason)
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}
