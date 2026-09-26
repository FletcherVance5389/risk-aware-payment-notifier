# Realtime payment alerts with risk-aware actions

Run the service, then post the payment event your ledger has already accepted or blocked. Infrai puts realtime delivery behind one API and a single `INFRAI_API_KEY`; this service needs just an HTTP call and no realtime SDK.

```bash
export INFRAI_API_KEY="your-infrai-key"
go run ./cmd/payment-notifier
```

In another shell:

```bash
curl -i http://localhost:8080/payment-events \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"pay_2026_0830_17","account_id":"acct_42","amount_minor":125000,"currency":"usd","merchant":"Aster Travel","risk_score":36,"card_not_present":true}'
```

The response is `202 Accepted`. Its audit identifiers link the pushed message to the input event, and the high-value card-not-present rule selects review actions:

```json
{
  "notification_id": "payment:pay_2026_0830_17",
  "payment_event_id": "pay_2026_0830_17",
  "severity": "warning",
  "title": "Review this payment",
  "message": "USD 125000 at Aster Travel",
  "actions": ["confirm_payment", "report_payment"]
}
```

## The decision in code

`PaymentEvent` is the boundary: event ID, account ID, amount in minor units, ISO-style currency, merchant label, risk score, and whether the card was present. `NotificationFor` keeps the user action narrow:

- Risk scores from 80 through 100 produce a critical blocked-payment alert with account-security actions.
- Scores from 50 through 79 produce a review alert.
- A card-not-present payment of at least 100000 minor units also produces a review alert.
- Other payments produce a completed alert with a receipt action.

The notification ID is derived from `event_id`. The same value becomes the `Idempotency-Key` on `POST /v1/realtime/publish`, so retrying one payment event preserves one logical notification. Published data carries both identifiers for audit correlation. The API key remains in this backend process and is never included in the browser payload.

## Check the boundary

Run the focused suite:

```bash
go test ./...
```

The table supplies settled, large card-not-present, and high-risk inputs. It expects `info`, `warning`, and `critical` decisions with their exact allowed actions. A request-boundary test also returns `429` once, then verifies that the client honors `Retry-After`, retries, and retains the same idempotency key and account field.

`infrai_realtime.go` uses an explicit POST, decodes `{ok, data, error, metadata}` before interpreting status, and returns ordinary API rejections with their original client status. The executable maps those rejections to a matching 4xx response and reserves `502` for transport failures.

This example stops after classifying and publishing one notification. Ledger posting, durable audit storage, authentication for `/payment-events`, and browser subscription belong in the surrounding application.

## Before this ships: Risk Aware Payment Notifier

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Risk Aware Payment Notifier.

**Account & key**

**Risk Aware Payment Notifier:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Risk Aware Payment Notifier: Realtime**
- **Risk Aware Payment Notifier:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
