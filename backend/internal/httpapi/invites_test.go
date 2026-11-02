package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
)

func TestInviteHTTPPreviewIsMinimalAndRedeemRequiresIdentity(t *testing.T) {
	f := setupApp(t)
	a, b := ids.New(), ids.New()
	ta, tb := f.token(a), f.token(b)
	call(t, f.handler, "GET", "/v1/me", ta, "", "", 200)
	body := `{"title":"Dinner","activity":"food","starts_at":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `","location":"Private location"}`
	p := call(t, f.handler, "POST", "/v1/plans", ta, body, "plan", 201)
	pid := p["id"].(string)
	link := call(t, f.handler, "POST", "/v1/plans/"+pid+"/invite-links", ta, `{"max_uses":2}`, "link", 201)
	raw := link["token"].(string)
	request := func(method, path, identity string, want int) map[string]any {
		t.Helper()
		r := httptest.NewRequest(method, path, nil)
		r.Header.Set("X-Invite-Token", raw)
		if identity != "" {
			r.Header.Set("Authorization", "Bearer "+identity)
		}
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		var result map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	preview := request("GET", "/v1/invite-preview", "", 200)
	if len(preview) != 3 || preview["title"] != "Dinner" || preview["location"] != nil || preview["members"] != nil {
		t.Fatal("preview leaked private fields", preview)
	}
	request("POST", "/v1/invite-redeem", "", 401)
	redeemed := request("POST", "/v1/invite-redeem", tb, 200)
	if redeemed["plan_id"] != pid {
		t.Fatal("wrong plan")
	}
	call(t, f.handler, "GET", "/v1/plans/"+pid, tb, "", "", 200)
	call(t, f.handler, "GET", "/v1/plans/"+pid+"/invite-links", tb, "", "", 403)
	call(t, f.handler, "DELETE", "/v1/invite-links/"+link["id"].(string), ta, "", "", 200)
	request("GET", "/v1/invite-preview", "", 404)
}
