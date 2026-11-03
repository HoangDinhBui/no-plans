package push

import (
	"context"
	"errors"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/store"
	"github.com/jackc/pgx/v5"
)

type Transport interface {
	Send(context.Context, []Message) ([]Result, error)
	Receipts(context.Context, []string) (map[string]Result, error)
}

// ProcessNext expands one outbox event and processes one due device delivery.
func ProcessNext(ctx context.Context, db *store.Store, transport Transport) (bool, error) {
	events, err := db.ClaimOutbox(ctx, 1, 8, time.Minute)
	if err != nil {
		return false, err
	}
	for _, event := range events {
		if _, err := db.ExpandPushEvent(ctx, event); err != nil {
			return false, err
		}
	}
	delivery, err := db.ClaimPushDelivery(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return len(events) > 0, nil
	}
	if err != nil {
		return false, err
	}

	ticket, failure, delay := processDelivery(ctx, transport, delivery)
	_, err = db.ResolvePushDelivery(ctx, delivery, ticket, failure, delay)
	return true, err
}

func processDelivery(ctx context.Context, transport Transport, delivery store.PushDelivery) (string, string, *time.Duration) {
	retry := time.Minute
	if delivery.Attempt > 8 {
		return "", "AttemptsExhausted", nil
	}
	if delivery.TicketID != "" {
		retry = 5 * time.Minute
		receipts, err := transport.Receipts(ctx, []string{delivery.TicketID})
		if err != nil {
			return "", "ReceiptRequestFailed", &retry
		}
		receipt, ready := receipts[delivery.TicketID]
		if !ready {
			return "", "", &retry
		}
		if receipt.Status == "ok" {
			return "", "", nil
		}
		return "", receipt.Details.Error, nil
	}

	tickets, err := transport.Send(ctx, []Message{{
		To: delivery.Token, Title: "NO PLANS", Body: "Bạn có thông báo mới",
		Data: map[string]string{"eventId": delivery.EventID, "planId": delivery.PlanID, "recipientId": delivery.RecipientID},
	}})
	if err != nil || len(tickets) != 1 {
		return "", "SendRequestFailed", &retry
	}
	ticket := tickets[0]
	if ticket.Status == "ok" {
		receiptDelay := 15 * time.Minute
		return ticket.ID, "", &receiptDelay
	}
	if ticket.Details.Error == "MessageRateExceeded" {
		return "", ticket.Details.Error, &retry
	}
	return "", ticket.Details.Error, nil
}
