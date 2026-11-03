CREATE UNIQUE INDEX outbox_plan_reminder_start
 ON outbox (plan_id, (payload->>'starts_at'))
 WHERE kind = 'plan.reminder' AND payload->>'schedule_key' = 'start';
