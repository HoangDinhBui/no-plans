CREATE TABLE devices (
 id uuid PRIMARY KEY,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 platform text NOT NULL CHECK (platform IN ('ios', 'android')),
 push_token text NOT NULL UNIQUE CHECK (char_length(push_token) BETWEEN 1 AND 256),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX devices_user_id ON devices(user_id);
REVOKE ALL ON devices FROM PUBLIC;
