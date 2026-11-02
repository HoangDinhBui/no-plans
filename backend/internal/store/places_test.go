package store

import (
	"errors"
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func TestCatalogueFiltersAndOptionSnapshot(t *testing.T) {
	s, ctx, a, _, _, p := inviteFixture(t)
	low, high := 30000, 70000
	place := Place{ID: ids.New(), Name: "Ý tưởng cà phê", Activity: "coffee", Kind: "idea", Area: "Trung tâm", BudgetMin: &low, BudgetMax: &high, PriceNote: "Ngân sách gợi ý, chưa phải báo giá quán", PeopleMin: 1, PeopleMax: 4, Active: true}
	if err := s.UpsertPlaces(ctx, []Place{place}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertPlaces(ctx, []Place{place}); err != nil {
		t.Fatal(err)
	}
	page, err := s.ListPlaces(ctx, PlaceFilter{Activity: "coffee", Area: "trung", MaxBudget: &high, People: 2, Limit: 20})
	if err != nil || len(page.Items) != 1 {
		t.Fatal("matching catalogue entry missing", err)
	}
	tooLow := 20000
	page, err = s.ListPlaces(ctx, PlaceFilter{MaxBudget: &tooLow, People: 2, Limit: 20})
	if err != nil || len(page.Items) != 0 {
		t.Fatal("over-budget idea returned", err)
	}
	page, err = s.ListPlaces(ctx, PlaceFilter{People: 5, Limit: 20})
	if err != nil || len(page.Items) != 0 {
		t.Fatal("group size filter ignored", err)
	}
	if _, err = s.ListPlaces(ctx, PlaceFilter{Activity: "invalid", People: 1, Limit: 20}); !errors.Is(err, plans.ErrInvalid) {
		t.Fatal("invalid activity accepted", err)
	}
	option, err := s.AddOption(ctx, p.ID, a, "catalogue", OptionInput{PlaceID: place.ID})
	if err != nil {
		t.Fatal(err)
	}
	place.Name = "Changed catalogue name"
	if err = s.UpsertPlaces(ctx, []Place{place}); err != nil {
		t.Fatal(err)
	}
	options, err := s.ListOptions(ctx, p.ID, a, "", 20)
	if err != nil || len(options.Items) != 1 || options.Items[0].ID != option.ID || options.Items[0].Name != "Ý tưởng cà phê" {
		t.Fatal("catalogue edit rewrote plan option snapshot", err)
	}
	bad := place
	bad.Kind = "place"
	bad.SourceURL = "javascript:alert(1)"
	if err = s.UpsertPlaces(ctx, []Place{bad}); !errors.Is(err, plans.ErrInvalid) {
		t.Fatal("unsafe catalogue source accepted", err)
	}
}
