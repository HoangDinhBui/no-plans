package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func TestProfileProvisioningPreservesEditsAndDeletion(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	s, err := openTestStore(t, ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	id := "00000000-0000-4000-8000-000000000101"
	_, _ = s.db.Exec(ctx, "DELETE FROM users WHERE id=$1", id)
	defer s.db.Exec(ctx, "DELETE FROM users WHERE id=$1", id)
	p, err := s.EnsureProfile(ctx, id)
	if err != nil || p.ID != id || p.DisplayName == "" {
		t.Fatalf("profile: %+v %v", p, err)
	}
	edited, err := s.UpdateProfile(ctx, id, "  Đình Bùi  ", "https://example.com/avatar.png")
	if err != nil || edited.DisplayName != "Đình Bùi" {
		t.Fatalf("edit: %+v %v", edited, err)
	}
	for i := 0; i < 2; i++ {
		p, err = s.EnsureProfile(ctx, id)
		if err != nil || p.DisplayName != "Đình Bùi" {
			t.Fatal("provision overwrote name", err)
		}
	}
	for _, tc := range []struct{ name, avatar string }{{" ", ""}, {"Name", "javascript:alert(1)"}, {"Name", "https://user:pass@example.com/a"}} {
		if _, err = s.UpdateProfile(ctx, id, tc.name, tc.avatar); !errors.Is(err, plans.ErrInvalid) {
			t.Fatal("invalid profile accepted", err)
		}
	}
	_, err = s.db.Exec(ctx, "UPDATE users SET deleted_at=now() WHERE id=$1", id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.EnsureProfile(ctx, id); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("deleted account resurrected", err)
	}
}
