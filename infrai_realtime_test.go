package paymentnotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPublishRetries429WithSameBoundary(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != realtimePublishPath {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") != "payment:evt-9" {
			t.Fatalf("idempotency key = %q", r.Header.Get("Idempotency-Key"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["account_id"] != "acct-2" {
			t.Fatalf("account_id = %#v", body["account_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"message":"retry later"},"metadata":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{},"metadata":{}}`))
	}))
	defer server.Close()

	client := &RealtimeClient{APIKey: "test-key", BaseURL: server.URL, HTTP: server.Client(), sleep: func(context.Context, time.Duration) error { return nil }}
	event := PaymentEvent{EventID: "evt-9", AccountID: "acct-2"}
	notification := Notification{NotificationID: "payment:evt-9", PaymentEventID: "evt-9"}
	if err := client.Publish(context.Background(), event, notification); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}
