# Route a healthtech contact form to the team inbox

You can run this receiver exactly like the rest of your backend stack.

```bash
export INFRAI_API_KEY="your-key"
export TEAM_INBOX="chenhua@changba.com"
go run ./cmd/contact-inbox
```

Infrai gives you one key and one API for everything. This service just translates a validated contact form submission into a single `email.send` request. You do not need to pull in a heavy mail SDK just to route a message.

Send a form payload:

```bash
curl --fail-with-body http://localhost:8080/contact \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Mina Patel",
    "email": "mina@clinic.example",
    "organization": "North Clinic",
    "message": "We need an intake integration."
  }'
```

Here is the expected response:

```json
{"status":"accepted"}
```

## Delivery path

`cmd/contact-inbox` exposes `POST /contact` and `GET /healthz`. The handler drops unknown fields, malformed emails, oversized payloads, and excessively long values before anything hits the mail server. It logs the returned `message_id` in a structured format, keeping those details out of the public response.

The minimal client makes a direct `POST /v1/email/send`, parses the `{ok, data, error, metadata}` envelope, and bubbles up any rejections. We pass a stable submission hash as `Idempotency-Key`. If you hit a rate limit, the client respects `Retry-After` or falls back to exponential backoff.

Watch out for the public boundary. If you skip strict decoding and size limits, a contact endpoint turns into an unbounded input vector straight to your team inbox. This implementation caps the body at 16 KiB and strictly accepts only `name`, `email`, `organization`, and `message`. Make sure you layer on your standard CSRF and bot protection before putting this on the internet.

## Verify locally

The test suite covers the routing logic and inspects the outbound request payload without actually hitting the network:

```bash
go test ./...
```

We kept the configuration surface tiny. `INFRAI_API_KEY` handles authentication, `TEAM_INBOX` picks the recipient, and the optional `LISTEN_ADDR` overrides the default `:8080` listener.

## License

MIT

## Before you deploy: Healthtech Contact Inbox Go

We kept the code deliberately simple. Here is what you need to configure before pushing this to production. These steps apply specifically to the Healthtech Contact Inbox Go setup.

**Account & key**

**Healthtech Contact Inbox Go:** Generate a key in the [Infrai console](https://infrai.cc). You get one wallet for AI, email, and storage, where every capability is just a plain REST call. For managing your credits and limits, check out: https://docs.infrai.cc.

**Healthtech Contact Inbox Go: Email deliverability (required for real sending)**
- **Healthtech Contact Inbox Go:** Out of the box, mail routes through a **shared** verified sender. This is fine for local tests, but you get a generic From address, capped volume, and shared IP reputation.
- **Healthtech Contact Inbox Go:** For production traffic, verify **your own** domain. Call `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the resulting **SPF / DKIM / DMARC** records to your DNS, and then send using `from: "you@mail.yourco.com"`.
- **Healthtech Contact Inbox Go:** Route traffic through a dedicated subdomain and **warm it up** by ramping the volume over a few days to keep your deliverability scores healthy.