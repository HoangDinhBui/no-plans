package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func TestBlockPreventsNewInvitationsAndUnblockRequiresFriendshipAgain(t *testing.T) {
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
	a, b := ids.New(), ids.New()
	for _, id := range []string{a, b} {
		if _, err = s.EnsureProfile(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	request, err := s.RequestFriend(ctx, a, b, "friend")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AcceptFriend(ctx, b, request.ID); err != nil {
		t.Fatal(err)
	}
	g, err := s.CreateGroup(ctx, a, "Blocked group", "group")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.BlockUser(ctx, b, a); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RequestFriend(ctx, a, b, "new"); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("invited across block", err)
	}
	if err = s.AddGroupMember(ctx, g.ID, a, b); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("added blocked member", err)
	}
	if err = s.UnblockUser(ctx, b, a); err != nil {
		t.Fatal(err)
	}
	if err = s.AddGroupMember(ctx, g.ID, a, b); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("unblock silently restored friendship", err)
	}
	if _, err = s.RequestFriend(ctx, a, b, "after-unblock"); err != nil {
		t.Fatal(err)
	}
}

func TestInvitationLimitsAreAtomicAndReplayDoesNotSpendQuota(t *testing.T) {
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
	a, b, c := ids.New(), ids.New(), ids.New()
	for _, id := range []string{a, b, c} {
		if _, err = s.EnsureProfile(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	original, err := s.RequestFriend(ctx, a, b, "original")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(ctx, "INSERT INTO invitation_batches(actor_id,recipient_count) VALUES($1,199)", a); err != nil {
		t.Fatal(err)
	}
	if replay, err := s.RequestFriend(ctx, a, b, "original"); err != nil || replay.ID != original.ID {
		t.Fatal("replay consumed quota", err)
	}
	if _, err = s.RequestFriend(ctx, a, c, "limited"); !errors.Is(err, ErrRateLimited) {
		t.Fatal("invitation limit bypassed", err)
	}
	var count int
	if err = s.db.QueryRow(ctx, "SELECT count(*) FROM friendships WHERE requester_id=$1", a).Scan(&count); err != nil || count != 1 {
		t.Fatal("failed quota left partial friendship", count, err)
	}
	if _, err = s.db.Exec(ctx, "INSERT INTO api_rate_buckets(actor_id,bucket,requests) VALUES($1,date_trunc('minute',clock_timestamp()),300)", a); err != nil {
		t.Fatal(err)
	}
	if err = s.CheckAPIRate(ctx, a); !errors.Is(err, ErrRateLimited) {
		t.Fatal("API rate limit bypassed", err)
	}
}
