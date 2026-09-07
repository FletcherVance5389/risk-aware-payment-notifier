package paymentnotify

import (
	"reflect"
	"testing"
)

func TestNotificationForRiskDecision(t *testing.T) {
	tests := []struct {
		name     string
		event    PaymentEvent
		severity string
		title    string
		actions  []string
	}{
		{
			name:     "settled payment offers receipt",
			event:    PaymentEvent{EventID: "evt-1", AccountID: "acct-7", AmountMinor: 2599, Currency: "usd", Merchant: "Metro", RiskScore: 12},
			severity: "info", title: "Payment completed", actions: []string{"view_receipt"},
		},
		{
			name:     "large card-not-present payment asks for review",
			event:    PaymentEvent{EventID: "evt-2", AccountID: "acct-7", AmountMinor: 100000, Currency: "usd", Merchant: "Aster", RiskScore: 30, CardNotPresent: true},
			severity: "warning", title: "Review this payment", actions: []string{"confirm_payment", "report_payment"},
		},
		{
			name:     "high risk payment restricts actions",
			event:    PaymentEvent{EventID: "evt-3", AccountID: "acct-7", AmountMinor: 4000, Currency: "usd", Merchant: "North", RiskScore: 84},
			severity: "critical", title: "Payment blocked", actions: []string{"secure_account", "contact_support"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NotificationFor(tt.event)
			if err != nil {
				t.Fatal(err)
			}
			if got.Severity != tt.severity || got.Title != tt.title || !reflect.DeepEqual(got.Actions, tt.actions) {
				t.Fatalf("decision = (%q, %q, %v), want (%q, %q, %v)", got.Severity, got.Title, got.Actions, tt.severity, tt.title, tt.actions)
			}
			if got.NotificationID != "payment:"+tt.event.EventID || got.PaymentEventID != tt.event.EventID {
				t.Fatalf("audit identifiers = %#v", got)
			}
		})
	}
}
