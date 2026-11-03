package store

import "context"

type IdentityDeletion struct {
	UserID  string
	Attempt int64
}

func (s *Store) ClaimIdentityDeletion(ctx context.Context) (IdentityDeletion, error) {
	var job IdentityDeletion
	err := s.db.QueryRow(ctx, `
 WITH candidate AS (
  SELECT user_id FROM identity_deletions
  WHERE completed_at IS NULL AND available_at <= now()
  ORDER BY available_at, user_id
  LIMIT 1 FOR UPDATE SKIP LOCKED
 )
 UPDATE identity_deletions d
 SET available_at = now() + interval '1 minute', attempt = attempt + 1
 FROM candidate c WHERE c.user_id = d.user_id
 RETURNING d.user_id::text, d.attempt
 `).Scan(&job.UserID, &job.Attempt)
	return job, err
}

func (s *Store) CompleteIdentityDeletion(ctx context.Context, job IdentityDeletion) error {
	_, err := s.db.Exec(ctx, `
 UPDATE identity_deletions SET completed_at = now()
 WHERE user_id = $1 AND attempt = $2 AND completed_at IS NULL
 `, job.UserID, job.Attempt)
	return err
}
