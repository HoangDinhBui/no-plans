package store

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func TestVotingOwnershipCrossPlanAndLockAfterConfirmation(t *testing.T) {
	s, ctx, a, b, _, p := inviteFixture(t)
	link, err := s.CreateInviteLink(ctx, p.ID, a, "link", 10, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, link.Token, b); err != nil {
		t.Fatal(err)
	}
	input := OptionInput{Name: "Cafe A", Address: "Địa chỉ A"}
	if _, err = s.AddOption(ctx, p.ID, b, "bad", input); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("member proposed owner-only option", err)
	}
	first, err := s.AddOption(ctx, p.ID, a, "first", input)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.AddOption(ctx, p.ID, a, "first", input)
	if err != nil || replay.ID != first.ID {
		t.Fatal("option duplicated", err)
	}
	second, err := s.AddOption(ctx, p.ID, a, "second", OptionInput{Name: "Cafe B"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.CreatePlanFor(ctx, a, "other", CreatePlanInput{Title: "Other", Activity: "coffee", StartsAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := s.AddOption(ctx, other.ID, a, "foreign", input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CastVote(ctx, p.ID, b, foreign.ID); !errors.Is(err, plans.ErrInvalid) {
		t.Fatal("cross-plan vote accepted", err)
	}
	if changed, err := s.CastVote(ctx, p.ID, b, first.ID); err != nil || !changed {
		t.Fatal(err)
	}
	if changed, err := s.CastVote(ctx, p.ID, b, first.ID); err != nil || changed {
		t.Fatal("duplicate vote changed state", err)
	}
	if _, err = s.CastVote(ctx, p.ID, b, second.ID); err != nil {
		t.Fatal(err)
	}
	options, err := s.ListOptions(ctx, p.ID, b, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, option := range options.Items {
		if option.ID == first.ID && option.VoteCount != 0 {
			t.Fatal("old vote retained")
		}
		if option.ID == second.ID && (option.VoteCount != 1 || !option.MyVote) {
			t.Fatal("new vote not counted")
		}
	}
	current, err := s.GetPlan(ctx, p.ID, a)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SelectOption(ctx, p.ID, a, current.Version-1, second.ID); !errors.Is(err, plans.ErrConflict) {
		t.Fatal("stale selection accepted", err)
	}
	if err = s.SelectOption(ctx, p.ID, a, current.Version, foreign.ID); !errors.Is(err, plans.ErrInvalid) {
		t.Fatal("cross-plan selection accepted", err)
	}
	if err = s.SelectOption(ctx, p.ID, a, current.Version, second.ID); err != nil {
		t.Fatal(err)
	}
	selected, err := s.GetPlan(ctx, p.ID, a)
	if err != nil || selected.SelectedOptionID != second.ID || selected.Location != "Cafe B" {
		t.Fatalf("selected location inconsistent: %+v %v", selected, err)
	}
	if err = s.TransitionPlan(ctx, p.ID, a, "confirm"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CastVote(ctx, p.ID, b, first.ID); !errors.Is(err, plans.ErrClosed) {
		t.Fatal("vote changed after confirmation", err)
	}
	if _, err = s.AddOption(ctx, p.ID, a, "late", input); !errors.Is(err, plans.ErrClosed) {
		t.Fatal("new option after confirmation", err)
	}
	if err = s.RevokePlanMember(ctx, p.ID, a, b); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = s.db.QueryRow(ctx, "SELECT count(*) FROM votes WHERE plan_id=$1", p.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("revoked member vote remained", err)
	}
	selected, err = s.GetPlan(ctx, p.ID, a)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.EditPlan(ctx, p.ID, a, selected.Version, selected.Title, selected.StartsAt, selected.Capacity, "Manual location"); err != nil {
		t.Fatal(err)
	}
	selected, err = s.GetPlan(ctx, p.ID, a)
	if err != nil || selected.SelectedOptionID != "" {
		t.Fatal("manual location kept stale selected option", err)
	}
}
