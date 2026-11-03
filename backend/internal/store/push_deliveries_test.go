package store

import (
	"testing"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
)

func TestExpandPushEventQueuesOnlyEligibleDevices(t *testing.T) {
	s, ctx, actor, recipient, blocked, _ := inviteFixture(t)
	for _, user := range []string{actor, recipient, blocked} {
		if err := s.RegisterDevice(ctx, user, ids.New(), "android", "ExpoPushToken["+user+"]"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.Exec(ctx, "INSERT INTO blocks(blocker_id, blocked_id) VALUES ($1,$2)", blocked, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(ctx, `INSERT INTO outbox(actor_id,kind,payload)
		VALUES ($1,'friend.requested',jsonb_build_object('recipient_ids',to_jsonb($2::text[])))`, actor, []string{actor, recipient, blocked}); err != nil {
		t.Fatal(err)
	}
	events, err := s.ClaimOutbox(ctx, 10, 3, time.Minute)
	if err != nil || len(events) != 2 {
		t.Fatalf("claim: %d events, %v", len(events), err)
	}
	for _, event := range events {
		if ok, err := s.ExpandPushEvent(ctx, event); err != nil || !ok {
			t.Fatalf("expand: %v, %v", ok, err)
		}
		if ok, err := s.ExpandPushEvent(ctx, event); err != nil || ok {
			t.Fatalf("repeat expansion: %v, %v", ok, err)
		}
	}
	var count int
	var target string
	if err := s.db.QueryRow(ctx, "SELECT count(*), COALESCE(min(recipient_id::text),'') FROM push_deliveries").Scan(&count, &target); err != nil || count != 1 || target != recipient {
		t.Fatalf("queued deliveries: %d, %s, %v", count, target, err)
	}
}
