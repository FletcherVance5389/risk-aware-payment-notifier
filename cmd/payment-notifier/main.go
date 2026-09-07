package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	notify "example.com/fintech-payment-notifier"
)

func main() {
	client, err := notify.NewRealtimeClient(os.Getenv("INFRAI_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /payment-events", func(w http.ResponseWriter, r *http.Request) {
		var event notify.PaymentEvent
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&event); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payment event"})
			return
		}
		notification, err := notify.NotificationFor(event)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		if err := client.Publish(r.Context(), event, notification); err != nil {
			var apiErr *notify.InfraiError
			if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
				writeJSON(w, apiErr.Status, map[string]string{"error": apiErr.Error()})
				return
			}
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "notification delivery failed"})
			return
		}
		writeJSON(w, http.StatusAccepted, notification)
	})

	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}
	log.Printf("payment notifier listening on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
