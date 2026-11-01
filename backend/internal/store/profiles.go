package store

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/jackc/pgx/v5"
)

type Profile struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

func (s *Store) EnsureProfile(ctx context.Context, id string) (Profile, error) {
	// Separate statements see the committed winner of a concurrent first login.
	if _, err := s.db.Exec(ctx, "INSERT INTO users(id,display_name) VALUES($1,'Bạn mới') ON CONFLICT DO NOTHING", id); err != nil {
		return Profile{}, err
	}
	var p Profile
	err := s.db.QueryRow(ctx, "SELECT id::text,display_name,avatar_url FROM users WHERE id=$1 AND deleted_at IS NULL", id).Scan(&p.ID, &p.DisplayName, &p.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, plans.ErrForbidden
	}
	return p, err
}

func (s *Store) UpdateProfile(ctx context.Context, id, name, avatar string) (Profile, error) {
	return s.PatchProfile(ctx, id, &name, &avatar)
}

func (s *Store) PatchProfile(ctx context.Context, id string, name, avatar *string) (Profile, error) {
	if name != nil {
		value := strings.TrimSpace(*name)
		if value == "" || utf8.RuneCountInString(value) > 80 {
			return Profile{}, plans.ErrInvalid
		}
		name = &value
	}
	if avatar != nil {
		value := strings.TrimSpace(*avatar)
		if len(value) > 2048 {
			return Profile{}, plans.ErrInvalid
		}
		avatar = &value
		if value != "" {
			u, err := url.Parse(value)
			if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
				return Profile{}, plans.ErrInvalid
			}
		}
	}
	var p Profile
	err := s.db.QueryRow(ctx, "UPDATE users SET display_name=COALESCE($2,display_name),avatar_url=COALESCE($3,avatar_url) WHERE id=$1 AND deleted_at IS NULL RETURNING id::text,display_name,avatar_url", id, name, avatar).Scan(&p.ID, &p.DisplayName, &p.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, plans.ErrForbidden
	}
	return p, err
}
