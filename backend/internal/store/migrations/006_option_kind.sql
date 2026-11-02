ALTER TABLE place_options ADD COLUMN kind text NOT NULL DEFAULT 'place' CHECK(kind IN ('place','idea'));
UPDATE place_options o SET kind=p.kind FROM places p WHERE o.place_id=p.id;
