package store

import (
	"testing"
	"time"
)

func TestOutboxLeaseRecovery(t *testing.T) {
	s, ctx, _, _, _, plan := inviteFixture(t)
	events, err := s.ClaimOutbox(ctx, 10, 2, time.Minute)
	if err != nil || len(events) != 1 || events[0].PlanID != plan.ID {
		t.Fatalf("claim: %d events, %v", len(events), err)
	}
	first := events[0]
	if events, err := s.ClaimOutbox(ctx, 10, 2, time.Minute); err != nil || len(events) != 0 {
		t.Fatalf("live lease reclaimed: %d events, %v", len(events), err)
	}
	expire := func() {
		t.Helper()
		if _, err := s.db.Exec(ctx, "UPDATE outbox SET lease_until = now() - interval '1 second' WHERE id = $1", first.ID); err != nil {
			t.Fatal(err)
		}
	}
	expire()
	events, err = s.ClaimOutbox(ctx, 10, 2, time.Minute)
	if err != nil || len(events) != 1 || events[0].Attempt != first.Attempt+1 {
		t.Fatalf("reclaim: %+v, %v", events, err)
	}
	if resolved, err := s.ResolveOutbox(ctx, first, nil); err != nil || resolved {
		t.Fatalf("stale acknowledgment: %v, %v", resolved, err)
	}
	expire()
	if events, err := s.ClaimOutbox(ctx, 10, 2, time.Minute); err != nil || len(events) != 0 {
		t.Fatalf("exhausted event reclaimed: %d events, %v", len(events), err)
	}
	var dead bool
	if err := s.db.QueryRow(ctx, "SELECT dead_letter_at IS NOT NULL FROM outbox WHERE id = $1", first.ID).Scan(&dead); err != nil || !dead {
		t.Fatalf("dead letter: %v, %v", dead, err)
	}
}

func TestOutboxRetryThenComplete(t *testing.T) {
	s, ctx, _, _, _, _ := inviteFixture(t)
	events, err := s.ClaimOutbox(ctx, 1, 3, time.Minute)
	if err != nil || len(events) != 1 {
		t.Fatalf("initial claim: %d events, %v", len(events), err)
	}
	event := events[0]
	delay := time.Hour
	if ok, err := s.ResolveOutbox(ctx, event, &delay); err != nil || !ok {
		t.Fatalf("schedule retry: %v, %v", ok, err)
	}
	if events, err := s.ClaimOutbox(ctx, 1, 3, time.Minute); err != nil || len(events) != 0 {
		t.Fatalf("retry ran early: %d events, %v", len(events), err)
	}
	if _, err := s.db.Exec(ctx, "UPDATE outbox SET available_at = now() WHERE id = $1", event.ID); err != nil {
		t.Fatal(err)
	}
	events, err = s.ClaimOutbox(ctx, 1, 3, time.Minute)
	if err != nil || len(events) != 1 || events[0].Attempt != event.Attempt+1 {
		t.Fatalf("retry claim: %+v, %v", events, err)
	}
	if ok, err := s.ResolveOutbox(ctx, events[0], nil); err != nil || !ok {
		t.Fatalf("complete: %v, %v", ok, err)
	}
	if events, err := s.ClaimOutbox(ctx, 1, 3, time.Minute); err != nil || len(events) != 0 {
		t.Fatalf("completed event reclaimed: %d events, %v", len(events), err)
	}
}
