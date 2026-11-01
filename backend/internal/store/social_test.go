package store

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func TestSocialOwnershipAndIdempotency(t *testing.T) {
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
	a := "00000000-0000-4000-8000-000000000201"
	b := "00000000-0000-4000-8000-000000000202"
	c := "00000000-0000-4000-8000-000000000203"
	for _, id := range []string{a, b, c} {
		if _, err = s.EnsureProfile(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = s.db.Exec(ctx, "DELETE FROM groups WHERE owner_id=$1", a)
	_, _ = s.db.Exec(ctx, "DELETE FROM friendships WHERE low_id=$1 OR high_id=$1", a)
	_, _ = s.db.Exec(ctx, "DELETE FROM idempotency_keys WHERE actor_id IN ($1,$2,$3)", a, b, c)
	defer s.db.Exec(ctx, "DELETE FROM groups WHERE owner_id IN ($1,$2)", a, b)
	defer s.db.Exec(ctx, "DELETE FROM friendships WHERE low_id=$1 OR high_id=$1", a)
	defer s.db.Exec(ctx, "DELETE FROM idempotency_keys WHERE actor_id IN ($1,$2,$3)", a, b, c)
	if _, err = s.RequestFriend(ctx, a, a, "self"); !errors.Is(err, plans.ErrInvalid) {
		t.Fatal("self friendship", err)
	}
	request, err := s.RequestFriend(ctx, a, b, "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AcceptFriend(ctx, a, request.ID); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("sender accepted own request", err)
	}
	if _, err = s.AcceptFriend(ctx, c, request.ID); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("stranger accepted request", err)
	}
	if _, err = s.AcceptFriend(ctx, b, request.ID); err != nil {
		t.Fatal(err)
	}
	g, err := s.CreateGroup(ctx, a, "Hội cà phê", "group-1")
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.CreateGroup(ctx, a, "Hội cà phê", "group-1")
	if err != nil || replay.ID != g.ID {
		t.Fatal("group duplicated on retry", err)
	}
	if _, err = s.CreateGroup(ctx, a, "Different", "group-1"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatal("key reused for different body", err)
	}
	if err = s.AddGroupMember(ctx, g.ID, b, c); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("member managed group", err)
	}
	if err = s.AddGroupMember(ctx, g.ID, a, c); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("added non-friend", err)
	}
	if err = s.AddGroupMember(ctx, g.ID, a, b); err != nil {
		t.Fatal(err)
	}
	if err = s.AddGroupMember(ctx, g.ID, a, b); err != nil {
		t.Fatal("duplicate add not idempotent", err)
	}
	if _, err = s.GetGroup(ctx, g.ID, c); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("private group leaked", err)
	}
	if err = s.RemoveGroupMember(ctx, g.ID, a, a); !errors.Is(err, plans.ErrConflict) {
		t.Fatal("owner left without transfer", err)
	}
	if err = s.TransferGroup(ctx, g.ID, a, b); err != nil {
		t.Fatal(err)
	}
	if err = s.RemoveGroupMember(ctx, g.ID, a, a); err != nil {
		t.Fatal("former owner could not leave", err)
	}
	if _, err = s.GetGroup(ctx, g.ID, a); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("removed member still reads group", err)
	}
}

func TestConcurrentGroupCreationWithSameKey(t *testing.T) {
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
	actor := "00000000-0000-4000-8000-000000000204"
	_, err = s.EnsureProfile(ctx, actor)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.db.Exec(ctx, "DELETE FROM groups WHERE owner_id=$1", actor)
	_, _ = s.db.Exec(ctx, "DELETE FROM idempotency_keys WHERE actor_id=$1", actor)
	defer s.db.Exec(ctx, "DELETE FROM groups WHERE owner_id=$1", actor)
	defer s.db.Exec(ctx, "DELETE FROM idempotency_keys WHERE actor_id=$1", actor)
	var wg sync.WaitGroup
	results := make(chan Group, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g, err := s.CreateGroup(ctx, actor, "Concurrent", "same-key")
			results <- g
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	first := ""
	for g := range results {
		if first == "" {
			first = g.ID
		} else if first != g.ID {
			t.Fatal("duplicate groups")
		}
	}
	var count int
	if err = s.db.QueryRow(ctx, "SELECT count(*) FROM groups WHERE owner_id=$1", actor).Scan(&count); err != nil || count != 1 {
		t.Fatal("creation not atomic", count, err)
	}
}
