package store

import (
	"encoding/json"
	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"testing"
)

func TestIdeaCanReceiveVotesButCannotBecomeMeetingLocation(t *testing.T) {
	s, ctx, a, _, _, p := inviteFixture(t)
	idea := Place{ID: ids.New(), Name: "Đi ăn bánh mì", Activity: "food", Kind: "idea", Area: "TP.HCM", PeopleMin: 1, Active: true}
	if err := s.UpsertPlaces(ctx, []Place{idea}); err != nil {
		t.Fatal(err)
	}
	option, err := s.AddOption(ctx, p.ID, a, "idea", OptionInput{PlaceID: idea.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CastVote(ctx, p.ID, a, option.ID); err != nil {
		t.Fatal(err)
	}
	current, err := s.GetPlan(ctx, p.ID, a)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SelectOption(ctx, p.ID, a, current.Version, option.ID); err == nil {
		t.Fatal("activity idea became a meeting location")
	}
	after, err := s.GetPlan(ctx, p.ID, a)
	if err != nil || after.Version != current.Version || after.Location != current.Location {
		t.Fatal("failed selection mutated plan", err)
	}
	idea.Kind = "place"
	idea.SourceURL = "https://example.com/venue"
	idea.SourceCheckedAt = &p.StartsAt
	if err = s.UpsertPlaces(ctx, []Place{idea}); err != nil {
		t.Fatal(err)
	}
	page, err := s.ListOptions(ctx, p.ID, a, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(page.Items[0])
	var fields map[string]any
	_ = json.Unmarshal(raw, &fields)
	if fields["kind"] != "idea" {
		t.Fatal("kind not snapshotted", fields)
	}
}

func TestLegacyIdeaSelectionMustBeReplacedBeforeConfirmation(t *testing.T) {
	s, ctx, a, _, _, p := inviteFixture(t)
	idea := Place{ID: ids.New(), Name: "Đi dạo", Activity: "other", Kind: "idea", Area: "TP.HCM", PeopleMin: 1, Active: true}
	if err := s.UpsertPlaces(ctx, []Place{idea}); err != nil {
		t.Fatal(err)
	}
	option, err := s.AddOption(ctx, p.ID, a, "idea", OptionInput{PlaceID: idea.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(ctx, "UPDATE plans SET selected_option_id=$2,location=$3 WHERE id=$1", p.ID, option.ID, idea.Name); err != nil {
		t.Fatal(err)
	}
	if err = s.TransitionPlan(ctx, p.ID, a, "confirm"); err == nil {
		t.Fatal("legacy idea-only selection confirmed")
	}
	current, err := s.GetPlan(ctx, p.ID, a)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.EditPlan(ctx, p.ID, a, current.Version, current.Title, current.StartsAt, current.Capacity, "Cổng chính công viên"); err != nil {
		t.Fatal(err)
	}
	if err = s.TransitionPlan(ctx, p.ID, a, "confirm"); err != nil {
		t.Fatal(err)
	}
}

func TestConfirmedLegacyIdeaAllowsUnrelatedEdits(t *testing.T) {
	s, ctx, a, _, _, p := inviteFixture(t)
	idea := Place{ID: ids.New(), Name: "Ý tưởng cũ", Activity: "other", Kind: "idea", Area: "TP.HCM", PeopleMin: 1, Active: true}
	if err := s.UpsertPlaces(ctx, []Place{idea}); err != nil {
		t.Fatal(err)
	}
	option, err := s.AddOption(ctx, p.ID, a, "old-idea", OptionInput{PlaceID: idea.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(ctx, "UPDATE plans SET selected_option_id=$2,location=$3,status='confirmed' WHERE id=$1", p.ID, option.ID, idea.Name); err != nil {
		t.Fatal(err)
	}
	current, err := s.GetPlan(ctx, p.ID, a)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.EditPlan(ctx, p.ID, a, current.Version, "Tên mới", current.StartsAt, current.Capacity, current.Location); err != nil {
		t.Fatal("unrelated legacy edit blocked", err)
	}
	updated, err := s.GetPlan(ctx, p.ID, a)
	if err != nil || updated.Title != "Tên mới" || updated.Location != idea.Name || updated.SelectedOptionID != option.ID || updated.Status != "confirmed" {
		t.Fatal("legacy plan changed unexpectedly", err)
	}
}
