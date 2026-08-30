# Launch security checklist

## Environment isolation

- [ ] Production and staging database names differ.
- [ ] Production and staging Redis ports/processes differ.
- [ ] Staging has an OS-enforced outbound network deny rule.
- [ ] Staging background jobs are explicitly disabled and verified in the
      running process environment.

## Credentials and customer data

- [ ] API key overlap is zero.
- [ ] Password-hash overlap is zero.
- [ ] Provider account credentials and proxy credentials are absent.
- [ ] OAuth, passkey, pending login, SMTP, payment, webhook, object-storage, and
      monitoring credentials are absent or disabled. Only normal email
      identities owned by the two dedicated staging users may be created after
      a login smoke test.
- [ ] Customer email, IP, message, prompt, media, and free-form audit content is
      redacted.

## Billing and routing

- [ ] No staging account is schedulable.
- [ ] Only dedicated staging users can own active staging API keys.
- [ ] Provider, image, text, and web-search kill switches are active.
- [ ] No paid request is used for a staging smoke test.
- [ ] Billing preflight uses the same bounded output limit, model price, and
      customer group multiplier as the forwarded request.
- [ ] Payment order creation and reseller conversion retries are idempotent.
- [ ] A usage-settlement shortfall creates a durable ledger event and alert.

## Sensitive operations

- [ ] Step-up 2FA is enabled and verified for balance changes, refunds, payment
      configuration/providers/plans, affiliate transfers, redeem generation,
      reseller withdrawals, and system settings.
- [ ] Admin API-key authentication cannot satisfy a step-up challenge.
- [ ] Public payment lookups are body-limited and IP-rate-limited; webhook body
      limits reject oversized input instead of truncating and acknowledging it.

## Deployment evidence

- [ ] Recovery bundle checksums pass before mutation.
- [ ] Isolation verifier exits zero before staging starts.
- [ ] Staging health and dedicated-login smoke tests return 200.
- [ ] Staging sockets are loopback-only.
- [ ] Production PID/start timestamp are unchanged and public health returns 200.
- [ ] Startup logs contain no backup, update, probe, refresh, provider, or SMTP
      attempts.

The executable business journeys and workspace trust boundaries are defined in
`business-regression-matrix.md` and
`unified-chat-workspace-v1-security-strategy.md`.
