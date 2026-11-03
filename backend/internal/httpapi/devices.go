package httpapi

import "net/http"

func (a *application) device(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "PUT", "DELETE") {
		return
	}
	id, ok := pathID(w, r, "deviceID")
	if !ok {
		return
	}
	var err error
	if r.Method == "DELETE" {
		err = a.db.RevokeDevice(r.Context(), actor(r), id)
	} else {
		var body struct {
			Platform  string `json:"platform"`
			PushToken string `json:"push_token"`
		}
		if !decode(w, r, &body) {
			return
		}
		err = a.db.RegisterDevice(r.Context(), actor(r), id, body.Platform, body.PushToken)
	}
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
