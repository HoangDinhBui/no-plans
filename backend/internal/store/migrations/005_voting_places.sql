CREATE TABLE places (
 id uuid PRIMARY KEY,
 name text NOT NULL CHECK(char_length(name) BETWEEN 1 AND 120),
 activity text NOT NULL CHECK(activity IN ('food','coffee','drinks','cinema','sports','other')),
 kind text NOT NULL CHECK(kind IN ('place','idea')),
 area text NOT NULL CHECK(char_length(area) BETWEEN 1 AND 80),
 address text NOT NULL DEFAULT '' CHECK(char_length(address)<=500),
 description text NOT NULL DEFAULT '' CHECK(char_length(description)<=1000),
 budget_min integer,
 budget_max integer,
 price_note text NOT NULL DEFAULT '' CHECK(char_length(price_note)<=200),
 people_min integer NOT NULL DEFAULT 1 CHECK(people_min BETWEEN 1 AND 1000),
 people_max integer NOT NULL DEFAULT 0 CHECK(people_max=0 OR people_max>=people_min),
 source_url text NOT NULL DEFAULT '',
 source_checked_at timestamptz,
 active boolean NOT NULL DEFAULT true,
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK((budget_min IS NULL AND budget_max IS NULL) OR (budget_min IS NOT NULL AND budget_max IS NOT NULL AND budget_min>=0 AND budget_max>=budget_min))
);
CREATE INDEX places_activity_area ON places(activity,area) WHERE active;
CREATE TABLE place_options (
 id uuid PRIMARY KEY,
 plan_id uuid NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
 created_by uuid NOT NULL REFERENCES users(id),
 place_id uuid REFERENCES places(id),
 name text NOT NULL CHECK(char_length(name) BETWEEN 1 AND 120),
 address text NOT NULL DEFAULT '' CHECK(char_length(name)+char_length(address)+3<=500),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(plan_id,id)
);
ALTER TABLE plans ADD COLUMN selected_option_id uuid;
ALTER TABLE plans ADD CONSTRAINT plan_selected_option FOREIGN KEY(id,selected_option_id) REFERENCES place_options(plan_id,id) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE votes (
 plan_id uuid NOT NULL,
 user_id uuid NOT NULL,
 option_id uuid NOT NULL,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(plan_id,user_id),
 FOREIGN KEY(plan_id,user_id) REFERENCES plan_members(plan_id,user_id) ON DELETE CASCADE,
 FOREIGN KEY(plan_id,option_id) REFERENCES place_options(plan_id,id) ON DELETE CASCADE
);
CREATE INDEX votes_option ON votes(plan_id,option_id);
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
