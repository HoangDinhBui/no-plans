package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
)

func TestEditRejectsStaleVersionAndPagingDoesNotRepeat(t *testing.T) {
	f := setupApp(t)
	token := f.token(ids.New())
	call(t, f.handler, "GET", "/v1/me", token, "", "", 200)
	var plan map[string]any
	at := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	for _, name := range []string{"A", "B", "C"} {
		plan = call(t, f.handler, "POST", "/v1/plans", token, `{"title":"`+name+`","activity":"coffee","starts_at":"`+at+`","capacity":0}`, name, 201)
	}
	id := plan["id"].(string)
	patch := func(version string, want int) {
		r := httptest.NewRequest("PATCH", "/v1/plans/"+id, strings.NewReader(`{"location":"Cafe","title":"Updated"}`))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("If-Match", version)
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("patch status %d body=%s", w.Code, w.Body.String())
		}
	}
	patch(`"1"`, 200)
	patch(`"1"`, 409)
	p := call(t, f.handler, "GET", "/v1/plans/"+id, token, "", "", 200)
	if p["title"] != "Updated" || p["activity"] != "coffee" || p["location"] != "Cafe" {
		t.Fatal("partial patch lost fields", p)
	}
	seen := map[string]bool{}
	path := "/v1/plans?limit=1"
	for i := 0; i < 4; i++ {
		page := call(t, f.handler, "GET", path, token, "", "", 200)
		for _, item := range page["items"].([]any) {
			id := item.(map[string]any)["id"].(string)
			if seen[id] {
				t.Fatal("cursor repeated plan")
			}
			seen[id] = true
		}
		cursor, ok := page["next_cursor"].(string)
		if !ok {
			break
		}
		path = "/v1/plans?limit=1&cursor=" + cursor
	}
	if len(seen) != 3 {
		data, _ := json.Marshal(seen)
		t.Fatalf("paging skipped plans: %s", data)
	}
}
