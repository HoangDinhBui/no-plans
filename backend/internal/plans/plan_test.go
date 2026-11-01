package plans

import (
	"errors"
	"testing"
	"time"
)

var clock = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)

func fixture(t *testing.T) *Plan {
	t.Helper()
	p, err := New("plan", "owner", "Cà phê tối nay", "coffee", clock.Add(time.Hour), 2, clock)
	if err != nil {
		t.Fatal(err)
	}
	p.Members["friend"] = "invited"
	p.Members["other"] = "invited"
	return p
}

func TestOwnerCountsTowardCapacity(t *testing.T) {
	p := fixture(t)
	if changed, err := p.Respond("friend", "going", clock); err != nil || !changed {
		t.Fatalf("first response: %v %v", changed, err)
	}
	if _, err := p.Respond("other", "going", clock); !errors.Is(err, ErrFull) {
		t.Fatalf("overbooked: %v", err)
	}
	if p.Members["other"] != "invited" {
		t.Fatal("failed mutation changed membership")
	}
	version := p.Version
	if changed, err := p.Respond("friend", "going", clock); err != nil || changed || p.Version != version {
		t.Fatal("duplicate RSVP not idempotent")
	}
	if _, err := p.Respond("friend", "not_going", clock); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Respond("other", "going", clock); err != nil {
		t.Fatal("seat not released", err)
	}
}

func TestResponseRejectsUnauthorizedInvalidAndLate(t *testing.T) {
	for _, tc := range []struct {
		actor, response string
		at              time.Time
		want            error
	}{
		{"stranger", "going", clock, ErrForbidden},
		{"friend", "invented", clock, ErrInvalid},
		{"friend", "going", clock.Add(time.Hour), ErrClosed},
	} {
		p := fixture(t)
		if _, err := p.Respond(tc.actor, tc.response, tc.at); !errors.Is(err, tc.want) {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
}

func TestLifecycle(t *testing.T) {
	p := fixture(t)
	if err := p.Transition("friend", "confirm", clock); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if err := p.Transition("owner", "confirm", clock); !errors.Is(err, ErrInvalid) {
		t.Fatal("confirmed without place", err)
	}
	p.Location = "Quán cà phê đã chọn"
	if err := p.Transition("owner", "confirm", clock); err != nil || p.Status != "confirmed" {
		t.Fatal(err)
	}
	if _, err := p.Respond("friend", "maybe", clock); err != nil {
		t.Fatal("confirmed RSVP rejected", err)
	}
	if err := p.Transition("owner", "complete", clock); !errors.Is(err, ErrInvalid) {
		t.Fatal("completed too soon", err)
	}
	if err := p.Transition("owner", "complete", clock.Add(2*time.Hour)); err != nil || p.Status != "completed" {
		t.Fatal(err)
	}
	if _, err := p.Respond("friend", "going", clock); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if err := p.Transition("owner", "cancel", clock); !errors.Is(err, ErrClosed) {
		t.Fatal("terminal transition allowed", err)
	}
}

func TestEditChecksVersionCapacityAndFuture(t *testing.T) {
	p := fixture(t)
	_, _ = p.Respond("friend", "going", clock)
	v := p.Version
	if err := p.Edit("owner", v-1, "new title", clock.Add(time.Hour), 2, "Cafe", clock); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if err := p.Edit("owner", v, "new title", clock.Add(time.Hour), 1, "Cafe", clock); !errors.Is(err, ErrFull) {
		t.Fatal(err)
	}
	if err := p.Edit("owner", v, "new title", clock, 2, "Cafe", clock); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if p.Version != v || p.Title != "Cà phê tối nay" {
		t.Fatal("failed edit mutated plan")
	}
	if err := p.Edit("owner", v, "new title", clock.Add(2*time.Hour), 2, "Cafe", clock); err != nil || p.Version != v+1 {
		t.Fatal(err)
	}
}

func TestCreateValidation(t *testing.T) {
	for _, tc := range []struct {
		title    string
		capacity int
		at       time.Time
	}{{" ", 2, clock.Add(time.Hour)}, {"valid", -1, clock.Add(time.Hour)}, {"valid", 1, clock}} {
		if _, err := New("id", "owner", tc.title, "coffee", tc.at, tc.capacity, clock); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid plan accepted: %+v %v", tc, err)
		}
	}
	p, err := New("id", "owner", "valid", "coffee", clock.Add(time.Hour), 0, clock)
	if err != nil || p.Members["owner"] != "going" {
		t.Fatal("unlimited capacity or owner invalid", err)
	}
}
