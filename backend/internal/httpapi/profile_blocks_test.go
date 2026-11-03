package httpapi

import (
	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"testing"
)

func TestPartialProfilePatchPreservesAvatar(t *testing.T) {
	f := setupApp(t)
	token := f.token(ids.New())
	call(t, f.handler, "GET", "/v1/me", token, "", "", 200)
	call(t, f.handler, "PATCH", "/v1/me", token, `{"display_name":"Name","avatar_url":"https://example.com/avatar.png"}`, "", 200)
	p := call(t, f.handler, "PATCH", "/v1/me", token, `{"display_name":"New name"}`, "", 200)
	if p["avatar_url"] != "https://example.com/avatar.png" {
		t.Fatal("partial name update erased avatar")
	}
}

func TestBlockRoutesPreventInvitations(t *testing.T) {
	f := setupApp(t)
	a, b := ids.New(), ids.New()
	ta, tb := f.token(a), f.token(b)
	call(t, f.handler, "GET", "/v1/me", ta, "", "", 200)
	call(t, f.handler, "GET", "/v1/me", tb, "", "", 200)
	call(t, f.handler, "POST", "/v1/blocks", tb, `{"user_id":"`+a+`"}`, "", 200)
	page := call(t, f.handler, "GET", "/v1/blocks", tb, "", "", 200)
	if len(page["items"].([]any)) != 1 {
		t.Fatal("blocked list missing user")
	}
	call(t, f.handler, "POST", "/v1/friend-requests", ta, `{"recipient_id":"`+b+`"}`, "blocked", 403)
	call(t, f.handler, "DELETE", "/v1/blocks/"+a, tb, "", "", 200)
	call(t, f.handler, "POST", "/v1/friend-requests", ta, `{"recipient_id":"`+b+`"}`, "unblocked", 201)
}

func TestDeleteProfileRejectsExistingToken(t *testing.T) {
	f := setupApp(t)
	token := f.token(ids.New())
	call(t, f.handler, "GET", "/v1/me", token, "", "", 200)
	call(t, f.handler, "DELETE", "/v1/me", "", "", "", 401)
	call(t, f.handler, "DELETE", "/v1/me", token, "", "", 200)
	call(t, f.handler, "DELETE", "/v1/me", token, "", "", 200)
	call(t, f.handler, "GET", "/v1/me", token, "", "", 403)
}
