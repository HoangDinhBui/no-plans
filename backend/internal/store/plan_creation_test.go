package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func TestGroupPlanSnapshotsMembershipAndReplaysCreation(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	s, err := openTestStore(t, ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	a, b, c := ids.New(), ids.New(), ids.New()
	for _, id := range []string{a, b, c} {
		if _, err = s.EnsureProfile(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	request, err := s.RequestFriend(ctx, a, b, "friend")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AcceptFriend(ctx, b, request.ID); err != nil {
		t.Fatal(err)
	}
	g, err := s.CreateGroup(ctx, a, "Test group", "group")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.AddGroupMember(ctx, g.ID, a, b); err != nil {
		t.Fatal(err)
	}
	input := CreatePlanInput{Title: "Cà phê", Activity: "coffee", StartsAt: time.Now().Add(time.Hour), Capacity: 2, Location: "Cafe", GroupID: g.ID}
	if _, err = s.CreatePlanFor(ctx, c, "forbidden", input); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("outsider invited private group", err)
	}
	p, err := s.CreatePlanFor(ctx, a, "create", input)
	if err != nil {
		t.Fatal(err)
	}
	if p.Members[a] != "going" || p.Members[b] != "invited" || p.GroupID != g.ID {
		t.Fatalf("bad snapshot %+v", p)
	}
	replay, err := s.CreatePlanFor(ctx, a, "create", input)
	if err != nil || replay.ID != p.ID {
		t.Fatal("created duplicate plan", err)
	}
	changed := input
	changed.Title = "Changed"
	if _, err = s.CreatePlanFor(ctx, a, "create", changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatal("different body reused key", err)
	}
	if err = s.RemoveGroupMember(ctx, g.ID, a, b); err != nil {
		t.Fatal(err)
	}
	if _, err = s.GetPlan(ctx, p.ID, b); err != nil {
		t.Fatal("group removal erased existing plan membership", err)
	}
	request, err = s.RequestFriend(ctx, a, c, "friend-c")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AcceptFriend(ctx, c, request.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.AddGroupMember(ctx, g.ID, a, c); err != nil {
		t.Fatal(err)
	}
	if _, err = s.GetPlan(ctx, p.ID, c); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("new group member automatically entered old plan", err)
	}
	if err = s.InvitePlanMember(ctx, p.ID, b, c); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("non-owner sent invite", err)
	}
	if err = s.InvitePlanMember(ctx, p.ID, a, c); err != nil {
		t.Fatal(err)
	}
	for _, user := range []string{a, b, c} {
		if err := s.RegisterDevice(ctx, user, ids.New(), "ios", "ExpoPushToken["+user+"]"); err != nil {
			t.Fatal(err)
		}
	}
	events, err := s.ClaimOutbox(ctx, 20, 3, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if _, err := s.ExpandPushEvent(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	for kind, expected := range map[string]string{"plan.created": b, "plan.invited": c} {
		var count int
		var recipient string
		err := s.db.QueryRow(ctx, `SELECT count(*), coalesce(min(d.recipient_id::text), '')
			FROM push_deliveries d JOIN outbox e ON e.id=d.event_id
			WHERE e.plan_id=$1 AND e.kind=$2`, p.ID, kind).Scan(&count, &recipient)
		if err != nil || count != 1 || recipient != expected {
			t.Fatalf("%s deliveries: %d, recipient %q, err %v", kind, count, recipient, err)
		}
	}
	if err := s.SetGroupNotificationsMuted(ctx, g.ID, c, true); err != nil {
		t.Fatal(err)
	}
	detail, err := s.GetGroup(ctx, g.ID, c)
	if err != nil || !detail.NotificationsMuted {
		t.Fatalf("mute preference: %+v, %v", detail, err)
	}
	for range len(events) {
		delivery, err := s.ClaimPushDelivery(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if delivery.Kind != "plan.invited" {
			continue
		}
		if allowed, err := s.PushRecipientEligible(ctx, delivery); err != nil || allowed {
			t.Fatalf("muted delivery eligible: %v, %v", allowed, err)
		}
		if err := s.SetGroupNotificationsMuted(ctx, g.ID, c, false); err != nil {
			t.Fatal(err)
		}
		if allowed, err := s.PushRecipientEligible(ctx, delivery); err != nil || !allowed {
			t.Fatalf("unmuted delivery eligible: %v, %v", allowed, err)
		}
		break
	}
	if err = s.RevokePlanMember(ctx, p.ID, a, c); err != nil {
		t.Fatal(err)
	}
	if _, err = s.GetPlan(ctx, p.ID, c); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("revoked member still reads plan", err)
	}
}
