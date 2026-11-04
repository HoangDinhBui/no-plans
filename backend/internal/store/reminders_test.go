package store

import (
	"testing"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
)

func TestPlanReminderBecomesStaleAfterReschedule(t *testing.T) {
	s, ctx, owner, _, _, plan := inviteFixture(t)
	if _, err := s.db.Exec(ctx, "UPDATE plans SET status='confirmed', starts_at=now()+interval '30 minutes' WHERE id=$1", plan.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterDevice(ctx, owner, ids.New(), "ios", "ExpoPushToken[reminder]"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.QueuePlanReminders(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.Exec(ctx, "UPDATE plans SET version=version+1 WHERE id=$1", plan.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.QueuePlanReminders(ctx); err != nil {
		t.Fatal(err)
	}
	events, err := s.ClaimOutbox(ctx, 10, 3, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, event := range events {
		if event.Kind == "plan.reminder" {
			count++
		}
		if ok, err := s.ExpandPushEvent(ctx, event); err != nil || !ok {
			t.Fatalf("expand: %v %v", ok, err)
		}
	}
	if count != 1 {
		t.Fatalf("got %d reminders, want 1", count)
	}
	delivery, err := s.ClaimPushDelivery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if current, err := s.CurrentPlanReminder(ctx, delivery); err != nil || !current {
		t.Fatalf("initial reminder: %v %v", current, err)
	}
	if _, err := s.db.Exec(ctx, "UPDATE plans SET status='cancelled' WHERE id=$1", plan.ID); err != nil {
		t.Fatal(err)
	}
	if current, err := s.CurrentPlanReminder(ctx, delivery); err != nil || current {
		t.Fatalf("cancelled reminder: %v %v", current, err)
	}
	if _, err := s.db.Exec(ctx, "UPDATE plans SET status='confirmed', starts_at=starts_at+interval '1 hour', version=version+1 WHERE id=$1", plan.ID); err != nil {
		t.Fatal(err)
	}
	if current, err := s.CurrentPlanReminder(ctx, delivery); err != nil || current {
		t.Fatalf("rescheduled reminder: %v %v", current, err)
	}
}

func TestQueuePlanRemindersPreservesLegacySchedule(t *testing.T) {
	s, ctx, _, _, _, plan := inviteFixture(t)
	if _, err := s.db.Exec(ctx, "UPDATE plans SET status='confirmed', starts_at=now()+interval '30 minutes' WHERE id=$1", plan.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(ctx, `
 INSERT INTO outbox(plan_id, actor_id, kind, payload, processed_at)
 SELECT id, owner_id, 'plan.reminder', jsonb_build_object('version', version, 'starts_at', starts_at), now()
 FROM plans WHERE id=$1`, plan.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.QueuePlanReminders(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE plan_id=$1 AND kind='plan.reminder'", plan.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("legacy schedule reminders: %d, %v", count, err)
	}
}
