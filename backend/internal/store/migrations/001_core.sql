CREATE TABLE users (
 id uuid PRIMARY KEY,
 display_name text NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 80),
 avatar_url text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz
);
CREATE TABLE groups (
 id uuid PRIMARY KEY,
 owner_id uuid NOT NULL REFERENCES users(id),
 name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 80),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE group_members (
 group_id uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id),
 muted boolean NOT NULL DEFAULT false,
 PRIMARY KEY (group_id,user_id)
);
CREATE TABLE plans (
 id uuid PRIMARY KEY,
 owner_id uuid NOT NULL REFERENCES users(id),
 group_id uuid REFERENCES groups(id),
 title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
 activity text NOT NULL CHECK (activity IN ('food','coffee','drinks','cinema','sports','other')),
 starts_at timestamptz NOT NULL,
 capacity integer NOT NULL DEFAULT 0 CHECK (capacity>=0),
 status text NOT NULL CHECK (status IN ('inviting','confirmed','cancelled','completed')),
 location text NOT NULL DEFAULT '' CHECK (char_length(location)<=500),
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK (status NOT IN ('confirmed','completed') OR char_length(trim(location))>0)
);
CREATE TABLE plan_members (
 plan_id uuid NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id),
 response text NOT NULL CHECK (response IN ('invited','going','not_going','maybe')),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (plan_id,user_id)
);
CREATE INDEX plan_members_user ON plan_members(user_id,plan_id);
CREATE INDEX plans_upcoming ON plans(starts_at,id);
CREATE TABLE outbox (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 plan_id uuid REFERENCES plans(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id),
 kind text NOT NULL,
 payload jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 available_at timestamptz NOT NULL DEFAULT now(),
 lease_until timestamptz,
 attempts integer NOT NULL DEFAULT 0 CHECK (attempts>=0),
 processed_at timestamptz,
 dead_letter_at timestamptz
);
CREATE INDEX outbox_pending ON outbox(available_at) WHERE processed_at IS NULL AND dead_letter_at IS NULL;
CREATE TABLE idempotency_keys (
 actor_id uuid NOT NULL REFERENCES users(id),
 route text NOT NULL,
 key text NOT NULL CHECK (char_length(key) BETWEEN 1 AND 128),
 request_hash bytea NOT NULL,
 status_code integer NOT NULL,
 response jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 expires_at timestamptz NOT NULL,
 PRIMARY KEY(actor_id,route,key)
);
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
