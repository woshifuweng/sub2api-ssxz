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

## Deployment evidence

- [ ] Recovery bundle checksums pass before mutation.
- [ ] Isolation verifier exits zero before staging starts.
- [ ] Staging health and dedicated-login smoke tests return 200.
- [ ] Staging sockets are loopback-only.
- [ ] Production PID/start timestamp are unchanged and public health returns 200.
- [ ] Startup logs contain no backup, update, probe, refresh, provider, or SMTP
      attempts.

The repository still lacks the broader unified-workspace security strategy
document referenced by the security review workflow. That separate document is
not silently inferred from this staging checklist.
