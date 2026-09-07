package paymentnotify

import (
	"fmt"
	"strings"
)

type PaymentEvent struct {
	EventID        string `json:"event_id"`
	AccountID      string `json:"account_id"`
	AmountMinor    int64  `json:"amount_minor"`
	Currency       string `json:"currency"`
	Merchant       string `json:"merchant"`
	RiskScore      int    `json:"risk_score"`
	CardNotPresent bool   `json:"card_not_present"`
}

type Notification struct {
	NotificationID string   `json:"notification_id"`
	PaymentEventID string   `json:"payment_event_id"`
	Severity       string   `json:"severity"`
	Title          string   `json:"title"`
	Message        string   `json:"message"`
	Actions        []string `json:"actions"`
}

func NotificationFor(event PaymentEvent) (Notification, error) {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.AccountID) == "" {
		return Notification{}, fmt.Errorf("event_id and account_id are required")
	}
	if event.AmountMinor <= 0 || len(event.Currency) != 3 || event.RiskScore < 0 || event.RiskScore > 100 {
		return Notification{}, fmt.Errorf("amount_minor, currency, or risk_score is invalid")
	}

	n := Notification{
		NotificationID: "payment:" + event.EventID,
		PaymentEventID: event.EventID,
		Severity:       "info",
		Title:          "Payment completed",
		Message:        fmt.Sprintf("%s %d at %s", strings.ToUpper(event.Currency), event.AmountMinor, event.Merchant),
		Actions:        []string{"view_receipt"},
	}

	if event.RiskScore >= 80 {
		n.Severity = "critical"
		n.Title = "Payment blocked"
		n.Actions = []string{"secure_account", "contact_support"}
	} else if event.RiskScore >= 50 || (event.CardNotPresent && event.AmountMinor >= 100000) {
		n.Severity = "warning"
		n.Title = "Review this payment"
		n.Actions = []string{"confirm_payment", "report_payment"}
	}

	return n, nil
}
