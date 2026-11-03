CREATE UNIQUE INDEX outbox_plan_reminder_version
 ON outbox (plan_id, (payload->>'version'))
 WHERE kind = 'plan.reminder';

CREATE INDEX plans_confirmed_reminder_window ON plans (starts_at)
 WHERE status = 'confirmed';
