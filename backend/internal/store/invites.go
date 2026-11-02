package store

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/jackc/pgx/v5"
)

var ErrInviteUnavailable = errors.New("invite_unavailable")
var ErrInviteKeyChanged = errors.New("invite_key_changed")
var ErrInviteNotConfigured = errors.New("invite_not_configured")

type InviteLink struct {
	ID        string     `json:"id"`
	PlanID    string     `json:"plan_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	MaxUses   int        `json:"max_uses"`
	UseCount  int        `json:"use_count"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}
type IssuedInvite struct {
	InviteLink
	Token string `json:"token"`
}
type InvitePreview struct {
	Title    string    `json:"title"`
	Activity string    `json:"activity"`
	StartsAt time.Time `json:"starts_at"`
}

func inviteToken(id string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte("no-plans:invite:v1:" + id))
	return id + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func tokenID(token string) (string, error) {
	if len(token) != 80 {
		return "", ErrInviteUnavailable
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 || !ids.Valid(parts[0]) {
		return "", ErrInviteUnavailable
	}
	digest, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(digest) != 32 {
		return "", ErrInviteUnavailable
	}
	return parts[0], nil
}

func (s *Store) CreateInviteLink(ctx context.Context, planID, actor, key string, maxUses int, secret []byte) (IssuedInvite, error) {
	if len(secret) != 32 {
		return IssuedInvite{}, ErrInviteNotConfigured
	}
	if maxUses < 1 || maxUses > 1000 {
		return IssuedInvite{}, plans.ErrInvalid
	}
	var link InviteLink
	input := struct {
		PlanID  string
		MaxUses int
	}{PlanID: planID, MaxUses: maxUses}
	err := s.idempotent(ctx, actor, "invite-link-create", key, input, func(tx pgx.Tx) (any, error) {
		p, err := load(ctx, tx, planID, actor, true)
		if err != nil {
			return nil, err
		}
		if p.OwnerID != actor {
			return nil, plans.ErrForbidden
		}
		now := time.Now().UTC()
		if (p.Status != "inviting" && p.Status != "confirmed") || !p.StartsAt.After(now) {
			return nil, plans.ErrClosed
		}
		result := InviteLink{ID: ids.New(), PlanID: planID, ExpiresAt: now.Add(7 * 24 * time.Hour), MaxUses: maxUses}
		hash := sha256.Sum256([]byte(inviteToken(result.ID, secret)))
		if _, err = tx.Exec(ctx, "INSERT INTO invite_links(id,plan_id,created_by,token_hash,expires_at,max_uses) VALUES($1,$2,$3,$4,$5,$6)", result.ID, planID, actor, hash[:], result.ExpiresAt, maxUses); err != nil {
			return nil, err
		}
		if err = audit(ctx, tx, actor, "invite_link.created", result.ID); err != nil {
			return nil, err
		}
		return result, nil
	}, &link)
	if err != nil {
		return IssuedInvite{}, err
	}
	raw := inviteToken(link.ID, secret)
	hash := sha256.Sum256([]byte(raw))
	var stored []byte
	if err = s.db.QueryRow(ctx, "SELECT token_hash FROM invite_links WHERE id=$1", link.ID).Scan(&stored); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return IssuedInvite{}, ErrInviteUnavailable
		}
		return IssuedInvite{}, err
	}
	// After secret rotation old presented tokens remain redeemable by their hash.
	// Replaying a create with the new secret must never return a nonworking token.
	if !hmac.Equal(hash[:], stored) {
		return IssuedInvite{}, ErrInviteKeyChanged
	}
	return IssuedInvite{InviteLink: link, Token: raw}, nil
}

func (s *Store) PreviewInvite(ctx context.Context, token string) (InvitePreview, error) {
	id, err := tokenID(token)
	if err != nil {
		return InvitePreview{}, err
	}
	hash := sha256.Sum256([]byte(token))
	var preview InvitePreview
	err = s.db.QueryRow(ctx, `SELECT p.title,p.activity,p.starts_at FROM invite_links l JOIN plans p ON p.id=l.plan_id WHERE l.id=$1 AND l.token_hash=$2 AND l.revoked_at IS NULL AND l.expires_at>clock_timestamp() AND l.use_count<l.max_uses AND p.status IN ('inviting','confirmed') AND p.starts_at>clock_timestamp()`, id, hash[:]).Scan(&preview.Title, &preview.Activity, &preview.StartsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return preview, ErrInviteUnavailable
	}
	return preview, err
}

func (s *Store) RedeemInvite(ctx context.Context, token, actor string) (string, error) {
	id, err := tokenID(token)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(token))
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var planID string
	if err = tx.QueryRow(ctx, "SELECT plan_id::text FROM invite_links WHERE id=$1 AND token_hash=$2", id, hash[:]).Scan(&planID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrInviteUnavailable
		}
		return "", err
	}
	// All mutations use plan -> link -> relationship lock order, including revoke.
	var owner, status string
	var startsAt time.Time
	var version int64
	if err = tx.QueryRow(ctx, "SELECT owner_id::text,status,starts_at,version FROM plans WHERE id=$1 FOR UPDATE", planID).Scan(&owner, &status, &startsAt, &version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrInviteUnavailable
		}
		return "", err
	}
	var stored []byte
	var expires time.Time
	var maximum, used int
	var revoked *time.Time
	if err = tx.QueryRow(ctx, "SELECT token_hash,expires_at,max_uses,use_count,revoked_at FROM invite_links WHERE id=$1 FOR UPDATE", id).Scan(&stored, &expires, &maximum, &used, &revoked); err != nil {
		return "", err
	}
	if err = lockPair(ctx, tx, owner, actor); err != nil {
		return "", err
	}
	if err = canConnect(ctx, tx, actor, owner); err != nil {
		if errors.Is(err, plans.ErrForbidden) {
			return "", ErrInviteUnavailable
		}
		return "", err
	}
	now := time.Now()
	if !bytes.Equal(stored, hash[:]) || revoked != nil || !expires.After(now) || (status != "inviting" && status != "confirmed") || !startsAt.After(now) {
		return "", ErrInviteUnavailable
	}
	var member, wasRemoved bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM plan_members WHERE plan_id=$1 AND user_id=$2),EXISTS(SELECT 1 FROM plan_member_revocations WHERE plan_id=$1 AND user_id=$2)", planID, actor).Scan(&member, &wasRemoved); err != nil {
		return "", err
	}
	if wasRemoved {
		return "", ErrInviteUnavailable
	}
	if member {
		return planID, nil
	}
	if used >= maximum {
		return "", ErrInviteUnavailable
	}
	if _, err = tx.Exec(ctx, "INSERT INTO plan_members(plan_id,user_id,response) VALUES($1,$2,'invited')", planID, actor); err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, "UPDATE invite_links SET use_count=use_count+1 WHERE id=$1", id); err != nil {
		return "", err
	}
	version++
	if _, err = tx.Exec(ctx, "UPDATE plans SET version=$2 WHERE id=$1", planID, version); err != nil {
		return "", err
	}
	if err = event(ctx, tx, planID, actor, "plan.joined", version); err != nil {
		return "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return planID, nil
}

func (s *Store) RevokeInviteLink(ctx context.Context, id, actor string) error {
	if !ids.Valid(id) {
		return plans.ErrInvalid
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var planID string
	if err = tx.QueryRow(ctx, "SELECT plan_id::text FROM invite_links WHERE id=$1", id).Scan(&planID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return plans.ErrForbidden
		}
		return err
	}
	p, err := load(ctx, tx, planID, actor, true)
	if err != nil {
		return err
	}
	if p.OwnerID != actor {
		return plans.ErrForbidden
	}
	tag, err := tx.Exec(ctx, "UPDATE invite_links SET revoked_at=clock_timestamp() WHERE id=$1 AND revoked_at IS NULL", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		if err = audit(ctx, tx, actor, "invite_link.revoked", id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ListInviteLinks(ctx context.Context, planID, actor, cursor string, limit int) (Page[InviteLink], error) {
	page := Page[InviteLink]{Items: []InviteLink{}}
	if !validPage(cursor, limit) {
		return page, plans.ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	p, err := load(ctx, tx, planID, actor, false)
	if err != nil {
		return page, err
	}
	if p.OwnerID != actor {
		return page, plans.ErrForbidden
	}
	rows, err := tx.Query(ctx, "SELECT id::text,plan_id::text,expires_at,max_uses,use_count,revoked_at FROM invite_links WHERE plan_id=$1 AND ($2='' OR id>NULLIF($2,'')::uuid) ORDER BY id LIMIT $3", planID, strings.ToLower(cursor), limit+1)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var l InviteLink
		if err = rows.Scan(&l.ID, &l.PlanID, &l.ExpiresAt, &l.MaxUses, &l.UseCount, &l.RevokedAt); err != nil {
			return page, err
		}
		page.Items = append(page.Items, l)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.NextCursor = page.Items[limit-1].ID
	}
	return page, tx.Commit(ctx)
}
