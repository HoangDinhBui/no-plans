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

func TestPushDeliveryTicketLifecycle(t *testing.T) {
	s, ctx, actor, recipient, _, _ := inviteFixture(t)
	device := ids.New()
	if err := s.RegisterDevice(ctx, recipient, device, "ios", "ExpoPushToken[test]"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(ctx, `INSERT INTO push_deliveries(event_id, recipient_id, device_id)
 SELECT id, $1, $2 FROM outbox WHERE actor_id = $3 LIMIT 1`, recipient, device, actor); err != nil {
		t.Fatal(err)
	}
	first, err := s.ClaimPushDelivery(ctx)
	if err != nil || first.TicketID != "" {
		t.Fatalf("first claim: %+v, %v", first, err)
	}
	immediate := time.Duration(0)
	if ok, err := s.ResolvePushDelivery(ctx, first, "test-ticket", "", &immediate); err != nil || !ok {
		t.Fatalf("save ticket: %v, %v", ok, err)
	}
	receipt, err := s.ClaimPushDelivery(ctx)
	if err != nil || receipt.TicketID != "test-ticket" || receipt.Attempt != first.Attempt+1 {
		t.Fatalf("receipt claim: %+v, %v", receipt, err)
	}
	if ok, err := s.ResolvePushDelivery(ctx, first, "", "", nil); err != nil || ok {
		t.Fatalf("stale completion: %v, %v", ok, err)
	}
	if ok, err := s.ResolvePushDelivery(ctx, receipt, "", "", nil); err != nil || !ok {
		t.Fatalf("complete: %v, %v", ok, err)
	}
	var completed bool
	if err := s.db.QueryRow(ctx, "SELECT completed_at IS NOT NULL FROM push_deliveries").Scan(&completed); err != nil || !completed {
		t.Fatalf("completion persisted: %v, %v", completed, err)
	}
}

func TestPushDeliveryRevokesOnlySentRegistration(t *testing.T) {
	for _, refreshed := range []bool{false, true} {
		name := "original registration"
		if refreshed {
			name = "refreshed registration"
		}
		t.Run(name, func(t *testing.T) {
			s, ctx, actor, recipient, _, _ := inviteFixture(t)
			device := ids.New()
			if err := s.RegisterDevice(ctx, recipient, device, "ios", "ExpoPushToken[original]"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.Exec(ctx, `INSERT INTO push_deliveries(event_id, recipient_id, device_id)
   SELECT id, $1, $2 FROM outbox WHERE actor_id = $3 LIMIT 1`, recipient, device, actor); err != nil {
				t.Fatal(err)
			}
			send, err := s.ClaimPushDelivery(ctx)
			if err != nil {
				t.Fatal(err)
			}
			immediate := time.Duration(0)
			if ok, err := s.ResolvePushDelivery(ctx, send, "test-ticket", "", &immediate); err != nil || !ok {
				t.Fatalf("save ticket: %v, %v", ok, err)
			}
			if refreshed {
				if err := s.RegisterDevice(ctx, recipient, device, "ios", "ExpoPushToken[refreshed]"); err != nil {
					t.Fatal(err)
				}
			}
			receipt, err := s.ClaimPushDelivery(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if ok, err := s.ResolvePushDelivery(ctx, receipt, "", "DeviceNotRegistered", nil); err != nil || !ok {
				t.Fatalf("resolve receipt: %v, %v", ok, err)
			}
			var exists bool
			if err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM devices WHERE id = $1)", device).Scan(&exists); err != nil || exists != refreshed {
				t.Fatalf("device exists: %v, want %v, err %v", exists, refreshed, err)
			}
		})
	}
}
