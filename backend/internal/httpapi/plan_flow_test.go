package httpapi

import (
	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"strings"
	"testing"
	"time"
)

func TestPlanHTTPFlowAndBoundaries(t *testing.T) {
	f := setupApp(t)
	a, b, c := ids.New(), ids.New(), ids.New()
	ta, tb, tc := f.token(a), f.token(b), f.token(c)
	for _, token := range []string{ta, tb, tc} {
		call(t, f.handler, "GET", "/v1/me", token, "", "", 200)
	}
	req := call(t, f.handler, "POST", "/v1/friend-requests", ta, `{"recipient_id":"`+b+`"}`, "friend", 201)
	call(t, f.handler, "POST", "/v1/friend-requests/"+req["id"].(string)+"/accept", tb, "", "", 200)
	group := call(t, f.handler, "POST", "/v1/groups", ta, `{"name":"Cafe"}`, "group", 201)
	gid := group["id"].(string)
	call(t, f.handler, "POST", "/v1/groups/"+gid+"/members/"+b, ta, "", "", 200)
	body := `{"title":"Tối nay cà phê","activity":"coffee","starts_at":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `","capacity":2,"location":"Cafe","group_id":"` + gid + `"}`
	tooLarge := strings.Replace(body, `"capacity":2`, `"capacity":2147483648`, 1)
	call(t, f.handler, "POST", "/v1/plans", ta, tooLarge, "too-large", 400)
	p := call(t, f.handler, "POST", "/v1/plans", ta, body, "plan", 201)
	id := p["id"].(string)
	members := call(t, f.handler, "GET", "/v1/plans/"+id+"/members", tb, "", "", 200)
	if len(members["items"].([]any)) != 2 {
		t.Fatal("participant list incomplete")
	}
	call(t, f.handler, "GET", "/v1/plans/"+id+"/members", tc, "", "", 403)
	repeat := call(t, f.handler, "POST", "/v1/plans", ta, body, "plan", 201)
	if repeat["id"] != id {
		t.Fatal("plan duplicated on HTTP retry")
	}
	call(t, f.handler, "GET", "/v1/plans/"+id, tc, "", "", 403)
	response := call(t, f.handler, "PUT", "/v1/plans/"+id+"/rsvp", tb, `{"response":"going"}`, "", 200)
	if response["changed"] != true {
		t.Fatal("RSVP not saved")
	}
	response = call(t, f.handler, "PUT", "/v1/plans/"+id+"/rsvp", tb, `{"response":"going"}`, "", 200)
	if response["changed"] != false {
		t.Fatal("replay not idempotent")
	}
	call(t, f.handler, "POST", "/v1/plans/"+id+"/confirm", tb, "", "", 403)
	call(t, f.handler, "POST", "/v1/plans/"+id+"/confirm", ta, "", "", 200)
	call(t, f.handler, "GET", "/v1/plans/"+id, tb, "", "", 200)
	call(t, f.handler, "POST", "/v1/plans/"+id+"/cancel", ta, "", "", 200)
	call(t, f.handler, "PUT", "/v1/plans/"+id+"/rsvp", tb, `{"response":"maybe"}`, "", 409)
	for _, path := range []string{"/v1/groups?limit=101", "/v1/groups?cursor=not-uuid", "/v1/plans?limit=0", "/v1/plans?cursor=invalid", "/v1/plans?status=unknown"} {
		call(t, f.handler, "GET", path, ta, "", "", 400)
	}
	for _, path := range []string{"/v1/groups", "/v1/friend-requests", "/v1/plans?status=past"} {
		list := call(t, f.handler, "GET", path, ta, "", "", 200)
		if len(list["items"].([]any)) != 1 {
			t.Fatalf("list %s missing entity: %v", path, list)
		}
	}
	call(t, f.handler, "PATCH", "/v1/plans/"+id, ta, `{"title":"missing version"}`, "", 428)
}
