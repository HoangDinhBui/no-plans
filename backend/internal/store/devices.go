package store

import (
	"context"
	"errors"
	"regexp"

	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
	"github.com/jackc/pgx/v5/pgconn"
)

var expoToken = regexp.MustCompile(`^(ExpoPushToken|ExponentPushToken)\[[A-Za-z0-9_-]+\]$`)

func (s *Store) RegisterDevice(ctx context.Context, actor, id, platform, token string) error {
	if (platform != "ios" && platform != "android") || len(token) > 256 || !expoToken.MatchString(token) {
		return plans.ErrInvalid
	}
	tag, err := s.db.Exec(ctx, `
		INSERT INTO devices (id, user_id, platform, push_token)
		SELECT $1, id, $3, $4 FROM users WHERE id = $2 AND deleted_at IS NULL
		ON CONFLICT (id) DO UPDATE SET platform = excluded.platform,
			push_token = excluded.push_token, updated_at = now()
		WHERE devices.user_id = excluded.user_id`, id, actor, platform, token)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return plans.ErrConflict
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return plans.ErrForbidden
	}
	return nil
}

func (s *Store) RevokeDevice(ctx context.Context, actor, id string) error {
	_, err := s.db.Exec(ctx, "DELETE FROM devices WHERE id = $1 AND user_id = $2", id, actor)
	return err
}
