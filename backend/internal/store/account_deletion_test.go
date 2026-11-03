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

func TestIdentityDeletionReclaim(t *testing.T) {
	s, ctx, actor, _, _, _ := inviteFixture(t)
	if err := s.DeleteAccount(ctx, actor); err != nil {
		t.Fatal(err)
	}
	first, err := s.ClaimIdentityDeletion(ctx)
	if err != nil || first.UserID != actor {
		t.Fatalf("claim: %+v %v", first, err)
	}
	if _, err := s.db.Exec(ctx, "UPDATE identity_deletions SET available_at=now() WHERE user_id=$1", actor); err != nil {
		t.Fatal(err)
	}
	second, err := s.ClaimIdentityDeletion(ctx)
	if err != nil || second.Attempt != first.Attempt+1 {
		t.Fatalf("reclaim: %+v %v", second, err)
	}
	if err := s.CompleteIdentityDeletion(ctx, first); err != nil {
		t.Fatal(err)
	}
	var completed bool
	if err := s.db.QueryRow(ctx, "SELECT completed_at IS NOT NULL FROM identity_deletions WHERE user_id=$1", actor).Scan(&completed); err != nil || completed {
		t.Fatalf("stale completion: %v %v", completed, err)
	}
	if err := s.CompleteIdentityDeletion(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(ctx, "SELECT completed_at IS NOT NULL FROM identity_deletions WHERE user_id=$1", actor).Scan(&completed); err != nil || !completed {
		t.Fatalf("completion: %v %v", completed, err)
	}
}
