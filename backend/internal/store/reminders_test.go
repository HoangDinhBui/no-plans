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
	if _, err := s.db.Exec(ctx, "UPDATE plans SET starts_at=starts_at+interval '1 hour', version=version+1 WHERE id=$1", plan.ID); err != nil {
		t.Fatal(err)
	}
	if current, err := s.CurrentPlanReminder(ctx, delivery); err != nil || current {
		t.Fatalf("rescheduled reminder: %v %v", current, err)
	}
}
