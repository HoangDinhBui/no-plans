CREATE TABLE identity_deletions (
 user_id uuid PRIMARY KEY REFERENCES users(id),
 requested_at timestamptz NOT NULL DEFAULT now(),
 completed_at timestamptz
);
REVOKE ALL ON identity_deletions FROM PUBLIC;

INSERT INTO identity_deletions (user_id, requested_at)
 SELECT id, deleted_at FROM users WHERE deleted_at IS NOT NULL;

ALTER TABLE identity_deletions
 ADD COLUMN available_at timestamptz NOT NULL DEFAULT now(),
 ADD COLUMN attempt bigint NOT NULL DEFAULT 0;

CREATE INDEX identity_deletions_pending ON identity_deletions (available_at)
 WHERE completed_at IS NULL;
