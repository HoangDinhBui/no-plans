package store

import (
	"errors"
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func TestCreateReportAuthorizationAndLimit(t *testing.T) {
	s, ctx, actor, target, _, plan := inviteFixture(t)
	if _, err := s.CreateReport(ctx, actor, actor, "", "spam"); !errors.Is(err, plans.ErrForbidden) {
		t.Fatalf("self report: %v", err)
	}
	if _, err := s.CreateReport(ctx, actor, target, plan.ID, "spam"); !errors.Is(err, plans.ErrForbidden) {
		t.Fatalf("report outside shared plan: %v", err)
	}
	for range 10 {
		if _, err := s.CreateReport(ctx, actor, target, "", "spam"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.CreateReport(ctx, actor, target, "", "spam"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("report limit: %v", err)
	}
}
