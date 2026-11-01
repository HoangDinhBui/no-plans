package store

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

const owner = "00000000-0000-4000-8000-000000000001"
const friend = "00000000-0000-4000-8000-000000000002"
const other = "00000000-0000-4000-8000-000000000003"
const planID = "00000000-0000-4000-8000-000000000010"

func TestResponseRechecksTimeAfterLockWait(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	s, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	id := "00000000-0000-4000-8000-000000000011"
	if _, err = s.db.Exec(ctx, "INSERT INTO users(id,display_name) VALUES($1,'Test') ON CONFLICT DO NOTHING", owner); err != nil {
		t.Fatal(err)
	}
	_, _ = s.db.Exec(ctx, "DELETE FROM plans WHERE id=$1", id)
	defer s.db.Exec(ctx, "DELETE FROM plans WHERE id=$1", id)
	captured := time.Now()
	starts := captured.Add(350 * time.Millisecond)
	p, err := plans.New(id, owner, "Time check", "coffee", starts, 2, captured)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CreatePlan(ctx, p); err != nil {
		t.Fatal(err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT id FROM plans WHERE id=$1 FOR UPDATE", id); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := s.Respond(ctx, id, owner, "maybe"); done <- err }()
	<-time.After(time.Until(starts.Add(30 * time.Millisecond)))
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, plans.ErrClosed) {
			t.Fatalf("RSVP allowed after start following lock wait: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RSVP stuck")
	}
}

func TestPostgresLastSeatAndOutbox(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required: dedicated disposable PostgreSQL database")
	}
	ctx := context.Background()
	s, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	// Integration tests only run against a dedicated test database; rollback fixture cleanup.
	for _, id := range []string{owner, friend, other} {
		if _, err = s.db.Exec(ctx, "INSERT INTO users(id,display_name) VALUES($1,'Test') ON CONFLICT DO NOTHING", id); err != nil {
			t.Fatal(err)
		}
	}
	_, err = s.db.Exec(ctx, "DELETE FROM plans WHERE id=$1", planID)
	if err != nil {
		t.Fatal(err)
	}
	defer s.db.Exec(ctx, "DELETE FROM plans WHERE id=$1", planID)
	p, err := plans.New(planID, owner, "Last seat", "coffee", time.Now().Add(time.Hour), 2, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	p.Members[friend] = "invited"
	p.Members[other] = "invited"
	if err = s.CreatePlan(ctx, p); err != nil {
		t.Fatal(err)
	}
	// Open a second pool so the contention cannot be hidden by process-local locking.
	second, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	var wg sync.WaitGroup
	out := make(chan error, 2)
	for i, id := range []string{friend, other} {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			db := s
			if i == 1 {
				db = second
			}
			_, err := db.Respond(ctx, planID, id, "going")
			out <- err
		}(i, id)
	}
	wg.Wait()
	close(out)
	success, full := 0, 0
	for err := range out {
		if err == nil {
			success++
		} else if errors.Is(err, plans.ErrFull) {
			full++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || full != 1 {
		t.Fatalf("success=%d full=%d", success, full)
	}
	persisted, err := second.GetPlan(ctx, planID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Going() != 2 {
		t.Fatalf("going=%d", persisted.Going())
	}
	winner := friend
	if persisted.Members[other] == "going" {
		winner = other
	}
	var before, after int
	if err = s.db.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE plan_id=$1", planID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before != 2 {
		t.Fatalf("expected create + one response event, got %d", before)
	}
	if changed, err := s.Respond(ctx, planID, winner, "going"); err != nil || changed {
		t.Fatal("replay changed state", err)
	}
	if _, err = s.GetPlan(ctx, planID, "00000000-0000-4000-8000-000000000099"); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("private plan leaked", err)
	}
	if _, err = s.Respond(ctx, planID, owner, "invalid"); !errors.Is(err, plans.ErrInvalid) {
		t.Fatal(err)
	}
	if err = s.db.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE plan_id=$1", planID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("duplicate/failed mutation emitted event")
	}
	if err = s.EditPlan(ctx, planID, owner, persisted.Version, "Updated", persisted.StartsAt, 2, "Cafe"); err != nil {
		t.Fatal(err)
	}
	if err = s.EditPlan(ctx, planID, owner, persisted.Version, "Stale", persisted.StartsAt, 2, "Cafe"); !errors.Is(err, plans.ErrConflict) {
		t.Fatal("stale edit accepted", err)
	}
	if err = s.TransitionPlan(ctx, planID, friend, "cancel"); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("non-owner transition accepted", err)
	}
	if err = s.TransitionPlan(ctx, planID, owner, "confirm"); err != nil {
		t.Fatal(err)
	}
	confirmed, err := second.GetPlan(ctx, planID, owner)
	if err != nil || confirmed.Status != "confirmed" || confirmed.Location != "Cafe" {
		t.Fatalf("not persisted: %+v %v", confirmed, err)
	}
	// Force an outbox failure after the member UPDATE to prove transaction atomicity.
	_, err = s.db.Exec(ctx, `CREATE OR REPLACE FUNCTION test_reject_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test outbox failure'; END $$;
	 CREATE TRIGGER test_reject_outbox BEFORE INSERT ON outbox FOR EACH ROW EXECUTE FUNCTION test_reject_outbox()`)
	if err != nil {
		t.Fatal(err)
	}
	defer s.db.Exec(ctx, "DROP TRIGGER IF EXISTS test_reject_outbox ON outbox; DROP FUNCTION IF EXISTS test_reject_outbox()")
	if _, err = s.Respond(ctx, planID, owner, "not_going"); err == nil {
		t.Fatal("outbox failure did not abort mutation")
	}
	rolledBack, err := second.GetPlan(ctx, planID, owner)
	if err != nil || rolledBack.Members[owner] != "going" || rolledBack.Version != confirmed.Version {
		t.Fatalf("failed outbox partially committed: %+v %v", rolledBack, err)
	}
}
