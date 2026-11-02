package store

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

func inviteFixture(t *testing.T) (*Store, context.Context, string, string, string, *plans.Plan) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	s, err := openTestStore(t, ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	a, b, c := ids.New(), ids.New(), ids.New()
	for _, id := range []string{a, b, c} {
		if _, err = s.EnsureProfile(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	p, err := s.CreatePlanFor(ctx, a, "plan", CreatePlanInput{Title: "Private dinner", Activity: "food", StartsAt: time.Now().Add(48 * time.Hour), Location: "Private address"})
	if err != nil {
		t.Fatal(err)
	}
	return s, ctx, a, b, c, p
}

func TestInviteLinkHashOnlyReplayAndPrivacy(t *testing.T) {
	s, ctx, a, b, _, p := inviteFixture(t)
	secret := bytes.Repeat([]byte{7}, 32)
	if _, err := s.CreateInviteLink(ctx, p.ID, b, "bad", 2, secret); !errors.Is(err, plans.ErrForbidden) {
		t.Fatal("non-owner created link", err)
	}
	link, err := s.CreateInviteLink(ctx, p.ID, a, "link", 2, secret)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.CreateInviteLink(ctx, p.ID, a, "link", 2, secret)
	if err != nil || replay.ID != link.ID || replay.Token != link.Token {
		t.Fatal("link creation not idempotent", err)
	}
	if _, err = s.CreateInviteLink(ctx, p.ID, a, "link", 3, secret); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatal("key body conflict accepted", err)
	}
	if _, err = s.CreateInviteLink(ctx, p.ID, a, "link", 2, bytes.Repeat([]byte{8}, 32)); !errors.Is(err, ErrInviteKeyChanged) {
		t.Fatal("secret rotation returned broken replay token", err)
	}
	preview, err := s.PreviewInvite(ctx, link.Token)
	if err != nil || preview.Title != "Private dinner" {
		t.Fatal("preview failed", err)
	}
	var rawStored bool
	err = s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM idempotency_keys WHERE response::text LIKE '%'||$1::text||'%') OR EXISTS(SELECT 1 FROM outbox WHERE payload::text LIKE '%'||$1::text||'%')", link.Token).Scan(&rawStored)
	if err != nil || rawStored {
		t.Fatal("raw token stored in replay/outbox", err)
	}
	var digest []byte
	if err = s.db.QueryRow(ctx, "SELECT token_hash FROM invite_links WHERE id=$1", link.ID).Scan(&digest); err != nil || len(digest) != 32 {
		t.Fatal("missing SHA256 hash", err)
	}
	if _, err = s.PreviewInvite(ctx, link.Token+"changed"); !errors.Is(err, ErrInviteUnavailable) {
		t.Fatal("tampered token accepted", err)
	}
	if err = s.RevokeInviteLink(ctx, link.ID, a); err != nil {
		t.Fatal(err)
	}
	if _, err = s.PreviewInvite(ctx, link.Token); !errors.Is(err, ErrInviteUnavailable) {
		t.Fatal("revoked preview readable", err)
	}
	if _, err = s.RedeemInvite(ctx, link.Token, b); !errors.Is(err, ErrInviteUnavailable) {
		t.Fatal("revoked link redeemed", err)
	}
}

func TestInviteConcurrentLastUseAndRepeatRedemption(t *testing.T) {
	s, ctx, a, b, c, p := inviteFixture(t)
	link, err := s.CreateInviteLink(ctx, p.ID, a, "one-use", 1, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Open(ctx, s.db.Config().ConnConfig.ConnString())
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	type result struct {
		actor string
		err   error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i, id := range []string{b, c} {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			db := s
			if i == 1 {
				db = second
			}
			_, err := db.RedeemInvite(ctx, link.Token, id)
			results <- result{actor: id, err: err}
		}(i, id)
	}
	wg.Wait()
	close(results)
	winner := ""
	unavailable := 0
	for result := range results {
		if result.err == nil {
			winner = result.actor
		} else if errors.Is(result.err, ErrInviteUnavailable) {
			unavailable++
		} else {
			t.Fatal(result.err)
		}
	}
	if winner == "" || unavailable != 1 {
		t.Fatal("last link use not exclusive")
	}
	if id, err := s.RedeemInvite(ctx, link.Token, winner); err != nil || id != p.ID {
		t.Fatal("repeat redeem spent another use", err)
	}
	var used int
	if err = s.db.QueryRow(ctx, "SELECT use_count FROM invite_links WHERE id=$1", link.ID).Scan(&used); err != nil || used != 1 {
		t.Fatal("bad use count", used, err)
	}
	if err = s.RevokePlanMember(ctx, p.ID, a, winner); err != nil {
		t.Fatal(err)
	}
	fresh, err := s.CreateInviteLink(ctx, p.ID, a, "new-link", 10, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, fresh.Token, winner); !errors.Is(err, ErrInviteUnavailable) {
		t.Fatal("revoked member rejoined via another link", err)
	}
}

func TestInviteExpiryBlockAndPlanCancellation(t *testing.T) {
	s, ctx, a, b, _, p := inviteFixture(t)
	secret := bytes.Repeat([]byte{7}, 32)
	link, err := s.CreateInviteLink(ctx, p.ID, a, "expired", 10, secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(ctx, "UPDATE invite_links SET expires_at=clock_timestamp()-interval '1 second' WHERE id=$1", link.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, link.Token, b); !errors.Is(err, ErrInviteUnavailable) {
		t.Fatal("expired link accepted", err)
	}
	fresh, err := s.CreateInviteLink(ctx, p.ID, a, "active", 10, secret)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.BlockUser(ctx, b, a); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, fresh.Token, b); !errors.Is(err, ErrInviteUnavailable) {
		t.Fatal("block bypassed with link", err)
	}
	if err = s.UnblockUser(ctx, b, a); err != nil {
		t.Fatal(err)
	}
	if err = s.TransitionPlan(ctx, p.ID, a, "cancel"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.PreviewInvite(ctx, fresh.Token); !errors.Is(err, ErrInviteUnavailable) {
		t.Fatal("cancelled plan preview readable", err)
	}
}
