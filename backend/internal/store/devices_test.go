package store

import (
	"errors"
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func TestDeviceOwnershipAndRevocation(t *testing.T) {
	s, ctx, owner, other, _, _ := inviteFixture(t)
	id := ids.New()
	token := "ExpoPushToken[example_token]"
	if err := s.RegisterDevice(ctx, owner, id, "android", token); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterDevice(ctx, other, id, "ios", "ExpoPushToken[other]"); !errors.Is(err, plans.ErrForbidden) {
		t.Fatalf("foreign update: %v", err)
	}
	if err := s.RegisterDevice(ctx, other, ids.New(), "ios", token); !errors.Is(err, plans.ErrConflict) {
		t.Fatalf("duplicate token: %v", err)
	}
	if err := s.RevokeDevice(ctx, other, id); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterDevice(ctx, other, id, "ios", token); !errors.Is(err, plans.ErrForbidden) {
		t.Fatalf("foreign deletion removed registration: %v", err)
	}
	if err := s.RevokeDevice(ctx, owner, id); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterDevice(ctx, other, id, "ios", token); err != nil {
		t.Fatalf("registration after revocation: %v", err)
	}
}
