package store

import "context"

// PurgeExpiredQuotas removes records outside every quota window.
func (s *Store) PurgeExpiredQuotas(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
 WITH expired_buckets AS (
  DELETE FROM api_rate_buckets WHERE bucket < now() - interval '1 day'
 )
 DELETE FROM invitation_batches WHERE created_at < now() - interval '1 day'
 `)
	return err
}
