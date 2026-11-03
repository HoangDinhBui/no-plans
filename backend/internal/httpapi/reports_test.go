package httpapi

import (
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
)

func TestReportHTTP(t *testing.T) {
	f := setupApp(t)
	a, b := ids.New(), ids.New()
	ta, tb := f.token(a), f.token(b)
	call(t, f.handler, "GET", "/v1/me", ta, "", "", 200)
	call(t, f.handler, "GET", "/v1/me", tb, "", "", 200)
	body := `{"target_id":"` + b + `","reason":"spam"}`
	call(t, f.handler, "POST", "/v1/reports", "", body, "", 401)
	call(t, f.handler, "POST", "/v1/reports", ta, `{"target_id":"`+a+`","reason":"spam"}`, "", 403)
	report := call(t, f.handler, "POST", "/v1/reports", ta, body, "", 201)
	if id, ok := report["id"].(string); !ok || !ids.Valid(id) {
		t.Fatalf("invalid report id: %v", report)
	}
}
