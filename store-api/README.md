# DCS-Gate test storefront

The site stays on GitHub Pages (`docs/`). New pages are `checkout.html`, `account.html`, and `store-terms.html`. The existing five services link to a test-only checkout; a separate subscription reproduces a one-day trial followed by USD12/month in Stripe test mode. No real service is provisioned or sold through these links.

The account flow is an actual Cloudflare Worker + D1 implementation. Stripe-hosted Payment Links collect billing name/address and payment details. The Worker never accepts card data and does not possess a Stripe API credential. It creates local orders, returns an allowlisted `buy.stripe.com/test_` link with an opaque order reference, and verifies signed Stripe webhook events. A return URL is not evidence of payment.

## Deployment state (2026-11-08, Tijuana)

The backend was deployed after account email verification. GET `/config` from the site origin returned the expected test-only catalog and Turnstile sitekey. The eight Node/SQLite tests pass. Frontend publication and end-to-end browser verification follow this deployment; consult the actual GitHub branch/deployment status for the latest state.

Created resources (identifiers are not secrets):

- Cloudflare account: `56b4af156206bd73ea1be1682ccc4122`
- Workers subdomain: `corekeeper-research`
- Intended Worker: `dcs-store`
- Intended endpoint: `https://dcs-store.corekeeper-research.workers.dev`
- D1: `dcs-store-test`, UUID `0c4e808f-cb47-42d1-abd5-ca354c059479`; schema already applied.
- Turnstile widget: `DCS-Gate account`, sitekey `0x4AAAAAAF5dEqAgxxMb-JgN`; hostname `diluidocognit.store`.
- Stripe test account: `acct_1KA4voJ6LGLQ4OOc` (livemode=false).
- Stripe test webhook: `we_1UZkCPJ6LGLQ4OOch07TPRG6`, destination `/stripe/webhook`, API version `2025-02-24.acacia`.
- Six test Payment Links: see `CATALOG` in `worker.mjs`. All require billing address collection. Automatic tax is disabled for this test-only flow; live tax configuration is not part of this change.

No secrets are committed. The initial deployment must bind:

| Binding | Type | Value |
| --- | --- | --- |
| DB | d1 | D1 UUID above |
| TURNSTILE_SITEKEY | plain_text | Sitekey above |
| TURNSTILE_SECRET | secret_text | Secret from the configured widget |
| STRIPE_WEBHOOK_SECRET | secret_text | Signing secret for the test endpoint |

For redeployment, upload `worker.mjs` as the ESM main module with compatibility date `2026-11-08`, enable its workers.dev subdomain, and enable observability logs/traces. Use the connected Cloudflare API, with secret values held in memory. If the original webhook signing secret is no longer accessible, rotate/recreate the test endpoint through Stripe and bind the replacement signing secret; do not fetch or print unrelated credentials. Verify GET `/config` with Origin `https://diluidocognit.store`, then check the full browser flow on the actual hostname before declaring completion.

## Verification

Run with Node 24+:

```sh
node --test store-api/worker.test.mjs
node --check store-api/worker.mjs
node --check docs/assets/store.js
git diff --check
```

Tests use SQLite for D1-like query semantics and Web Crypto. They check credential hashing, login/logout, expiration, account isolation, fixed server-side catalog, checkout retry/resume, CORS rejection, required Turnstile token, rate limits, HMAC signature/timestamp validation, webhook retry idempotency, mismatched amount rejection, trial completion, and rejection of live events. Turnstile's provider response is mocked only in unit tests; deployed code always calls Cloudflare Siteverify and checks hostname and action.

These are Node/SQLite tests, not Cloudflare-runtime or end-to-end browser tests. The cloud browser could not reach the local preview (`ERR_CONNECTION_REFUSED`). Actual Turnstile, Stripe Checkout completion, webhook delivery, D1 runtime behavior and mobile rendering still need deployment verification.

## Operational scope

- Session bearer tokens: random 256-bit values, SHA-256 digests in D1; sessionStorage in the browser, two-hour expiry. Close-tab logout is local; the server-side token expires after two hours or explicit logout.
- Passwords: PBKDF2-SHA256, random individual salt, 100,000 iterations (Workers-compatible Web Crypto); minimum 12 and maximum 128 characters.
- No email verification, password-reset delivery or automatic account deletion is implemented. The UI and conditions state that these are test accounts and recommend fictitious data and a unique password.
- Subscription cancellation is currently administered in the Stripe test Dashboard, on request; self-service cancellation is not implemented.
- Signup and checkout require Turnstile. Rate limits use atomic SQLite upserts. Cleanup of expired sessions and counters is sampled on requests.
- Structured logs contain only request ID, path, method, status and duration; no bodies, identity data, authorization values or card data.
- No automatic fulfillment, real-card checking, live authorization or live-mode switch exists.
