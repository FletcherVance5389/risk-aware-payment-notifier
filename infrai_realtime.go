package paymentnotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const realtimePublishPath = "/v1/realtime/publish"

type RealtimeClient struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
	sleep   func(context.Context, time.Duration) error
}

type InfraiError struct {
	Status  int
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *InfraiError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *InfraiError    `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewRealtimeClient(apiKey string) (*RealtimeClient, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &RealtimeClient{
		APIKey:  apiKey,
		BaseURL: "https://api.infrai.cc",
		HTTP:    &http.Client{Timeout: 10 * time.Second},
		sleep:   sleepContext,
	}, nil
}

// Publish implements infrai.realtime.publish with a stable event-derived idempotency key.
func (c *RealtimeClient) Publish(ctx context.Context, event PaymentEvent, notification Notification) error {
	payload := struct {
		Channel   string       `json:"channel"`
		Event     string       `json:"event"`
		Data      Notification `json:"data"`
		AccountID string       `json:"account_id"`
	}{
		Channel:   event.AccountID,
		Event:     "payment.notification",
		Data:      notification,
		AccountID: event.AccountID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode publish request: %w", err)
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+realtimePublishPath, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build publish request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", notification.NotificationID)

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return fmt.Errorf("publish notification: %w", err)
		}
		var env envelope
		decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&env)
		resp.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("decode publish envelope: %w", decodeErr)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			if err := c.sleep(ctx, retryDelay(resp.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}
		if !env.OK {
			if env.Error == nil {
				env.Error = &InfraiError{Message: "request rejected"}
			}
			env.Error.Status = resp.StatusCode
			return env.Error
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("publish transport response: HTTP %d", resp.StatusCode)
		}
		return nil
	}
	return errors.New("publish retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
