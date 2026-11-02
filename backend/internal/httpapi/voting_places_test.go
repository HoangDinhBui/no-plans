package httpapi

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/store"
)

func TestVotingAndDiscoveryHTTPFlow(t *testing.T) {
	f := setupApp(t)
	a, b := ids.New(), ids.New()
	ta, tb := f.token(a), f.token(b)
	call(t, f.handler, "GET", "/v1/me", ta, "", "", 200)
	call(t, f.handler, "GET", "/v1/me", tb, "", "", 200)
	priceMin, priceMax := 30000, 60000
	idea := store.Place{ID: ids.New(), Name: "Cà phê cùng hội", Kind: "idea", Activity: "coffee", Area: "TP.HCM", BudgetMin: &priceMin, BudgetMax: &priceMax, PriceNote: "Ngân sách gợi ý", PeopleMin: 1, PeopleMax: 6, Active: true}
	if err := f.db.UpsertPlaces(context.Background(), []store.Place{idea}); err != nil {
		t.Fatal(err)
	}
	page := call(t, f.handler, "GET", "/v1/places?activity=coffee&max_budget=60000&people=2", ta, "", "", 200)
	if len(page["items"].([]any)) != 1 {
		t.Fatal("catalogue filter failed")
	}
	for _, path := range []string{"/v1/places?max_budget=-1", "/v1/places?people=invalid", "/v1/places?kind=unknown"} {
		call(t, f.handler, "GET", path, ta, "", "", 400)
	}
	p := call(t, f.handler, "POST", "/v1/plans", ta, `{"title":"Coffee","activity":"coffee","starts_at":"`+time.Now().Add(time.Hour).UTC().Format(time.RFC3339)+`"}`, "plan", 201)
	id := p["id"].(string)
	option := call(t, f.handler, "POST", "/v1/plans/"+id+"/options", ta, `{"name":"Quán A","address":"Điểm hẹn A"}`, "option", 201)
	oid := option["id"].(string)
	call(t, f.handler, "GET", "/v1/plans/"+id+"/options", tb, "", "", 403)
	call(t, f.handler, "PUT", "/v1/plans/"+id+"/vote", tb, `{"option_id":"`+oid+`"}`, "", 403)
	call(t, f.handler, "PUT", "/v1/plans/"+id+"/vote", ta, `{"option_id":"`+oid+`"}`, "", 200)
	options := call(t, f.handler, "GET", "/v1/plans/"+id+"/options", ta, "", "", 200)
	row := options["items"].([]any)[0].(map[string]any)
	if row["vote_count"] != float64(1) || row["my_vote"] != true {
		t.Fatal("vote totals wrong")
	}
	current := call(t, f.handler, "GET", "/v1/plans/"+id, ta, "", "", 200)
	r := httptest.NewRequest("PUT", "/v1/plans/"+id+"/selection", strings.NewReader(`{"option_id":"`+oid+`"}`))
	r.Header.Set("Authorization", "Bearer "+ta)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("If-Match", fmt.Sprintf("\"%.0f\"", current["version"]))
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("select status %d %s", w.Code, w.Body.String())
	}
	call(t, f.handler, "POST", "/v1/plans/"+id+"/confirm", ta, "", "", 200)
	call(t, f.handler, "PUT", "/v1/plans/"+id+"/vote", ta, `{"option_id":"`+oid+`"}`, "", 409)
}

func TestIdeaSelectionReturnsVenueRequired(t *testing.T) {
	f := setupApp(t)
	actor := ids.New()
	token := f.token(actor)
	call(t, f.handler, "GET", "/v1/me", token, "", "", 200)
	idea := store.Place{ID: ids.New(), Name: "Đi ăn cùng nhau", Kind: "idea", Activity: "food", Area: "TP.HCM", PeopleMin: 1, Active: true}
	if err := f.db.UpsertPlaces(context.Background(), []store.Place{idea}); err != nil {
		t.Fatal(err)
	}
	p := call(t, f.handler, "POST", "/v1/plans", token, `{"title":"Dinner","activity":"food","starts_at":"`+time.Now().Add(time.Hour).UTC().Format(time.RFC3339)+`"}`, "plan", 201)
	id := p["id"].(string)
	option := call(t, f.handler, "POST", "/v1/plans/"+id+"/options", token, `{"place_id":"`+idea.ID+`"}`, "idea", 201)
	if option["kind"] != "idea" {
		t.Fatal("missing option classification", option)
	}
	current := call(t, f.handler, "GET", "/v1/plans/"+id, token, "", "", 200)
	r := httptest.NewRequest("PUT", "/v1/plans/"+id+"/selection", strings.NewReader(`{"option_id":"`+option["id"].(string)+`"}`))
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("If-Match", fmt.Sprintf("\"%.0f\"", current["version"]))
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	if w.Code != 409 || !strings.Contains(w.Body.String(), `"code":"venue_required"`) {
		t.Fatalf("unexpected selection contract: %d %s", w.Code, w.Body.String())
	}
}
