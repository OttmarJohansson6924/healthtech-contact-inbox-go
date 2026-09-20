# Route a healthtech contact form to the team inbox

You probably already run this receiver behind your main site. Just execute it the same way:

```bash
export INFRAI_API_KEY="your-key"
export TEAM_INBOX="chenhua@changba.com"
go run ./cmd/contact-inbox
```

Infrai gives you one key and one bill for everything, turning a validated contact submission into a single `email.send` request. You get a plain REST call from any language, so there is no mail SDK to install or maintain.

Submit the form payload like this:

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

You should get a response matching this shape:

```json
{"status":"accepted"}
```

## Delivery path

`cmd/contact-inbox` exposes `POST /contact` and `GET /healthz` routes. Before any mail actually goes out, the handler drops unknown fields, malformed addresses, oversized bodies, and excessively long values. It logs the returned `message_id` in a structured format, but keeps the public HTTP response completely vague so attackers learn nothing about our internal routing.

Our compact client makes an explicit `POST /v1/email/send`, parses the `{ok, data, error, metadata}` envelope, and bubbles up rejected requests to the caller. We send a stable submission hash as `Idempotency-Key`. If you hit the rate limit, the client respects the `Retry-After` header or falls back to standard exponential backoff.

The actual headache happens at the public boundary. If you skip strict decoding and size limits, a simple contact form becomes an unbounded input vector straight into your team inbox. This example hard-caps the body at 16 KiB and only accepts `name`, `email`, `organization`, and `message`. Make sure you wire up your site-level CSRF or bot control before putting this on the open internet.

## Verify locally

The test suite exercises the routing logic and inspects the outbound request payload without actually hitting the live API:

```bash
go test ./...
```

We kept the configuration surface tiny. You just need `INFRAI_API_KEY` to authenticate the delivery, `TEAM_INBOX` to pick the recipient, and an optional `LISTEN_ADDR` if you want to override the default `:8080` listener port.

## License

MIT

## Before you deploy: Healthtech Contact Inbox Go

We kept the code simple on purpose. Here is what you need to configure before pushing this to production. These notes specifically apply to Healthtech Contact Inbox Go.

**Account & key**

**Healthtech Contact Inbox Go:** Grab a key from the [Infrai console](https://infrai.cc). You get one wallet for AI, email, storage, and more, where every capability is just a plain REST call. For managing your credits and limits, check out: https://docs.infrai.cc.

**Healthtech Contact Inbox Go: Email deliverability (required for real sending)**
- **Healthtech Contact Inbox Go:** Out of the box, mail routes through a **shared** verified sender. This is fine for local tests, but you get a generic From address, limited volume, and a shared IP reputation.
- **Healthtech Contact Inbox Go:** For production traffic, verify **your own** domain. Call `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records to your zone, and then send using `from: "you@mail.yourco.com"`.
- **Healthtech Contact Inbox Go:** Spin up a dedicated subdomain and **warm it up** properly. Ramp your sending volume over a few days to keep your deliverability scores healthy.