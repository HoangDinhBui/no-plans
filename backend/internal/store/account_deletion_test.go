package store

import (
	"errors"
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func TestDeleteAccountRevokesDevices(t *testing.T) {
	s, ctx, actor, _, _, _ := inviteFixture(t)
	device := ids.New()
	if err := s.RegisterDevice(ctx, actor, device, "ios", "ExpoPushToken[delete]"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteAccount(ctx, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EnsureProfile(ctx, actor); !errors.Is(err, plans.ErrForbidden) {
		t.Fatalf("deleted profile accessible: %v", err)
	}
	if err := s.RegisterDevice(ctx, actor, device, "ios", "ExpoPushToken[delete]"); !errors.Is(err, plans.ErrForbidden) {
		t.Fatalf("deleted account registered a device: %v", err)
	}
	var count int
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM devices WHERE user_id=$1", actor).Scan(&count); err != nil || count != 0 {
		t.Fatalf("remaining devices: %d, %v", count, err)
	}
}
