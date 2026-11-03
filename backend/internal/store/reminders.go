package store

import "context"

// QueuePlanReminders records one reminder per confirmed plan start time.
func (s *Store) QueuePlanReminders(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
 INSERT INTO outbox(plan_id, actor_id, kind, payload)
 SELECT id, owner_id, 'plan.reminder',
        jsonb_build_object('schedule_key', 'start', 'starts_at', starts_at AT TIME ZONE 'UTC')
 FROM plans p
 WHERE status = 'confirmed' AND starts_at > now()
   AND starts_at <= now() + interval '1 hour'
   AND NOT EXISTS (
    SELECT 1 FROM outbox e
    WHERE e.plan_id = p.id AND e.kind = 'plan.reminder'
      AND e.payload->>'schedule_key' IS NULL
      AND (e.payload->>'starts_at')::timestamptz = p.starts_at
   )
 ON CONFLICT DO NOTHING
 `)
	return err
}

// CurrentPlanReminder checks the schedule and RSVP before a device send.
func (s *Store) CurrentPlanReminder(ctx context.Context, delivery PushDelivery) (bool, error) {
	var current bool
	err := s.db.QueryRow(ctx, `
 SELECT EXISTS (
  SELECT 1 FROM outbox e
  JOIN plans p ON p.id = e.plan_id
  JOIN plan_members m ON m.plan_id = p.id AND m.user_id = $2
  WHERE e.id = $1 AND p.status = 'confirmed' AND p.starts_at > now()
    AND p.starts_at = CASE WHEN e.payload->>'schedule_key' = 'start'
      THEN (e.payload->>'starts_at')::timestamp AT TIME ZONE 'UTC'
      ELSE (e.payload->>'starts_at')::timestamptz END
    AND m.response = 'going'
 )`, delivery.EventID, delivery.RecipientID).Scan(&current)
	return current, err
}
